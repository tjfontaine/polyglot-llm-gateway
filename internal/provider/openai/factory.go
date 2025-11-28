package openai

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "openai"

// ProviderTypeCompatible is the provider type for OpenAI-compatible APIs.
const ProviderTypeCompatible = "openai-compatible"

// CreateFromConfig creates a new OpenAI provider from configuration.
// This function is used by the provider registry factory.
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
