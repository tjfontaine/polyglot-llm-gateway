package frontdoor

import (
	"fmt"
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	anthropic_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/anthropic"
	openai_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/openai"
	responses_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/responses"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Register built-in frontdoors at package initialization.
// New frontdoors should add their registration here.
func init() {
	// Register OpenAI frontdoor
	RegisterFactory(FrontdoorFactory{
		Type:        openai_frontdoor.FrontdoorType,
		APIType:     openai_frontdoor.APIType(),
		Description: "OpenAI Chat Completions API format",
		CreateHandlers: func(cfg HandlerConfig) []HandlerRegistration {
			handler := openai_frontdoor.NewHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models)
			regs := openai_frontdoor.CreateHandlerRegistrations(handler, cfg.BasePath)
			result := make([]HandlerRegistration, len(regs))
			for i, r := range regs {
				result[i] = HandlerRegistration{Path: r.Path, Method: r.Method, Handler: r.Handler}
			}
			return result
		},
	})

	// Register Anthropic frontdoor
	RegisterFactory(FrontdoorFactory{
		Type:        anthropic_frontdoor.FrontdoorType,
		APIType:     anthropic_frontdoor.APIType(),
		Description: "Anthropic Messages API format",
		CreateHandlers: func(cfg HandlerConfig) []HandlerRegistration {
			handler := anthropic_frontdoor.NewHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models)
			regs := anthropic_frontdoor.CreateHandlerRegistrations(handler, cfg.BasePath)
			result := make([]HandlerRegistration, len(regs))
			for i, r := range regs {
				result[i] = HandlerRegistration{Path: r.Path, Method: r.Method, Handler: r.Handler}
			}
			return result
		},
	})
}

// HandlerRegistration represents a registered handler
type HandlerRegistration struct {
	Path    string
	Method  string
	Handler func(http.ResponseWriter, *http.Request)
}

// Registry creates and registers frontdoor handlers.
// Frontdoors are created using registered FrontdoorFactory instances.
// See factory.go for documentation on how to add new frontdoors.
type Registry struct{}

// NewRegistry creates a new frontdoor registry
func NewRegistry() *Registry {
	return &Registry{}
}

// CreateHandlers creates frontdoor handlers based on configuration.
// It uses the registered FrontdoorFactory for each specified frontdoor type.
func (r *Registry) CreateHandlers(configs []config.AppConfig, router domain.Provider, providers map[string]domain.Provider, store storage.ConversationStore) ([]HandlerRegistration, error) {
	var registrations []HandlerRegistration

	for _, cfg := range configs {
		// Determine which provider to use
		var p domain.Provider = router
		if cfg.Provider != "" {
			if specificProvider, ok := providers[cfg.Provider]; ok {
				p = specificProvider
			} else {
				return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
			}
		}

		// Apply model override if configured
		if cfg.DefaultModel != "" {
			p = provider.NewModelOverrideProvider(p, cfg.DefaultModel)
		}

		if len(cfg.ModelRouting.PrefixProviders) > 0 || len(cfg.ModelRouting.Rewrites) > 0 || cfg.ModelRouting.Fallback != nil {
			mapper, err := provider.NewModelMappingProvider(p, providers, cfg.ModelRouting)
			if err != nil {
				return nil, err
			}
			p = mapper
		}

		// Use the factory pattern to create handlers
		handlerCfg := HandlerConfig{
			Provider: p,
			Store:    store,
			AppName:  cfg.Name,
			BasePath: cfg.Path,
			Models:   cfg.Models,
		}

		handlers, err := createHandlersFromFactory(cfg.Frontdoor, handlerCfg)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, handlers...)
	}

	return registrations, nil
}

// CreateResponsesHandlers creates Responses API handlers
func (r *Registry) CreateResponsesHandlers(basePath string, store storage.ConversationStore, provider domain.Provider) []HandlerRegistration {
	handler := responses_frontdoor.NewHandler(store, provider)

	return []HandlerRegistration{
		// Responses API (new)
		{Path: basePath + "/v1/responses", Method: http.MethodPost, Handler: handler.HandleCreateResponse},
		{Path: basePath + "/v1/responses/{response_id}", Method: http.MethodGet, Handler: handler.HandleGetResponse},
		{Path: basePath + "/v1/responses/{response_id}/cancel", Method: http.MethodPost, Handler: handler.HandleCancelResponse},

		// Threads API (legacy)
		{Path: basePath + "/v1/threads", Method: http.MethodPost, Handler: handler.HandleCreateThread},
		{Path: basePath + "/v1/threads/{thread_id}", Method: http.MethodGet, Handler: handler.HandleGetThread},
		{Path: basePath + "/v1/threads/{thread_id}/messages", Method: http.MethodPost, Handler: handler.HandleCreateMessage},
		{Path: basePath + "/v1/threads/{thread_id}/messages", Method: http.MethodGet, Handler: handler.HandleListMessages},
		{Path: basePath + "/v1/threads/{thread_id}/runs", Method: http.MethodPost, Handler: handler.HandleCreateRun},
	}
}
