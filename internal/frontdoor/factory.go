// Package frontdoor contains the frontdoor factory and registry for API handlers.
//
// # Adding a New Frontdoor
//
// To add a new frontdoor (e.g., for Google Gemini API format), implement the
// handler + codec and expose an explicit registration function that calls
// registry.RegisterFactory. Wire that function from cmd/gateway (or tests) so
// registration is explicit instead of relying on init() side effects.
//
// Example factory.go in a frontdoor package:
//
//	func RegisterFrontdoor() {
//	    if registry.IsRegistered(FrontdoorType) {
//	        return
//	    }
//	    registry.RegisterFactory(registry.FrontdoorFactory{
//	        Type:           FrontdoorType,
//	        APIType:        APIType(),
//	        Description:    "Gemini API format",
//	        CreateHandlers: createHandlers,
//	    })
//	}
package frontdoor

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/registry"
)

// Re-export types from registry for convenience
type FrontdoorFactory = registry.FrontdoorFactory
type HandlerConfig = registry.HandlerConfig
type HandlerRegistration = registry.HandlerRegistration

// RegisterFactory registers a frontdoor factory (delegated to registry).
var RegisterFactory = registry.RegisterFactory

// GetFrontdoorFactory returns the factory for a frontdoor type (delegated to registry).
var GetFrontdoorFactory = registry.GetFactory

// ListFrontdoorFactories returns all registered frontdoor factories (delegated to registry).
var ListFrontdoorFactories = registry.ListFactories

// ListFrontdoorTypes returns all registered frontdoor type names (delegated to registry).
var ListFrontdoorTypes = registry.ListFrontdoorTypes

// IsFrontdoorRegistered returns true if a frontdoor type is registered (delegated to registry).
var IsFrontdoorRegistered = registry.IsRegistered

// ClearFrontdoorFactories removes all registered factories (for testing only).
var ClearFrontdoorFactories = registry.ClearFactories

// createHandlersFromFactory creates handlers using the registered factory.
func createHandlersFromFactory(frontdoorType string, cfg HandlerConfig) ([]HandlerRegistration, error) {
	return registry.CreateHandlersFromFactory(frontdoorType, cfg)
}
