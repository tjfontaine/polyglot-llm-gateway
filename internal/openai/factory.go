// Factory provides registration for provider functionality.
package openai

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "openai"

// ProviderTypeCompatible is the provider type for OpenAI-compatible APIs.
const ProviderTypeCompatible = "openai-compatible"

// RegisterProviderFactories registers the OpenAI provider factories.
func RegisterProviderFactories() {
	if !provider.IsRegistered(ProviderType) {
		provider.RegisterFactory(provider.ProviderFactory{
			Type:           ProviderType,
			APIType:        domain.APITypeOpenAI,
			Description:    "OpenAI API provider (GPT models)",
			Create:         CreateProviderFromConfig,
			ValidateConfig: ValidateProviderConfig,
		})
	}

	if !provider.IsRegistered(ProviderTypeCompatible) {
		provider.RegisterFactory(provider.ProviderFactory{
			Type:           ProviderTypeCompatible,
			APIType:        domain.APITypeOpenAI,
			Description:    "OpenAI-compatible API provider (local models, etc.)",
			Create:         CreateProviderFromConfig,
			ValidateConfig: ValidateProviderConfig,
		})
	}
}

// CreateProviderFromConfig creates a new OpenAI provider from configuration.
func CreateProviderFromConfig(cfg config.ProviderConfig) (ports.Provider, error) {
	var opts []ProviderOption
	if cfg.BaseURL != "" {
		opts = append(opts, WithProviderBaseURL(cfg.BaseURL))
	}
	if cfg.UseResponsesAPI {
		opts = append(opts, WithResponsesAPI(true))
	}
	if cfg.ResponsesThreadKeyPath != "" {
		opts = append(opts, WithResponsesThreadKeyPath(cfg.ResponsesThreadKeyPath))
	}
	if cfg.ResponsesThreadPersistence {
		opts = append(opts, WithResponsesThreadPersistence(true))
	}
	return NewProvider(cfg.APIKey, opts...), nil
}

// ValidateProviderConfig validates the provider configuration.
func ValidateProviderConfig(cfg config.ProviderConfig) error {
	// API key is optional for OpenAI-compatible providers (some local models don't need it)
	return nil
}
