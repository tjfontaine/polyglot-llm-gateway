package main

import (
	"log"

	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
	openai_frontdoor "github.com/tjfontaine/poly-llm-gateway/internal/frontdoor/openai"
	"github.com/tjfontaine/poly-llm-gateway/internal/policy"
	"github.com/tjfontaine/poly-llm-gateway/internal/provider"
	anthropic_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/anthropic"
	openai_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/openai"
	"github.com/tjfontaine/poly-llm-gateway/internal/server"
)

func main() {
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

	// Initialize Frontdoor with Router
	handler := openai_frontdoor.NewHandler(router)

	// Initialize Server
	srv := server.New(cfg.Server.Port)

	// Register Routes
	srv.Router.Post("/v1/chat/completions", handler.HandleChatCompletion)

	log.Printf("Starting server on port %d", cfg.Server.Port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
