package openai

import (
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/registry"
)

// FrontdoorType is the frontdoor type identifier used in configuration.
const FrontdoorType = "openai"

// APIType returns the canonical API type for this frontdoor.
func APIType() domain.APIType {
	return domain.APITypeOpenAI
}

// Route defines an HTTP route registration.
type Route struct {
	Path    string
	Method  string
	Handler func(http.ResponseWriter, *http.Request)
}

// Register this frontdoor at package initialization.
func init() {
	registry.RegisterFactory(registry.FrontdoorFactory{
		Type:           FrontdoorType,
		APIType:        APIType(),
		Description:    "OpenAI Chat Completions API format",
		CreateHandlers: createHandlers,
	})
}

// createHandlers creates handler registrations for OpenAI frontdoor.
func createHandlers(cfg registry.HandlerConfig) []registry.HandlerRegistration {
	handler := NewHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models)
	routes := CreateHandlerRegistrations(handler, cfg.BasePath)
	result := make([]registry.HandlerRegistration, len(routes))
	for i, r := range routes {
		result[i] = registry.HandlerRegistration{Path: r.Path, Method: r.Method, Handler: r.Handler}
	}
	return result
}

// CreateHandlerRegistrations creates the HTTP handler registrations for OpenAI frontdoor.
func CreateHandlerRegistrations(handler *Handler, basePath string) []Route {
	return []Route{
		{Path: basePath + "/v1/chat/completions", Method: http.MethodPost, Handler: handler.HandleChatCompletion},
		{Path: basePath + "/v1/models", Method: http.MethodGet, Handler: handler.HandleListModels},
	}
}
