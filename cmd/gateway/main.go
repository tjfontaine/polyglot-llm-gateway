package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
	"github.com/tjfontaine/poly-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/poly-llm-gateway/internal/policy"
	"github.com/tjfontaine/poly-llm-gateway/internal/provider"
	anthropic_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/anthropic"
	openai_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/openai"
	"github.com/tjfontaine/poly-llm-gateway/internal/server"
	"github.com/tjfontaine/poly-llm-gateway/internal/telemetry"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Initialize OpenTelemetry
	shutdown, err := telemetry.InitTracer("poly-llm-gateway", logger)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown tracer", slog.String("error", err.Error()))
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Provider Registry
	registry := provider.NewRegistry()

	// Create providers from config
	var providers map[string]domain.Provider
	if len(cfg.Providers) > 0 {
		providers, err = registry.CreateProviders(cfg.Providers)
		if err != nil {
			log.Fatalf("Failed to create providers: %v", err)
		}
	} else {
		// Fallback to legacy env-based config for backwards compatibility
		log.Println("No providers in config, using legacy env-based setup")
		openaiP := openai_provider.New(cfg.OpenAI.APIKey)
		anthropicP := anthropic_provider.New(cfg.Anthropic.APIKey)
		providers = map[string]domain.Provider{
			"openai":    openaiP,
			"anthropic": anthropicP,
		}
		// Use default routing if no config
		cfg.Routing = config.RoutingConfig{
			Rules: []config.RoutingRule{
				{ModelPrefix: "claude", Provider: "anthropic"},
				{ModelPrefix: "gpt", Provider: "openai"},
			},
			DefaultProvider: "openai",
		}
	}

	// Initialize Router with config-based routing
	router := policy.NewRouter(providers, cfg.Routing)

	// Initialize Frontdoor Registry
	frontdoorRegistry := frontdoor.NewRegistry()

	// Create frontdoor handlers from config
	var handlerRegs []frontdoor.HandlerRegistration
	if len(cfg.Frontdoors) > 0 {
		handlerRegs, err = frontdoorRegistry.CreateHandlers(cfg.Frontdoors, router)
		if err != nil {
			log.Fatalf("Failed to create frontdoor handlers: %v", err)
		}
	} else {
		// Default frontdoors if no config
		log.Println("No frontdoors in config, using defaults")
		cfg.Frontdoors = []config.FrontdoorConfig{
			{Type: "openai", Path: "/openai"},
			{Type: "anthropic", Path: "/anthropic"},
		}
		handlerRegs, err = frontdoorRegistry.CreateHandlers(cfg.Frontdoors, router)
		if err != nil {
			log.Fatalf("Failed to create default frontdoor handlers: %v", err)
		}
	}

	// Initialize Server
	srv := server.New(cfg.Server.Port, logger)

	// Register all frontdoor handlers
	for _, reg := range handlerRegs {
		srv.Router.Post(reg.Path, reg.Handler)
		log.Printf("Registered %s", reg.Path)
	}

	log.Printf("Starting server on port %d", cfg.Server.Port)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutting down gracefully...")
}
