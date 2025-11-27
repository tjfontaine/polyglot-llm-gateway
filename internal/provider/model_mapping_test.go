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

// Test that fallback-only config creates working ModelMappingProvider
func TestModelMappingProviderFallbackOnly(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai": openai,
	}, config.ModelRoutingConfig{
		// Only fallback, no rewrites
		Fallback: &config.ModelRewriteRule{
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	// Any model should fallback to openai with gpt-4o-mini
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "any-unknown-model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if openai.lastReq == nil {
		t.Fatal("request was not routed to openai")
	}
	if openai.lastReq.Model != "gpt-4o-mini" {
		t.Fatalf("expected fallback rewrite to gpt-4o-mini, got %s", openai.lastReq.Model)
	}
}

// Test multiple prefix rules with different providers
func TestModelMappingProviderMultiplePrefixRules(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	anthropic := &recordingProvider{name: "anthropic"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai":    openai,
		"anthropic": anthropic,
	}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			{ModelPrefix: "claude-", Provider: "anthropic"},
			{ModelPrefix: "gpt-", Provider: "openai"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	// Test claude model goes to anthropic
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-3-haiku-20240307"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anthropic.lastReq == nil {
		t.Fatal("claude request was not routed to anthropic")
	}
	if anthropic.lastReq.Model != "claude-3-haiku-20240307" {
		t.Fatalf("expected model unchanged, got %s", anthropic.lastReq.Model)
	}

	// Test gpt model goes to openai
	anthropic.lastReq = nil
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openai.lastReq == nil {
		t.Fatal("gpt request was not routed to openai")
	}
	if openai.lastReq.Model != "gpt-4o" {
		t.Fatalf("expected model unchanged, got %s", openai.lastReq.Model)
	}
}

// Test exact match takes precedence over prefix
func TestModelMappingProviderExactMatchPrecedence(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	anthropic := &recordingProvider{name: "anthropic"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai":    openai,
		"anthropic": anthropic,
	}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			// Exact match should take precedence
			{ModelExact: "claude-haiku", Provider: "openai", Model: "gpt-4o"},
			// Prefix match for other claude models
			{ModelPrefix: "claude-", Provider: "anthropic"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	// Exact match should go to openai with rewritten model
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-haiku"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openai.lastReq == nil {
		t.Fatal("exact match was not routed to openai")
	}
	if openai.lastReq.Model != "gpt-4o" {
		t.Fatalf("expected model rewritten to gpt-4o, got %s", openai.lastReq.Model)
	}

	// Other claude models should go to anthropic
	openai.lastReq = nil
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-sonnet"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anthropic.lastReq == nil {
		t.Fatal("prefix match was not routed to anthropic")
	}
}

// Test response model rewriting
func TestModelMappingProviderResponseModelRewrite(t *testing.T) {
	upstream := &recordingProvider{name: "upstream"}
	mapper, err := NewModelMappingProvider(upstream, map[string]domain.Provider{}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{{
			ModelPrefix:          "alias-",
			Model:                "real-model",
			RewriteResponseModel: true,
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	resp, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "alias-v1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Response model should be the original requested model
	if resp.Model != "alias-v1" {
		t.Fatalf("expected response model to be alias-v1, got %s", resp.Model)
	}

	// But the actual request should use real-model
	if upstream.lastReq.Model != "real-model" {
		t.Fatalf("expected request model to be real-model, got %s", upstream.lastReq.Model)
	}
}

// Test CountTokens delegation
func TestModelMappingProviderCountTokens(t *testing.T) {
	upstream := &countableProvider{
		recordingProvider: recordingProvider{name: "upstream"},
		countResult:       []byte(`{"input_tokens": 42}`),
	}
	mapper, err := NewModelMappingProvider(upstream, map[string]domain.Provider{}, config.ModelRoutingConfig{})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	result, err := mapper.CountTokens(context.Background(), []byte(`{"model":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `{"input_tokens": 42}` {
		t.Fatalf("expected delegated count_tokens result, got %s", string(result))
	}
}

// Test CountTokens returns error when provider doesn't support it
func TestModelMappingProviderCountTokensUnsupported(t *testing.T) {
	upstream := &recordingProvider{name: "upstream"} // doesn't implement CountTokens
	mapper, err := NewModelMappingProvider(upstream, map[string]domain.Provider{}, config.ModelRoutingConfig{})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	_, err = mapper.CountTokens(context.Background(), []byte(`{"model":"test"}`))
	if err == nil {
		t.Fatal("expected error for unsupported count_tokens")
	}
}

// Helper type that implements CountTokens
type countableProvider struct {
	recordingProvider
	countResult []byte
}

func (p *countableProvider) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	return p.countResult, nil
}

// Test prefix/provider routing (e.g., "openai/gpt-4o" -> openai provider with "gpt-4o")
func TestModelMappingProviderSlashPrefixRouting(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	anthropic := &recordingProvider{name: "anthropic"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai":    openai,
		"anthropic": anthropic,
	}, config.ModelRoutingConfig{
		PrefixProviders: map[string]string{
			"openai":    "openai",
			"anthropic": "anthropic",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	// Test "openai/gpt-4o" goes to openai with model "gpt-4o"
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "openai/gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openai.lastReq == nil {
		t.Fatal("request was not routed to openai")
	}
	if openai.lastReq.Model != "gpt-4o" {
		t.Fatalf("expected model stripped to gpt-4o, got %s", openai.lastReq.Model)
	}

	// Test "anthropic/claude-3" goes to anthropic with model "claude-3"
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "anthropic/claude-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anthropic.lastReq == nil {
		t.Fatal("request was not routed to anthropic")
	}
	if anthropic.lastReq.Model != "claude-3" {
		t.Fatalf("expected model stripped to claude-3, got %s", anthropic.lastReq.Model)
	}
}

// Test combined rewrites, prefix providers, and fallback
func TestModelMappingProviderCombinedConfig(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	anthropic := &recordingProvider{name: "anthropic"}
	router := &recordingProvider{name: "router"}

	mapper, err := NewModelMappingProvider(router, map[string]domain.Provider{
		"openai":    openai,
		"anthropic": anthropic,
	}, config.ModelRoutingConfig{
		// Explicit rewrites take precedence
		Rewrites: []config.ModelRewriteRule{
			{ModelExact: "alias-model", Provider: "anthropic", Model: "claude-3-haiku"},
		},
		// Then prefix providers
		PrefixProviders: map[string]string{
			"openai": "openai",
		},
		// Finally fallback
		Fallback: &config.ModelRewriteRule{
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating mapper: %v", err)
	}

	// 1. Exact rewrite matches first
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "alias-model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anthropic.lastReq == nil || anthropic.lastReq.Model != "claude-3-haiku" {
		t.Fatal("exact rewrite did not work")
	}
	anthropic.lastReq = nil

	// 2. Prefix provider matches second
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "openai/gpt-4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openai.lastReq == nil || openai.lastReq.Model != "gpt-4" {
		t.Fatal("prefix provider did not work")
	}
	openai.lastReq = nil

	// 3. Fallback matches last
	_, err = mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "unknown-model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openai.lastReq == nil || openai.lastReq.Model != "gpt-4o-mini" {
		t.Fatalf("fallback did not work, got %v", openai.lastReq)
	}
}
