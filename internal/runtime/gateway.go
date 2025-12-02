// Package runtime provides the core Gateway struct and lifecycle management
// for the extensible LLM gateway.
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/events/direct"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/policy/basic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/controlplane"
	apimw "github.com/tjfontaine/polyglot-llm-gateway/internal/api/middleware"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/openai"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/tenant"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
	routerpkg "github.com/tjfontaine/polyglot-llm-gateway/internal/router"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/shadow"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Gateway is the main entry point for running the LLM gateway.
// It manages configuration, providers, frontdoors, and HTTP server lifecycle.
// Gateway can be embedded in larger applications or run standalone.
type Gateway struct {
	// Dependencies (injected via options)
	config  ports.ConfigProvider
	auth    ports.AuthProvider
	storage ports.StorageProvider
	events  ports.EventPublisher
	policy  ports.QualityPolicy

	// Internal state
	providers map[string]ports.Provider
	handlers  []frontdoor.HandlerRegistration // Frontdoor handlers (HTTP routes)
	server    *http.Server
	logger    *slog.Logger

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// New creates a new Gateway with the given options.
// By default, uses file-based config, API key auth, and SQLite storage.
func New(opts ...Option) (*Gateway, error) {
	gw := &Gateway{
		logger:    slog.Default(),
		providers: make(map[string]ports.Provider),
		handlers:  make([]frontdoor.HandlerRegistration, 0),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(gw); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	// Validate required dependencies
	if gw.config == nil {
		return nil, fmt.Errorf("config provider required (use WithFileConfig or WithRemoteConfig)")
	}
	if gw.storage == nil {
		return nil, fmt.Errorf("storage provider required (use WithSQLite or WithPostgres)")
	}

	// Set defaults for optional dependencies
	if gw.auth == nil {
		gw.logger.Info("no auth provider specified, using no-auth mode (all requests allowed)")
		// No auth provider = all requests allowed (single-tenant mode)
	}
	if gw.events == nil {
		gw.logger.Info("no event publisher specified, using direct storage")
		if gw.storage != nil {
			publisher, err := direct.NewPublisher(gw.storage)
			if err != nil {
				return nil, fmt.Errorf("create default event publisher: %w", err)
			}
			gw.events = publisher
		}
	}
	if gw.policy == nil {
		gw.logger.Info("no quality policy specified, using basic policy (no rate limiting)")
		gw.policy = basic.NewPolicy()
	}

	return gw, nil
}

// Start initializes and starts the gateway.
func (g *Gateway) Start(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.ctx, g.cancel = context.WithCancel(ctx)

	// Load initial config
	cfg, err := g.config.Load(g.ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize providers from config
	if err := g.initProviders(cfg); err != nil {
		return fmt.Errorf("init providers: %w", err)
	}

	// Initialize frontdoors from config
	if err := g.initFrontdoors(cfg); err != nil {
		return fmt.Errorf("init frontdoors: %w", err)
	}

	// Start HTTP server
	if err := g.startServer(cfg); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	// Watch for config changes
	go g.watchConfig()

	g.logger.Info("gateway started",
		slog.Int("port", cfg.Server.Port),
		slog.Int("providers", len(g.providers)),
		slog.Int("apps", len(cfg.Apps)))

	return nil
}

// Shutdown gracefully stops the gateway.
func (g *Gateway) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.logger.Info("shutting down gateway")

	if g.cancel != nil {
		g.cancel()
	}

	// Stop HTTP server
	if g.server != nil {
		if err := g.server.Shutdown(ctx); err != nil {
			g.logger.Error("failed to shutdown server", slog.String("error", err.Error()))
			return err
		}
	}

	// Close resources
	if g.storage != nil {
		if err := g.storage.Close(); err != nil {
			g.logger.Error("failed to close storage", slog.String("error", err.Error()))
		}
	}

	if g.events != nil {
		if err := g.events.Close(); err != nil {
			g.logger.Error("failed to close events", slog.String("error", err.Error()))
		}
	}

	if g.config != nil {
		if err := g.config.Close(); err != nil {
			g.logger.Error("failed to close config", slog.String("error", err.Error()))
		}
	}

	g.logger.Info("gateway shutdown complete")
	return nil
}

// watchConfig watches for config changes and reloads.
func (g *Gateway) watchConfig() {
	onChange := func(newCfg *config.Config) {
		g.logger.Info("config changed, reloading")
		if err := g.reload(newCfg); err != nil {
			g.logger.Error("failed to reload", slog.String("error", err.Error()))
		}
	}

	if err := g.config.Watch(g.ctx, onChange); err != nil {
		if err != context.Canceled {
			g.logger.Error("config watch failed", slog.String("error", err.Error()))
		}
	}
}

// reload updates the gateway with new configuration.
func (g *Gateway) reload(cfg *config.Config) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Reinitialize providers
	if err := g.initProviders(cfg); err != nil {
		return fmt.Errorf("reinit providers: %w", err)
	}

	// Reinitialize frontdoors
	if err := g.initFrontdoors(cfg); err != nil {
		return fmt.Errorf("reinit frontdoors: %w", err)
	}

	g.logger.Info("reload complete",
		slog.Int("providers", len(g.providers)),
		slog.Int("apps", len(cfg.Apps)))

	return nil
}

// initProviders initializes providers from configuration.
func (g *Gateway) initProviders(cfg *config.Config) error {
	g.logger.Debug("initializing providers", slog.Int("count", len(cfg.Providers)))

	// Create provider registry
	providerRegistry := provider.NewRegistry()

	// Handle multi-tenant vs single-tenant modes
	if len(cfg.Tenants) > 0 {
		// Multi-tenant mode
		g.logger.Info("multi-tenant mode", slog.Int("tenant_count", len(cfg.Tenants)))

		tenantRegistry := tenant.NewRegistry()
		tenants, err := tenantRegistry.LoadTenants(cfg.Tenants, providerRegistry)
		if err != nil {
			return fmt.Errorf("load tenants: %w", err)
		}

		// Update auth provider with reloaded tenants if it supports reload
		if g.auth != nil {
			if reloader, ok := g.auth.(interface{ ReloadFromConfig(*config.Config) error }); ok {
				if err := reloader.ReloadFromConfig(cfg); err != nil {
					g.logger.Warn("failed to reload auth provider", slog.String("error", err.Error()))
				}
			}
		}

		// Use first tenant's providers as default (will be overridden per-request)
		if len(tenants) > 0 {
			g.providers = tenants[0].Providers
			// Create router for the first tenant as default
			router := routerpkg.NewProviderRouter(tenants[0].Providers, tenants[0].Routing)
			g.providers["_router"] = router // Store router with special key
		}

		// Attach thread/event stores to tenant providers
		if threadStore, ok := g.storage.(interface{ SetThreadState(string, string) error }); ok {
			for _, t := range tenants {
				attachThreadStore(threadStore, g.storage, t.Providers, cfg.Providers)
			}
		}

	} else {
		// Single-tenant mode (backwards compatible)
		g.logger.Info("single-tenant mode (no authentication)")

		var err error
		if len(cfg.Providers) > 0 {
			g.providers, err = providerRegistry.CreateProviders(cfg.Providers)
			if err != nil {
				return fmt.Errorf("create providers: %w", err)
			}
		} else {
			// Fallback to legacy env-based config
			g.logger.Info("using legacy env-based provider setup")
			openaiP := openai.NewProvider(cfg.OpenAI.APIKey)
			anthropicP := anthropic.NewProvider(cfg.Anthropic.APIKey)
			g.providers = map[string]ports.Provider{
				"openai":    openaiP,
				"anthropic": anthropicP,
			}
			// Use default routing if no config
			if len(cfg.Routing.Rules) == 0 {
				cfg.Routing = config.RoutingConfig{
					Rules: []config.RoutingRule{
						{ModelPrefix: "claude", Provider: "anthropic"},
						{ModelPrefix: "gpt", Provider: "openai"},
					},
					DefaultProvider: "openai",
				}
			}
		}

		// Create router for single-tenant mode
		router := routerpkg.NewProviderRouter(g.providers, cfg.Routing)
		g.providers["_router"] = router

		// Attach thread/event stores
		if threadStore, ok := g.storage.(interface{ SetThreadState(string, string) error }); ok {
			attachThreadStore(threadStore, g.storage, g.providers, cfg.Providers)
		}
	}

	// Initialize shadow manager if storage supports it
	if shadowStore, ok := g.storage.(ports.ShadowStore); ok {
		// Register shadow codecs
		shadow.RegisterCodec(domain.APITypeOpenAI, openai.NewCodec())
		shadow.RegisterCodec(domain.APITypeAnthropic, anthropic.NewCodec())

		// Create provider lookup
		providerLookup := func(name string) (ports.Provider, error) {
			if p, ok := g.providers[name]; ok {
				return p, nil
			}
			return nil, nil
		}

		shadowMgr := shadow.NewManager(shadow.ManagerConfig{
			Store:          shadowStore,
			ProviderLookup: providerLookup,
			CodecLookup:    shadow.DefaultCodecLookup(),
			Logger:         g.logger,
		})
		shadow.SetGlobalManager(shadowMgr)
		g.logger.Info("shadow mode manager initialized")
	}

	return nil
}

// initFrontdoors initializes frontdoors from configuration.
func (g *Gateway) initFrontdoors(cfg *config.Config) error {
	g.logger.Debug("initializing frontdoors", slog.Int("count", len(cfg.Apps)))

	// Create frontdoor registry
	frontdoorRegistry := frontdoor.NewRegistry()

	// Determine which apps to create
	var apps []config.AppConfig
	if len(cfg.Apps) > 0 {
		apps = cfg.Apps
	} else if len(cfg.Frontdoors) > 0 {
		// Legacy frontdoor config
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
		// Default frontdoors
		g.logger.Info("using default frontdoor configuration")
		apps = []config.AppConfig{
			{Name: "openai", Frontdoor: "openai", Path: "/openai"},
			{Name: "anthropic", Frontdoor: "anthropic", Path: "/anthropic"},
		}
	}

	// Get router from providers
	router, ok := g.providers["_router"]
	if !ok {
		return fmt.Errorf("router not initialized")
	}

	// Remove the special router key from providers map for handler creation
	providers := make(map[string]ports.Provider)
	for k, v := range g.providers {
		if k != "_router" {
			providers[k] = v
		}
	}

	// Create handlers
	var err error
	g.handlers, err = frontdoorRegistry.CreateHandlers(apps, router, providers, g.storage)
	if err != nil {
		return fmt.Errorf("create handlers: %w", err)
	}

	g.logger.Info("frontdoor handlers created", slog.Int("count", len(g.handlers)))
	return nil
}

// startServer starts the HTTP server.
func (g *Gateway) startServer(cfg *config.Config) error {
	g.logger.Debug("starting HTTP server", slog.Int("port", cfg.Server.Port))

	// Create Chi router with middleware
	r := chi.NewRouter()

	// Apply middleware
	r.Use(apimw.RequestIDMiddleware)
	r.Use(apimw.LoggingMiddleware(g.logger))

	// Add auth middleware if auth provider is configured
	if g.auth != nil {
		// Create authenticator wrapper for middleware
		// Note: This requires adapting our auth.AuthProvider to auth.Authenticator interface
		// For now, we'll skip auth middleware and handle it in the future
		g.logger.Info("auth provider configured but middleware not yet wired")
	}

	r.Use(apimw.TimeoutMiddleware(30 * time.Second))
	r.Use(middleware.Recoverer)

	// Wrap with OpenTelemetry
	r.Use(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "poly-gateway")
	})

	// Register all frontdoor handlers
	for _, reg := range g.handlers {
		method := reg.Method
		if method == "" {
			method = http.MethodPost
		}

		switch method {
		case http.MethodGet:
			r.Get(reg.Path, reg.Handler)
		case http.MethodPost:
			r.Post(reg.Path, reg.Handler)
		default:
			r.Method(method, reg.Path, http.HandlerFunc(reg.Handler))
		}

		g.logger.Info("registered handler",
			slog.String("method", method),
			slog.String("path", reg.Path))
	}

	// Register Responses API handlers if storage supports it
	if interactionStore, ok := g.storage.(ports.InteractionStore); ok {
		// Get router for Responses API
		router, _ := g.providers["_router"]

		// Determine which apps should have Responses API
		responsesApps := g.resolveResponsesBasePaths(cfg)

		frontdoorRegistry := frontdoor.NewRegistry()
		for _, appCfg := range responsesApps {
			opts := frontdoor.ResponsesHandlerOptions{
				ForceStore: appCfg.ForceStore,
			}
			responsesHandlers := frontdoorRegistry.CreateResponsesHandlers(
				appCfg.Path,
				interactionStore,
				router,
				opts,
			)

			for _, reg := range responsesHandlers {
				method := reg.Method
				if method == "" {
					method = http.MethodPost
				}

				switch method {
				case http.MethodGet:
					r.Get(reg.Path, reg.Handler)
				case http.MethodPost:
					r.Post(reg.Path, reg.Handler)
				default:
					r.Method(method, reg.Path, http.HandlerFunc(reg.Handler))
				}

				g.logger.Info("registered Responses API handler",
					slog.String("method", method),
					slog.String("path", reg.Path))
			}
		}
	}

	// Initialize Control Plane
	// Note: This needs refactoring to use our new storage interface
	// For now, we'll try to mount it if possible
	if conversationStore, ok := g.storage.(storage.ConversationStore); ok {
		cpServer := controlplane.NewServer(cfg, conversationStore, nil)
		r.Mount("/admin", cpServer)
		g.logger.Info("registered control plane", slog.String("path", "/admin"))
	}

	// Create HTTP server
	g.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second, // Default timeout
		WriteTimeout: 30 * time.Second, // Default timeout
	}

	// Start server in background
	go func() {
		g.logger.Info("HTTP server listening", slog.Int("port", cfg.Server.Port))
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			g.logger.Error("server error", slog.String("error", err.Error()))
		}
	}()

	return nil
}

// responsesAppConfig holds config for Responses API
type responsesAppConfig struct {
	Path       string
	ForceStore bool
}

// resolveResponsesBasePaths determines which apps should have Responses API enabled
func (g *Gateway) resolveResponsesBasePaths(cfg *config.Config) []responsesAppConfig {
	seen := make(map[string]responsesAppConfig)

	for _, app := range cfg.Apps {
		if app.Frontdoor == "openai" && app.EnableResponses {
			if app.Path != "" && seen[app.Path].Path == "" {
				seen[app.Path] = responsesAppConfig{
					Path:       app.Path,
					ForceStore: app.ForceStore,
				}
			}
		}
	}

	result := make([]responsesAppConfig, 0, len(seen))
	for _, cfg := range seen {
		result = append(result, cfg)
	}
	return result
}
