package frontdoor

import (
	"fmt"
	"net/http" // Keep net/http for HandlerRegistration type

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	anthropic_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/anthropic"
	openai_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/openai"
	responses_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/responses"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// HandlerRegistration represents a registered handler
type HandlerRegistration struct {
	Path    string
	Handler func(http.ResponseWriter, *http.Request)
}

// Registry creates and registers frontdoor handlers
type Registry struct{}

// NewRegistry creates a new frontdoor registry
func NewRegistry() *Registry {
	return &Registry{}
}

// CreateHandlers creates frontdoor handlers based on configuration
func (r *Registry) CreateHandlers(configs []config.FrontdoorConfig, router domain.Provider, providers map[string]domain.Provider) ([]HandlerRegistration, error) {
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

		switch cfg.Type {
		case "openai":
			handler := openai_frontdoor.NewHandler(p)
			registrations = append(registrations, HandlerRegistration{
				Path:    cfg.Path + "/v1/chat/completions",
				Handler: handler.HandleChatCompletion,
			})
		case "anthropic":
			handler := anthropic_frontdoor.NewHandler(p)
			registrations = append(registrations, HandlerRegistration{
				Path:    cfg.Path + "/v1/messages",
				Handler: handler.HandleMessages,
			})
		default:
			return nil, fmt.Errorf("unknown frontdoor type: %s", cfg.Type)
		}
	}

	return registrations, nil
}

// CreateResponsesHandlers creates Responses API handlers
func (r *Registry) CreateResponsesHandlers(basePath string, store storage.ConversationStore, provider domain.Provider) []HandlerRegistration {
	handler := responses_frontdoor.NewHandler(store, provider)

	return []HandlerRegistration{
		{Path: basePath + "/v1/threads", Handler: handler.HandleCreateThread},
		{Path: basePath + "/v1/threads/{thread_id}", Handler: handler.HandleGetThread},
		{Path: basePath + "/v1/threads/{thread_id}/messages", Handler: handler.HandleCreateMessage},
		{Path: basePath + "/v1/threads/{thread_id}/messages", Handler: handler.HandleListMessages},
		{Path: basePath + "/v1/threads/{thread_id}/runs", Handler: handler.HandleCreateRun},
	}
}
