package anthropic

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "anthropic"

// CreateFromConfig creates a new Anthropic provider from configuration.
// This function is used by the provider registry factory.
func CreateFromConfig(cfg config.ProviderConfig) (domain.Provider, error) {
	var opts []ProviderOption
	if cfg.BaseURL != "" {
		opts = append(opts, WithBaseURL(cfg.BaseURL))
	}
	return New(cfg.APIKey, opts...), nil
}

// ValidateConfig validates the provider configuration.
func ValidateConfig(cfg config.ProviderConfig) error {
	// API key validation could be added here
	return nil
}
