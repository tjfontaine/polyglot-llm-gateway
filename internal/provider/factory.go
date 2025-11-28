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
//  4. Factory Registration: Add a ProviderFactory to this package's init() or
//     create a self-registering init() in your provider package.
//
// Example factory registration:
//
//	func init() {
//	    provider.RegisterFactory(provider.ProviderFactory{
//	        Type:        "gemini",
//	        APIType:     domain.APITypeGemini,
//	        Description: "Google Gemini API provider",
//	        Create: func(cfg config.ProviderConfig) (domain.Provider, error) {
//	            var opts []gemini.ProviderOption
//	            if cfg.BaseURL != "" {
//	                opts = append(opts, gemini.WithBaseURL(cfg.BaseURL))
//	            }
//	            return gemini.New(cfg.APIKey, opts...), nil
//	        },
//	        ValidateConfig: func(cfg config.ProviderConfig) error {
//	            if cfg.APIKey == "" {
//	                return fmt.Errorf("api_key is required for gemini provider")
//	            }
//	            return nil
//	        },
//	    })
//	}
package provider

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ProviderFactory defines how to create a provider of a specific type.
// Each provider type (openai, anthropic, etc.) registers a factory that
// knows how to create instances from configuration.
type ProviderFactory struct {
	// Type is the provider type identifier used in configuration
	// (e.g., "openai", "anthropic", "openai-compatible")
	Type string

	// APIType is the canonical API type this provider implements
	APIType domain.APIType

	// Description provides a human-readable description of the provider
	Description string

	// Create instantiates a new provider from configuration.
	// This is called by the registry to create provider instances.
	Create func(cfg config.ProviderConfig) (domain.Provider, error)

	// ValidateConfig performs provider-specific configuration validation.
	// Optional: if nil, no additional validation is performed.
	ValidateConfig func(cfg config.ProviderConfig) error
}

// factoryRegistry holds registered provider factories
var (
	factoryMu   sync.RWMutex
	factoryMap  = make(map[string]ProviderFactory)
	factoryList []ProviderFactory
)

// RegisterFactory registers a provider factory for a specific type.
// This should be called from init() in each provider package.
// Panics if a factory with the same type is already registered.
func RegisterFactory(f ProviderFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()

	if f.Type == "" {
		panic("provider factory type cannot be empty")
	}
	if f.Create == nil {
		panic(fmt.Sprintf("provider factory %q must have a Create function", f.Type))
	}

	if _, exists := factoryMap[f.Type]; exists {
		panic(fmt.Sprintf("provider factory %q already registered", f.Type))
	}

	factoryMap[f.Type] = f
	factoryList = append(factoryList, f)
}

// GetFactory returns the factory for a provider type, if registered.
func GetFactory(providerType string) (ProviderFactory, bool) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()

	f, ok := factoryMap[providerType]
	return f, ok
}

// ListFactories returns all registered provider factories sorted by type.
func ListFactories() []ProviderFactory {
	factoryMu.RLock()
	defer factoryMu.RUnlock()

	result := make([]ProviderFactory, len(factoryList))
	copy(result, factoryList)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Type < result[j].Type
	})
	return result
}

// ListProviderTypes returns all registered provider type names.
func ListProviderTypes() []string {
	factories := ListFactories()
	types := make([]string, len(factories))
	for i, f := range factories {
		types[i] = f.Type
	}
	return types
}

// IsRegistered returns true if a provider type is registered.
func IsRegistered(providerType string) bool {
	_, ok := GetFactory(providerType)
	return ok
}

// ValidateProviderConfig validates a provider configuration using
// the registered factory's validation function.
func ValidateProviderConfig(cfg config.ProviderConfig) error {
	f, ok := GetFactory(cfg.Type)
	if !ok {
		return fmt.Errorf("unknown provider type: %s (registered types: %v)", cfg.Type, ListProviderTypes())
	}

	if f.ValidateConfig != nil {
		return f.ValidateConfig(cfg)
	}
	return nil
}

// createFromFactory creates a provider using the registered factory.
// This is called by the Registry.CreateProvider method.
func createFromFactory(cfg config.ProviderConfig) (domain.Provider, error) {
	f, ok := GetFactory(cfg.Type)
	if !ok {
		return nil, fmt.Errorf("unknown provider type: %s (registered types: %v)", cfg.Type, ListProviderTypes())
	}

	// Validate config if validator is provided
	if f.ValidateConfig != nil {
		if err := f.ValidateConfig(cfg); err != nil {
			return nil, fmt.Errorf("invalid configuration for provider type %s: %w", cfg.Type, err)
		}
	}

	return f.Create(cfg)
}

// ClearFactories removes all registered factories (for testing only).
func ClearFactories() {
	factoryMu.Lock()
	defer factoryMu.Unlock()

	factoryMap = make(map[string]ProviderFactory)
	factoryList = nil
}
