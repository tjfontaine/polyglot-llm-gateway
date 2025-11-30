// Factory provides registration for provider functionality.
package anthropic

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	providerregistry "github.com/tjfontaine/polyglot-llm-gateway/internal/provider/registry"
)

// ProviderType is the provider type identifier used in configuration.
const ProviderType = "anthropic"

// RegisterProviderFactory registers the Anthropic provider factory.
func RegisterProviderFactory() {
	if providerregistry.IsRegistered(ProviderType) {
		return
	}

	providerregistry.RegisterFactory(providerregistry.ProviderFactory{
		Type:           ProviderType,
		APIType:        domain.APITypeAnthropic,
		Description:    "Anthropic API provider (Claude models)",
		Create:         CreateProviderFromConfig,
		ValidateConfig: ValidateProviderConfig,
	})
}

// CreateProviderFromConfig creates a new Anthropic provider from configuration.
func CreateProviderFromConfig(cfg config.ProviderConfig) (ports.Provider, error) {
	var opts []ProviderOption
	if cfg.BaseURL != "" {
		opts = append(opts, WithProviderBaseURL(cfg.BaseURL))
	}
	return NewProvider(cfg.APIKey, opts...), nil
}

// ValidateProviderConfig validates the provider configuration.
func ValidateProviderConfig(cfg config.ProviderConfig) error {
	// API key validation could be added here
	return nil
}
