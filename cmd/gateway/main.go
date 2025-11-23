package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/auth"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/controlplane"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/policy"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
	anthropic_provider "github.com/tjfontaine/polyglot-llm-gateway/internal/provider/anthropic"
	openai_provider "github.com/tjfontaine/polyglot-llm-gateway/internal/provider/openai"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/memory"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/sqlite"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/telemetry"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tenant"
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
	shutdown, err := telemetry.InitTracer("polyglot-llm-gateway", logger)
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

	// Initialize storage if configured
	var store storage.ConversationStore
	if cfg.Storage.Type != "" && cfg.Storage.Type != "none" {
		switch cfg.Storage.Type {
		case "sqlite":
			dbPath := cfg.Storage.SQLite.Path
			if dbPath == "" {
				dbPath = "./data/conversations.db"
			}
			store, err = sqlite.New(dbPath)
			if err != nil {
				log.Fatalf("Failed to initialize SQLite storage: %v", err)
			}
			defer store.Close()
			logger.Info("storage initialized", slog.String("type", "sqlite"), slog.String("path", dbPath))
		case "memory":
			store = memory.New()
			logger.Info("storage initialized", slog.String("type", "memory"))
		default:
			log.Fatalf("Unknown storage type: %s", cfg.Storage.Type)
		}
	}

	// Initialize Provider Registry
	providerRegistry := provider.NewRegistry()

	// Multi-tenant mode or single-tenant mode
	var router domain.Provider
	var authenticator *auth.Authenticator
	var providers map[string]domain.Provider
	var tenants []*tenant.Tenant

	if len(cfg.Tenants) > 0 {
		// Multi-tenant mode
		logger.Info("multi-tenant mode enabled", slog.Int("tenant_count", len(cfg.Tenants)))

		tenantRegistry := tenant.NewRegistry()
		tenants, err = tenantRegistry.LoadTenants(cfg.Tenants, providerRegistry)
		if err != nil {
			log.Fatalf("Failed to load tenants: %v", err)
		}

		// Create authenticator
		authenticator = auth.NewAuthenticator(tenants)

		// Use first tenant's router as default (will be overridden per-request)
		if len(tenants) > 0 {
			router = policy.NewRouter(tenants[0].Providers, tenants[0].Routing)
		}
	} else {
		// Single-tenant mode (backwards compatible)
		logger.Info("single-tenant mode (no authentication)")

		if len(cfg.Providers) > 0 {
			providers, err = providerRegistry.CreateProviders(cfg.Providers)
			if err != nil {
				log.Fatalf("Failed to create providers: %v", err)
			}
		} else {
			// Fallback to legacy env-based config
			logger.Info("using legacy env-based provider setup")
			openaiP := openai_provider.New(cfg.OpenAI.APIKey)
			anthropicP := anthropic_provider.New(cfg.Anthropic.APIKey)
			providers = map[string]domain.Provider{
				"openai":    openaiP,
				"anthropic": anthropicP,
			}
			// Populate provider config for control plane display while keeping secrets empty
			cfg.Providers = []config.ProviderConfig{
				{Name: "openai", Type: "openai"},
				{Name: "anthropic", Type: "anthropic"},
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

		router = policy.NewRouter(providers, cfg.Routing)
	}

	// Initialize Frontdoor Registry
	frontdoorRegistry := frontdoor.NewRegistry()

	// Create frontdoor handlers from config
	var handlerRegs []frontdoor.HandlerRegistration
	var apps []config.AppConfig
	if len(cfg.Apps) > 0 {
		apps = cfg.Apps
	} else if len(cfg.Frontdoors) > 0 {
		for _, fd := range cfg.Frontdoors {
			apps = append(apps, config.AppConfig{
				Name:         fd.Type,
				Frontdoor:    fd.Type,
				Path:         fd.Path,
				Provider:     fd.Provider,
				DefaultModel: fd.DefaultModel,
			})
		}
	} else {
		// Default frontdoors if no config
		logger.Info("using default frontdoor configuration")
		apps = []config.AppConfig{
			{Name: "openai", Frontdoor: "openai", Path: "/openai"},
			{Name: "anthropic", Frontdoor: "anthropic", Path: "/anthropic"},
		}
	}

	handlerRegs, err = frontdoorRegistry.CreateHandlers(apps, router, providers, store)
	if err != nil {
		log.Fatalf("Failed to create frontdoor handlers: %v", err)
	}

	// Initialize Server
	srv := server.New(cfg.Server.Port, logger, authenticator)

	// Register all frontdoor handlers
	for _, reg := range handlerRegs {
		method := reg.Method
		if method == "" {
			method = http.MethodPost
		}

		switch method {
		case http.MethodGet:
			srv.Router.Get(reg.Path, reg.Handler)
		case http.MethodPost:
			srv.Router.Post(reg.Path, reg.Handler)
		default:
			srv.Router.Method(method, reg.Path, http.HandlerFunc(reg.Handler))
		}

		log.Printf("Registered %s %s", method, reg.Path)
	}

	// Register Responses API handlers if storage is configured
	if store != nil {
		responsesHandlers := frontdoorRegistry.CreateResponsesHandlers("/responses", store, router)
		for _, reg := range responsesHandlers {
			method := reg.Method
			if method == "" {
				method = http.MethodPost
			}

			switch method {
			case http.MethodGet:
				srv.Router.Get(reg.Path, reg.Handler)
			case http.MethodPost:
				srv.Router.Post(reg.Path, reg.Handler)
			default:
				srv.Router.Method(method, reg.Path, http.HandlerFunc(reg.Handler))
			}

			log.Printf("Registered Responses API: %s %s", method, reg.Path)
		}
	}

	// Initialize Control Plane
	cpServer := controlplane.NewServer(cfg, store, tenants)
	srv.Router.Mount("/admin", cpServer)
	log.Printf("Registered Control Plane at /admin")

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
