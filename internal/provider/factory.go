// Package provider contains the provider factory and registry for LLM backends.
//
// # Adding a New Provider
//
// To add a new provider (e.g., Google Gemini), you must implement the following
// components and register them with the factory:
//
//  1. API Types: Create `internal/api/<provider>/types.go` with the provider's
//     request/response types.
//
//  2. Codec: Create `internal/codec/<provider>/codec.go` implementing the
//     `codec.Codec` interface for request/response translation.
//
//  3. Provider: Create `internal/provider/<provider>/provider.go` implementing
//     the `domain.Provider` interface (and optionally `domain.CapableProvider`).
//
//  4. Factory: Create `internal/provider/<provider>/factory.go` with:
//     - Self-registration in init() using registry.RegisterFactory
//     - CreateFromConfig and ValidateConfig functions
//
//  5. Import: Add a blank import in this package's imports.go to trigger init()
//
// Example factory.go in provider package:
//
//	func init() {
//	    registry.RegisterFactory(registry.ProviderFactory{
//	        Type:           ProviderType,
//	        APIType:        domain.APITypeGemini,
//	        Description:    "Google Gemini API provider",
//	        Create:         CreateFromConfig,
//	        ValidateConfig: ValidateConfig,
//	    })
//	}
package provider

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/registry"
)

// Re-export types from registry for convenience
type ProviderFactory = registry.ProviderFactory

// RegisterFactory registers a provider factory (delegated to registry).
var RegisterFactory = registry.RegisterFactory

// GetFactory returns the factory for a provider type (delegated to registry).
var GetFactory = registry.GetFactory

// ListFactories returns all registered provider factories (delegated to registry).
var ListFactories = registry.ListFactories

// ListProviderTypes returns all registered provider type names (delegated to registry).
var ListProviderTypes = registry.ListProviderTypes

// IsRegistered returns true if a provider type is registered (delegated to registry).
var IsRegistered = registry.IsRegistered

// ValidateProviderConfig validates a provider configuration (delegated to registry).
var ValidateProviderConfig = registry.ValidateProviderConfig

// ClearFactories removes all registered factories (for testing only).
var ClearFactories = registry.ClearFactories

// createFromFactory creates a provider using the registered factory.
func createFromFactory(cfg config.ProviderConfig) (domain.Provider, error) {
	return registry.CreateFromFactory(cfg)
}
