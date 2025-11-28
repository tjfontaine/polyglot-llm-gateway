package openai

import (
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
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

// CreateHandlerRegistrations creates the HTTP handler registrations for OpenAI frontdoor.
// This function is called by the frontdoor registry factory.
func CreateHandlerRegistrations(handler *Handler, basePath string) []Route {
	return []Route{
		{Path: basePath + "/v1/chat/completions", Method: http.MethodPost, Handler: handler.HandleChatCompletion},
		{Path: basePath + "/v1/models", Method: http.MethodGet, Handler: handler.HandleListModels},
	}
}
