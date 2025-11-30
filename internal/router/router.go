package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Router chooses a provider for a canonical request based on routing rules.
type Router struct {
	providers       map[string]ports.Provider
	rules           []config.RoutingRule
	defaultProvider string
}

// Decision represents a routing decision.
type Decision struct {
	Provider      ports.Provider
	ProviderName  string
	UpstreamModel string
	// ShouldRewrite controls whether the response model should be rewritten back to the requested model.
	ShouldRewrite bool
}

// New creates a router using the provided providers and routing config.
func New(providers map[string]ports.Provider, routingConfig config.RoutingConfig) *Router {
	return &Router{
		providers:       providers,
		rules:           routingConfig.Rules,
		defaultProvider: routingConfig.DefaultProvider,
	}
}

// Decide returns a routing decision for the request.
func (r *Router) Decide(req *domain.CanonicalRequest) (*Decision, error) {
	provider, name, err := r.routeProvider(req.Model)
	if err != nil {
		return nil, err
	}
	return &Decision{
		Provider:      provider,
		ProviderName:  name,
		UpstreamModel: req.Model,
		ShouldRewrite: false,
	}, nil
}

// CountTokens routes to a provider and delegates CountTokens when supported.
func (r *Router) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	type countTokensProvider interface {
		CountTokens(ctx context.Context, body []byte) ([]byte, error)
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse count_tokens request: %w", err)
	}

	provider, _, err := r.routeProvider(req.Model)
	if err != nil {
		return nil, err
	}

	if ctp, ok := provider.(countTokensProvider); ok {
		return ctp.CountTokens(ctx, body)
	}

	return nil, fmt.Errorf("count_tokens not supported by provider for model %s", req.Model)
}

// routeProvider applies rules to select a provider.
func (r *Router) routeProvider(model string) (ports.Provider, string, error) {
	for _, rule := range r.rules {
		if rule.ModelExact != "" && model == rule.ModelExact {
			if p, ok := r.providers[rule.Provider]; ok {
				return p, rule.Provider, nil
			}
		}
		if rule.ModelPrefix != "" && strings.HasPrefix(model, rule.ModelPrefix) {
			if p, ok := r.providers[rule.Provider]; ok {
				return p, rule.Provider, nil
			}
		}
	}

	if r.defaultProvider != "" {
		if p, ok := r.providers[r.defaultProvider]; ok {
			return p, r.defaultProvider, nil
		}
	}

	return nil, "", fmt.Errorf("no provider configured for model: %s", model)
}
