package frontdoor

import (
	"fmt"
	"net/http"

	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
	anthropic_frontdoor "github.com/tjfontaine/poly-llm-gateway/internal/frontdoor/anthropic"
	openai_frontdoor "github.com/tjfontaine/poly-llm-gateway/internal/frontdoor/openai"
)

type HandlerRegistration struct {
	Path    string
	Handler http.HandlerFunc
}

type Registry struct{}

func NewRegistry() *Registry {
	return &Registry{}
}

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
