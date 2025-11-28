// Package registry provides provider factory registration and lookup.
//
// # Adding a New Provider
//
// Each provider package should register itself via init():
//
//	func init() {
//	    registry.RegisterFactory(registry.ProviderFactory{
//	        Type:        ProviderType,
//	        APIType:     domain.APITypeGemini,
//	        Description: "Google Gemini API provider",
//	        Create:      CreateFromConfig,
//	        ValidateConfig: ValidateConfig,
//	    })
//	}
//
// Provider packages must be imported (via blank import) in imports.go
// to ensure their init() functions run.
package registry

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

// CreateFromFactory creates a provider using the registered factory.
func CreateFromFactory(cfg config.ProviderConfig) (domain.Provider, error) {
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
