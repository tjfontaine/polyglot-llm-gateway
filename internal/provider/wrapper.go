package provider

import (
	"context"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// ModelOverrideProvider wraps a provider and overrides the model in requests
type ModelOverrideProvider struct {
	inner ports.Provider
	model string
}

// NewModelOverrideProvider creates a new ModelOverrideProvider
func NewModelOverrideProvider(inner ports.Provider, model string) *ModelOverrideProvider {
	return &ModelOverrideProvider{
		inner: inner,
		model: model,
	}
}

func (p *ModelOverrideProvider) Name() string {
	return p.inner.Name()
}

func (p *ModelOverrideProvider) APIType() domain.APIType {
	return p.inner.APIType()
}

func (p *ModelOverrideProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	// Clone request to avoid side effects
	newReq := *req
	newReq.Model = p.model
	return p.inner.Complete(ctx, &newReq)
}

func (p *ModelOverrideProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	// Clone request to avoid side effects
	newReq := *req
	newReq.Model = p.model
	return p.inner.Stream(ctx, &newReq)
}

func (p *ModelOverrideProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return p.inner.ListModels(ctx)
}
