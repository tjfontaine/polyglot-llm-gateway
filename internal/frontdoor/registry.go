package frontdoor

import (
	"fmt"
	"net/http" // Keep net/http for HandlerRegistration type

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	anthropic_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/anthropic"
	openai_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/openai"
	responses_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/responses"
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
func (r *Registry) CreateHandlers(configs []config.FrontdoorConfig, provider domain.Provider) ([]HandlerRegistration, error) {
	var registrations []HandlerRegistration

	for _, cfg := range configs {
		switch cfg.Type {
		case "openai":
			handler := openai_frontdoor.NewHandler(provider)
			registrations = append(registrations, HandlerRegistration{
				Path:    cfg.Path + "/v1/chat/completions",
				Handler: handler.HandleChatCompletion,
			})
		case "anthropic":
			handler := anthropic_frontdoor.NewHandler(provider)
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
