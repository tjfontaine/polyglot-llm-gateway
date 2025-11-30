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
	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/controlplane"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/middleware/tracer"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/auth"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/tenant"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/registration"
	routerpkg "github.com/tjfontaine/polyglot-llm-gateway/internal/router"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/memory"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/sqlite"

	// Import consolidated packages for legacy provider creation
	anthropic "github.com/tjfontaine/polyglot-llm-gateway/internal/backend/anthropic"
	openai "github.com/tjfontaine/polyglot-llm-gateway/internal/backend/openai"
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
	shutdown, err := tracer.InitTracer("polyglot-llm-gateway", logger)
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

	registration.RegisterBuiltins()

	// Initialize storage if configured
	var store storage.ConversationStore
	var threadStore storage.ThreadStateStore
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
			if ts, ok := store.(storage.ThreadStateStore); ok {
				threadStore = ts
			}
		case "memory":
			store = memory.New()
			logger.Info("storage initialized", slog.String("type", "memory"))
			if ts, ok := store.(storage.ThreadStateStore); ok {
				threadStore = ts
			}
		default:
			log.Fatalf("Unknown storage type: %s", cfg.Storage.Type)
		}
	}

	// Initialize Provider Registry
	providerRegistry := provider.NewRegistry()

	// Multi-tenant mode or single-tenant mode
	var router ports.Provider
	var authenticator *auth.Authenticator
	var providers map[string]ports.Provider
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
			router = routerpkg.NewProviderRouter(tenants[0].Providers, tenants[0].Routing)
		}

		if threadStore != nil {
			for _, tenantCfg := range cfg.Tenants {
				if t, ok := tenantRegistry.GetTenant(tenantCfg.ID); ok {
					attachThreadStore(threadStore, t.Providers, tenantCfg.Providers)
				}
			}
		}
	} else {
		// Single-tenant mode (backwards compatible)
		logger.Info("single-tenant mode (no authentication)")

		if len(cfg.Providers) > 0 {
			providers, err = providerRegistry.CreateProviders(cfg.Providers)
			if err != nil {
				log.Fatalf("Failed to create providers: %v", err)
			}
			if threadStore != nil {
				attachThreadStore(threadStore, providers, cfg.Providers)
			}
		} else {
			// Fallback to legacy env-based config
			logger.Info("using legacy env-based provider setup")
			openaiP := openai.NewProvider(cfg.OpenAI.APIKey)
			anthropicP := anthropic.NewProvider(cfg.Anthropic.APIKey)
			providers = map[string]ports.Provider{
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

		router = routerpkg.NewProviderRouter(providers, cfg.Routing)
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
				Name:            fd.Type,
				Frontdoor:       fd.Type,
				Path:            fd.Path,
				Provider:        fd.Provider,
				DefaultModel:    fd.DefaultModel,
				EnableResponses: fd.EnableResponses,
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
		basePaths := resolveResponsesBasePaths(apps, cfg.Routing)

		for _, base := range basePaths {
			responsesHandlers := frontdoorRegistry.CreateResponsesHandlers(base, store, router)
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

type threadStoreSetter interface {
	SetThreadStore(storage.ThreadStateStore)
}

func attachThreadStore(store storage.ThreadStateStore, providers map[string]ports.Provider, configs []config.ProviderConfig) {
	if store == nil || len(providers) == 0 {
		return
	}

	for _, cfg := range configs {
		if !cfg.ResponsesThreadPersistence {
			continue
		}
		prov, ok := providers[cfg.Name]
		if !ok {
			continue
		}
		if setter, ok := prov.(threadStoreSetter); ok {
			setter.SetThreadStore(store)
		}
	}
}

func buildProviderResponsesSupport(cfg *config.Config) map[string]bool {
	// Deprecated: keep for compatibility with callers; now all providers can back Responses when enabled at the frontdoor.
	return map[string]bool{}
}

func resolveResponsesBasePaths(apps []config.AppConfig, routing config.RoutingConfig) []string {
	seen := make(map[string]bool)
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
	}

	for _, app := range apps {
		if app.Frontdoor == "openai" && app.EnableResponses {
			add(app.Path)
		}
	}

	var paths []string
	for path := range seen {
		paths = append(paths, path)
	}
	return paths
}
