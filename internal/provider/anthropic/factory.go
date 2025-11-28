package anthropic

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/registry"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "anthropic"

// Register this provider at package initialization.
func init() {
	registry.RegisterFactory(registry.ProviderFactory{
		Type:           ProviderType,
		APIType:        domain.APITypeAnthropic,
		Description:    "Anthropic API provider (Claude models)",
		Create:         CreateFromConfig,
		ValidateConfig: ValidateConfig,
	})
}

// CreateFromConfig creates a new Anthropic provider from configuration.
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
