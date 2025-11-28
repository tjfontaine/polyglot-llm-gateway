package anthropic

import (
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/registry"
)

// FrontdoorType is the frontdoor type identifier used in configuration.
const FrontdoorType = "anthropic"

// APIType returns the canonical API type for this frontdoor.
func APIType() domain.APIType {
	return domain.APITypeAnthropic
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
		Description:    "Anthropic Messages API format",
		CreateHandlers: createHandlers,
	})
}

// createHandlers creates handler registrations for Anthropic frontdoor.
func createHandlers(cfg registry.HandlerConfig) []registry.HandlerRegistration {
	handler := NewHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models)
	routes := CreateHandlerRegistrations(handler, cfg.BasePath)
	result := make([]registry.HandlerRegistration, len(routes))
	for i, r := range routes {
		result[i] = registry.HandlerRegistration{Path: r.Path, Method: r.Method, Handler: r.Handler}
	}
	return result
}

// CreateHandlerRegistrations creates the HTTP handler registrations for Anthropic frontdoor.
func CreateHandlerRegistrations(handler *Handler, basePath string) []Route {
	return []Route{
		{Path: basePath + "/v1/messages", Method: http.MethodPost, Handler: handler.HandleMessages},
		{Path: basePath + "/v1/messages/count_tokens", Method: http.MethodPost, Handler: handler.HandleCountTokens},
		{Path: basePath + "/v1/models", Method: http.MethodGet, Handler: handler.HandleListModels},
	}
}
