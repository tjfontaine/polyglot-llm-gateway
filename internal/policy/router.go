package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

type Router struct {
	providers       map[string]domain.Provider
	rules           []config.RoutingRule
	defaultProvider string
}

func NewRouter(providers map[string]domain.Provider, routingConfig config.RoutingConfig) *Router {
	return &Router{
		providers:       providers,
		rules:           routingConfig.Rules,
		defaultProvider: routingConfig.DefaultProvider,
	}
}

func (r *Router) Name() string {
	return "router"
}

// APIType returns the API type of the default provider.
// Since the router can route to multiple providers, this returns the default.
func (r *Router) APIType() domain.APIType {
	if r.defaultProvider != "" {
		if p, ok := r.providers[r.defaultProvider]; ok {
			return p.APIType()
		}
	}
	// Fall back to first available provider
	for _, p := range r.providers {
		return p.APIType()
	}
	return domain.APITypeOpenAI // Default
}

func (r *Router) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	p, err := r.Route(req)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("no provider found for model: %s", req.Model)
	}
	return p.Complete(ctx, req)
}

func (r *Router) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	p, err := r.Route(req)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("no provider found for model: %s", req.Model)
	}
	return p.Stream(ctx, req)
}

func (r *Router) ListModels(ctx context.Context) (*domain.ModelList, error) {
	if r.defaultProvider != "" {
		if p, ok := r.providers[r.defaultProvider]; ok {
			return p.ListModels(ctx)
		}
	}

	for _, p := range r.providers {
		return p.ListModels(ctx)
	}

	return nil, fmt.Errorf("no provider configured for model listing")
}

func (r *Router) Route(req *domain.CanonicalRequest) (domain.Provider, error) {
	// Apply routing rules in order
	for _, rule := range r.rules {
		if rule.ModelExact != "" && req.Model == rule.ModelExact {
			if p, ok := r.providers[rule.Provider]; ok {
				return p, nil
			}
		}
		if rule.ModelPrefix != "" && strings.HasPrefix(req.Model, rule.ModelPrefix) {
			if p, ok := r.providers[rule.Provider]; ok {
				return p, nil
			}
		}
	}

	// Fall back to default provider
	if r.defaultProvider != "" {
		if p, ok := r.providers[r.defaultProvider]; ok {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no provider configured for model: %s", req.Model)
}

// CountTokens routes to the appropriate provider based on the model in the request,
// then delegates if that provider supports the CountTokens interface.
func (r *Router) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	type countTokensProvider interface {
		CountTokens(ctx context.Context, body []byte) ([]byte, error)
	}

	// Parse the model from the request body to determine routing
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse count_tokens request: %w", err)
	}

	// Use the same routing logic as Complete
	canonReq := &domain.CanonicalRequest{Model: req.Model}
	provider, err := r.Route(canonReq)
	if err != nil {
		return nil, err
	}

	// Check if the routed provider supports CountTokens
	if ctp, ok := provider.(countTokensProvider); ok {
		return ctp.CountTokens(ctx, body)
	}

	// Return error so caller can fall back to estimation
	return nil, fmt.Errorf("count_tokens not supported by provider for model %s", req.Model)
}
