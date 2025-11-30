// Package provider contains the provider factory and registry for LLM backends.
//
// # Adding a New Provider
//
// To add a new provider (e.g., Google Gemini), implement the provider + codec
// and expose an explicit registration function that calls
// registry.RegisterFactory. Wire that registration from cmd/gateway (or tests)
// so we avoid init() side effects.
//
// Example registration in a provider package:
//
//	func RegisterProviderFactory() {
//	    if registry.IsRegistered(ProviderType) {
//	        return
//	    }
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
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
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
func createFromFactory(cfg config.ProviderConfig) (ports.Provider, error) {
	return registry.CreateFromFactory(cfg)
}
