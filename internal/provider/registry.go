package provider

import (
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"

	// Import consolidated packages to trigger their init() registration.
	// All related code (types, client, codec, provider, frontdoor) is in one package.
	_ "github.com/tjfontaine/polyglot-llm-gateway/internal/anthropic"
	_ "github.com/tjfontaine/polyglot-llm-gateway/internal/openai"
)

// Registry creates providers from configuration.
// Providers are created using registered ProviderFactory instances.
// See factory.go for documentation on how to add new providers.
type Registry struct{}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// CreateProvider creates a provider instance from configuration.
// It uses the registered ProviderFactory for the specified provider type.
func (r *Registry) CreateProvider(cfg config.ProviderConfig) (domain.Provider, error) {
	// Use the factory pattern to create providers
	baseProvider, err := createFromFactory(cfg)
	if err != nil {
		return nil, err
	}

	// Wrap with pass-through if enabled
	if cfg.EnablePassthrough {
		var passthroughOpts []PassthroughOption
		passthroughOpts = append(passthroughOpts, WithPassthroughAPIKey(cfg.APIKey))
		if cfg.BaseURL != "" {
			passthroughOpts = append(passthroughOpts, WithPassthroughBaseURL(cfg.BaseURL))
		}
		return NewPassthroughProvider(baseProvider, passthroughOpts...), nil
	}

	return baseProvider, nil
}

func (r *Registry) CreateProviders(configs []config.ProviderConfig) (map[string]domain.Provider, error) {
	providers := make(map[string]domain.Provider)
	for _, cfg := range configs {
		p, err := r.CreateProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", cfg.Name, err)
		}
		providers[cfg.Name] = p
	}
	return providers, nil
}
