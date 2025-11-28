package provider

import (
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/openai"
)

// Register built-in providers at package initialization.
// New providers should add their registration here.
func init() {
	// Register OpenAI provider
	RegisterFactory(ProviderFactory{
		Type:           openai.ProviderType,
		APIType:        domain.APITypeOpenAI,
		Description:    "OpenAI API provider (GPT models)",
		Create:         openai.CreateFromConfig,
		ValidateConfig: openai.ValidateConfig,
	})

	// Register OpenAI-compatible provider (for local models, etc.)
	RegisterFactory(ProviderFactory{
		Type:           openai.ProviderTypeCompatible,
		APIType:        domain.APITypeOpenAI,
		Description:    "OpenAI-compatible API provider (local models, etc.)",
		Create:         openai.CreateFromConfig,
		ValidateConfig: openai.ValidateConfig,
	})

	// Register Anthropic provider
	RegisterFactory(ProviderFactory{
		Type:           anthropic.ProviderType,
		APIType:        domain.APITypeAnthropic,
		Description:    "Anthropic API provider (Claude models)",
		Create:         anthropic.CreateFromConfig,
		ValidateConfig: anthropic.ValidateConfig,
	})
}

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
