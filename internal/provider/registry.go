// Package provider provides provider factory registration and the Registry
// for creating provider instances from configuration.
//
// # Adding a New Provider
//
// To add a new provider (e.g., Google Gemini), implement the provider + codec
// and expose an explicit registration function that calls RegisterFactory.
// Wire that registration from cmd/gateway (or internal/registration) so
// registration is explicit instead of relying on init() side effects.
//
// Example in a provider package:
//
//	func RegisterProviderFactory() {
//	    if provider.IsRegistered(ProviderType) {
//	        return
//	    }
//	    provider.RegisterFactory(provider.ProviderFactory{
//	        Type:           ProviderType,
//	        APIType:        domain.APITypeGemini,
//	        Description:    "Google Gemini API provider",
//	        Create:         CreateFromConfig,
//	        ValidateConfig: ValidateConfig,
//	    })
//	}
package provider

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/passthrough"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
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
	Create func(cfg config.ProviderConfig) (ports.Provider, error)

	// ValidateConfig performs provider-specific configuration validation.
	// Optional: if nil, no additional validation is performed.
	ValidateConfig func(cfg config.ProviderConfig) error
}

// Factory registry: global registration of provider factories
var (
	factoryMu   sync.RWMutex
	factoryMap  = make(map[string]ProviderFactory)
	factoryList []ProviderFactory
)

// RegisterFactory registers a provider factory for a specific type.
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
func CreateFromFactory(cfg config.ProviderConfig) (ports.Provider, error) {
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

// Registry creates providers from configuration.
// Providers are created using registered ProviderFactory instances.
type Registry struct{}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// CreateProvider creates a provider instance from configuration.
// It uses the registered ProviderFactory for the specified provider type.
func (r *Registry) CreateProvider(cfg config.ProviderConfig) (ports.Provider, error) {
	// Use the factory pattern to create providers
	baseProvider, err := CreateFromFactory(cfg)
	if err != nil {
		return nil, err
	}

	// Wrap with pass-through if enabled
	if cfg.EnablePassthrough {
		var opts []passthrough.Option
		opts = append(opts, passthrough.WithAPIKey(cfg.APIKey))
		if cfg.BaseURL != "" {
			opts = append(opts, passthrough.WithBaseURL(cfg.BaseURL))
		}
		return passthrough.NewPassthroughProvider(baseProvider, opts...), nil
	}

	return baseProvider, nil
}

// CreateProviders creates multiple provider instances from configurations.
func (r *Registry) CreateProviders(configs []config.ProviderConfig) (map[string]ports.Provider, error) {
	providers := make(map[string]ports.Provider)
	for _, cfg := range configs {
		p, err := r.CreateProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", cfg.Name, err)
		}
		providers[cfg.Name] = p
	}
	return providers, nil
}
