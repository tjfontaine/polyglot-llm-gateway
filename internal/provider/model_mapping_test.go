package provider

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

type recordingProvider struct {
	name    string
	lastReq *domain.CanonicalRequest
}

func (p *recordingProvider) Name() string            { return p.name }
func (p *recordingProvider) APIType() domain.APIType { return domain.APITypeOpenAI }

func (p *recordingProvider) Complete(_ context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	p.lastReq = req
	return &domain.CanonicalResponse{Model: req.Model}, nil
}

func (p *recordingProvider) Stream(_ context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	p.lastReq = req
	ch := make(chan domain.CanonicalEvent)
	close(ch)
	return ch, nil
}

func (p *recordingProvider) ListModels(_ context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{Object: "list", Data: []domain.Model{{ID: p.name}}}, nil
}

func TestModelMappingProviderRewrite(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai": openai,
	}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			{ModelExact: "claude-haiku-4.5", Provider: "openai", Model: "gpt-5-mini"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-haiku-4.5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if openai.lastReq == nil || openai.lastReq.Model != "gpt-5-mini" {
		t.Fatalf("expected rewrite to gpt-5-mini on openai, got %#v", openai.lastReq)
	}
}

func TestModelMappingProviderPrefixRewrite(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai": openai,
	}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			{ModelPrefix: "claude-", Provider: "openai", Model: "gpt-5-mini"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-haiku-4.5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if openai.lastReq == nil || openai.lastReq.Model != "gpt-5-mini" {
		t.Fatalf("expected prefix rewrite to gpt-5-mini on openai, got %#v", openai.lastReq)
	}
}

func TestModelMappingProviderPrefixRouting(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai": openai,
	}, config.ModelRoutingConfig{
		PrefixProviders: map[string]string{
			"openai": "openai",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "openai/gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if openai.lastReq == nil || openai.lastReq.Model != "gpt-4o" {
		t.Fatalf("expected prefix mapping to strip provider: %#v", openai.lastReq)
	}
}

func TestModelMappingProviderFallbackRewrite(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai": openai,
	}, config.ModelRoutingConfig{
		Fallback: &config.ModelRewriteRule{
			Provider: "openai",
			Model:    "gpt-5-mini",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if openai.lastReq == nil || openai.lastReq.Model != "gpt-5-mini" {
		t.Fatalf("expected fallback rewrite to gpt-5-mini on openai, got %#v", openai.lastReq)
	}
}

func TestModelMappingProviderFallback(t *testing.T) {
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{}, config.ModelRoutingConfig{})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if router.lastReq == nil || router.lastReq.Model != "gpt-4o" {
		t.Fatalf("expected fallback to default provider: %#v", router.lastReq)
	}
}

func TestModelMappingProviderResponseRewrite(t *testing.T) {
	upstream := &recordingProvider{name: "openai"}
	mapper, err := NewModelMappingProvider(upstream, map[string]domain.Provider{}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{{
			ModelExact:           "alias-model",
			Model:                "provider-model",
			RewriteResponseModel: true,
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	resp, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "alias-model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Model != "alias-model" {
		t.Fatalf("expected response model to be rewritten, got %s", resp.Model)
	}
}

func TestModelMappingProviderListModelsRewrite(t *testing.T) {
	upstream := &recordingProvider{name: "provider-model"}
	mapper, err := NewModelMappingProvider(upstream, map[string]domain.Provider{}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{{
			ModelExact:           "alias-model",
			Model:                "provider-model",
			RewriteResponseModel: true,
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	list, err := mapper.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing models: %v", err)
	}

	if len(list.Data) != 1 || list.Data[0].ID != "alias-model" {
		t.Fatalf("expected rewritten model id, got %#v", list.Data)
	}
}
