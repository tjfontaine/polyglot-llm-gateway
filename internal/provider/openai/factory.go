package openai

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/registry"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "openai"

// ProviderTypeCompatible is the provider type for OpenAI-compatible APIs.
const ProviderTypeCompatible = "openai-compatible"

// Register this provider at package initialization.
func init() {
	// Register OpenAI provider
	registry.RegisterFactory(registry.ProviderFactory{
		Type:           ProviderType,
		APIType:        domain.APITypeOpenAI,
		Description:    "OpenAI API provider (GPT models)",
		Create:         CreateFromConfig,
		ValidateConfig: ValidateConfig,
	})

	// Register OpenAI-compatible provider (for local models, etc.)
	registry.RegisterFactory(registry.ProviderFactory{
		Type:           ProviderTypeCompatible,
		APIType:        domain.APITypeOpenAI,
		Description:    "OpenAI-compatible API provider (local models, etc.)",
		Create:         CreateFromConfig,
		ValidateConfig: ValidateConfig,
	})
}

// CreateFromConfig creates a new OpenAI provider from configuration.
func CreateFromConfig(cfg config.ProviderConfig) (domain.Provider, error) {
	var opts []ProviderOption
	if cfg.BaseURL != "" {
		opts = append(opts, WithBaseURL(cfg.BaseURL))
	}
	if cfg.UseResponsesAPI {
		opts = append(opts, WithResponsesAPI(true))
	}
	return New(cfg.APIKey, opts...), nil
}

// ValidateConfig validates the provider configuration.
func ValidateConfig(cfg config.ProviderConfig) error {
	// API key is optional for OpenAI-compatible providers (some local models don't need it)
	return nil
}
