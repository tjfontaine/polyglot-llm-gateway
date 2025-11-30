package router

import (
	"context"
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// ProviderRouter adapts Router to the ports.Provider interface for backward compatibility.
type ProviderRouter struct {
	core *Router
}

// NewProviderRouter constructs a Provider-compatible router.
func NewProviderRouter(providers map[string]ports.Provider, cfg config.RoutingConfig) *ProviderRouter {
	return &ProviderRouter{core: New(providers, cfg)}
}

func (p *ProviderRouter) Name() string {
	return "router"
}

func (p *ProviderRouter) APIType() domain.APIType {
	if p.core.defaultProvider != "" {
		if prov, ok := p.core.providers[p.core.defaultProvider]; ok {
			return prov.APIType()
		}
	}
	for _, prov := range p.core.providers {
		return prov.APIType()
	}
	return domain.APITypeOpenAI
}

func (p *ProviderRouter) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	decision, err := p.core.Decide(req)
	if err != nil {
		return nil, err
	}
	if decision.Provider == nil {
		return nil, fmt.Errorf("no provider found for model: %s", req.Model)
	}
	return decision.Provider.Complete(ctx, req)
}

func (p *ProviderRouter) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	decision, err := p.core.Decide(req)
	if err != nil {
		return nil, err
	}
	if decision.Provider == nil {
		return nil, fmt.Errorf("no provider found for model: %s", req.Model)
	}
	return decision.Provider.Stream(ctx, req)
}

func (p *ProviderRouter) ListModels(ctx context.Context) (*domain.ModelList, error) {
	if p.core.defaultProvider != "" {
		if prov, ok := p.core.providers[p.core.defaultProvider]; ok {
			return prov.ListModels(ctx)
		}
	}
	for _, prov := range p.core.providers {
		return prov.ListModels(ctx)
	}
	return nil, fmt.Errorf("no provider configured for model listing")
}

// CountTokens delegates to Router.CountTokens when the routed provider supports it.
func (p *ProviderRouter) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	return p.core.CountTokens(ctx, body)
}

// Core returns the underlying Router for advanced usage.
func (p *ProviderRouter) Core() *Router {
	return p.core
}
