package provider

import (
	"fmt"

	passthrough "github.com/tjfontaine/polyglot-llm-gateway/internal/backend/passthrough"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Registry creates providers from configuration.
// Providers are created using registered ProviderFactory instances.
// See factory.go for documentation on how to add new providers.
//
// Note: Provider packages (anthropic, openai) must be imported elsewhere
// (e.g., in cmd/gateway/main.go) to trigger their init() registration.
// This avoids import cycles since those packages now contain frontdoor code
// that depends on server, conversation, and other packages.
type Registry struct{}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// CreateProvider creates a provider instance from configuration.
// It uses the registered ProviderFactory for the specified provider type.
func (r *Registry) CreateProvider(cfg config.ProviderConfig) (ports.Provider, error) {
	// Use the factory pattern to create providers
	baseProvider, err := createFromFactory(cfg)
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
