package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ModelMappingProvider rewrites model names and routes requests to specific providers.
type ModelMappingProvider struct {
	defaultProvider domain.Provider
	providers       map[string]domain.Provider
	prefixMap       map[string]string
	rewrites        []config.ModelRewriteRule
	fallback        *config.ModelRewriteRule
}

// NewModelMappingProvider creates a provider that can rewrite and remap models before routing.
func NewModelMappingProvider(defaultProvider domain.Provider, providers map[string]domain.Provider, cfg config.ModelRoutingConfig) (*ModelMappingProvider, error) {
	// Validate that all configured providers exist
	for _, providerName := range cfg.PrefixProviders {
		if _, ok := providers[providerName]; !ok {
			return nil, fmt.Errorf("unknown provider in prefix map: %s", providerName)
		}
	}
	for _, rewrite := range cfg.Rewrites {
		if rewrite.Provider != "" {
			if _, ok := providers[rewrite.Provider]; !ok {
				return nil, fmt.Errorf("unknown provider in rewrite for model %s: %s", rewrite.Match, rewrite.Provider)
			}
		}
	}
	if cfg.Fallback != nil && cfg.Fallback.Provider != "" {
		if _, ok := providers[cfg.Fallback.Provider]; !ok {
			return nil, fmt.Errorf("unknown provider in fallback rewrite: %s", cfg.Fallback.Provider)
		}
	}

	return &ModelMappingProvider{
		defaultProvider: defaultProvider,
		providers:       providers,
		prefixMap:       cfg.PrefixProviders,
		rewrites:        cfg.Rewrites,
		fallback:        cfg.Fallback,
	}, nil
}

func (p *ModelMappingProvider) Name() string {
	return "model-mapper"
}

func (p *ModelMappingProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	provider, mappedReq, err := p.selectProvider(req)
	if err != nil {
		return nil, err
	}
	return provider.Complete(ctx, mappedReq)
}

func (p *ModelMappingProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	provider, mappedReq, err := p.selectProvider(req)
	if err != nil {
		return nil, err
	}
	return provider.Stream(ctx, mappedReq)
}

func (p *ModelMappingProvider) selectProvider(req *domain.CanonicalRequest) (domain.Provider, *domain.CanonicalRequest, error) {
	// Apply explicit rewrite rules first
	for _, rewrite := range p.rewrites {
		if rewrite.Match == req.Model {
			mapped := *req
			if rewrite.Model != "" {
				mapped.Model = rewrite.Model
			}

			if rewrite.Provider != "" {
				if provider, ok := p.providers[rewrite.Provider]; ok {
					return provider, &mapped, nil
				}
				return nil, nil, fmt.Errorf("no provider configured for rewrite: %s", rewrite.Provider)
			}

			return p.defaultProvider, &mapped, nil
		}
	}

	// Apply prefix-based routing (e.g. provider/model)
	if parts := strings.SplitN(req.Model, "/", 2); len(parts) == 2 {
		if providerName, ok := p.prefixMap[parts[0]]; ok {
			provider, ok := p.providers[providerName]
			if !ok {
				return nil, nil, fmt.Errorf("no provider configured for prefix: %s", providerName)
			}

			mapped := *req
			mapped.Model = parts[1]
			return provider, &mapped, nil
		}
	}

	if p.fallback != nil {
		mapped := *req
		if p.fallback.Model != "" {
			mapped.Model = p.fallback.Model
		}

		if p.fallback.Provider != "" {
			if provider, ok := p.providers[p.fallback.Provider]; ok {
				return provider, &mapped, nil
			}
			return nil, nil, fmt.Errorf("no provider configured for fallback: %s", p.fallback.Provider)
		}

		return p.defaultProvider, &mapped, nil
	}

	return p.defaultProvider, req, nil
}
