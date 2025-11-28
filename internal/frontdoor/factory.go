// Package frontdoor contains the frontdoor factory and registry for API handlers.
//
// # Adding a New Frontdoor
//
// To add a new frontdoor (e.g., for Google Gemini API format), you must implement
// the following components and register them with the factory:
//
//  1. Handler: Create `internal/frontdoor/<api>/handler.go` with HTTP handlers
//     that decode API-specific requests to canonical format.
//
//  2. Codec: Create `internal/codec/<api>/codec.go` implementing the
//     `codec.Codec` interface for request/response translation.
//
//  3. Factory: Create `internal/frontdoor/<api>/factory.go` with:
//     - Self-registration in init() using registry.RegisterFactory
//     - CreateHandlers function that creates handler registrations
//
//  4. Import: Add a blank import in this package's registry.go to trigger init()
//
// Example factory.go in frontdoor package:
//
//	func init() {
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
