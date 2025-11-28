package anthropic

import (
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// FrontdoorType is the frontdoor type identifier used in configuration.
const FrontdoorType = "anthropic"

// APIType returns the canonical API type for this frontdoor.
func APIType() domain.APIType {
	return domain.APITypeAnthropic
}

// CreateHandlerRegistrations creates the HTTP handler registrations for Anthropic frontdoor.
// This function is called by the frontdoor registry factory.
func CreateHandlerRegistrations(handler *Handler, basePath string) []struct {
	Path    string
	Method  string
	Handler func(http.ResponseWriter, *http.Request)
} {
	return []struct {
		Path    string
		Method  string
		Handler func(http.ResponseWriter, *http.Request)
	}{
		{Path: basePath + "/v1/messages", Method: http.MethodPost, Handler: handler.HandleMessages},
		{Path: basePath + "/v1/messages/count_tokens", Method: http.MethodPost, Handler: handler.HandleCountTokens},
		{Path: basePath + "/v1/models", Method: http.MethodGet, Handler: handler.HandleListModels},
	}
}
