package provider

import (
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider/openai"
)

// Registry creates providers from configuration
type Registry struct{}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) CreateProvider(cfg config.ProviderConfig) (domain.Provider, error) {
	switch cfg.Type {
	case "openai":
		return openai.New(cfg.APIKey), nil
	case "anthropic":
		return anthropic.New(cfg.APIKey), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
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
