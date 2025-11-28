// Factory provides registration for provider functionality.
package openai

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	providerregistry "github.com/tjfontaine/polyglot-llm-gateway/internal/provider/registry"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "openai"

// ProviderTypeCompatible is the provider type for OpenAI-compatible APIs.
const ProviderTypeCompatible = "openai-compatible"

// Register this provider at package initialization.
func init() {
	// Register OpenAI provider
	providerregistry.RegisterFactory(providerregistry.ProviderFactory{
		Type:           ProviderType,
		APIType:        domain.APITypeOpenAI,
		Description:    "OpenAI API provider (GPT models)",
		Create:         CreateProviderFromConfig,
		ValidateConfig: ValidateProviderConfig,
	})

	// Register OpenAI-compatible provider (for local models, etc.)
	providerregistry.RegisterFactory(providerregistry.ProviderFactory{
		Type:           ProviderTypeCompatible,
		APIType:        domain.APITypeOpenAI,
		Description:    "OpenAI-compatible API provider (local models, etc.)",
		Create:         CreateProviderFromConfig,
		ValidateConfig: ValidateProviderConfig,
	})
}

// CreateProviderFromConfig creates a new OpenAI provider from configuration.
func CreateProviderFromConfig(cfg config.ProviderConfig) (domain.Provider, error) {
	var opts []ProviderOption
	if cfg.BaseURL != "" {
		opts = append(opts, WithProviderBaseURL(cfg.BaseURL))
	}
	if cfg.UseResponsesAPI {
		opts = append(opts, WithResponsesAPI(true))
	}
	return NewProvider(cfg.APIKey, opts...), nil
}

// ValidateProviderConfig validates the provider configuration.
func ValidateProviderConfig(cfg config.ProviderConfig) error {
	// API key is optional for OpenAI-compatible providers (some local models don't need it)
	return nil
}
