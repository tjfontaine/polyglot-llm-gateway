package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
)

type Router struct {
	providers map[string]domain.Provider
}

func NewRouter(providers ...domain.Provider) *Router {
	pMap := make(map[string]domain.Provider)
	for _, p := range providers {
		pMap[p.Name()] = p
	}
	return &Router{
		providers: pMap,
	}
}

func (r *Router) Name() string {
	return "router"
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

func (r *Router) Route(req *domain.CanonicalRequest) (domain.Provider, error) {
	// Simple routing logic:
	// If model starts with "claude", route to anthropic.
	// Otherwise, route to openai.
	// In a real system, this would be more sophisticated (config-driven, etc.)

	if strings.HasPrefix(req.Model, "claude") {
		if p, ok := r.providers["anthropic"]; ok {
			return p, nil
		}
	}

	// Default to openai
	if p, ok := r.providers["openai"]; ok {
		return p, nil
	}

	return nil, fmt.Errorf("no default provider configured")
}
