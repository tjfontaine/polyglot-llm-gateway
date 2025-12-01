package router

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
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

type recordingCountTokensProvider struct {
	recordingProvider
	lastCountBody []byte
}

func (p *recordingCountTokensProvider) CountTokens(_ context.Context, body []byte) ([]byte, error) {
	p.lastCountBody = body
	return []byte(`{"count":100}`), nil
}

func TestNewMappingProvider(t *testing.T) {
	defaultProvider := &recordingProvider{name: "default"}
	providers := map[string]ports.Provider{
		"openai": &recordingProvider{name: "openai"},
	}

	t.Run("valid configuration", func(t *testing.T) {
		cfg := config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{ModelExact: "test-model", Provider: "openai", Model: "gpt-4"},
			},
		}

		mapper, err := NewMappingProvider(defaultProvider, providers, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mapper == nil {
			t.Fatal("expected mapper, got nil")
		}
	})

	t.Run("unknown provider in rewrite", func(t *testing.T) {
		cfg := config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{ModelExact: "test-model", Provider: "unknown-provider", Model: "gpt-4"},
			},
		}

		_, err := NewMappingProvider(defaultProvider, providers, cfg)
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})

	t.Run("unknown provider in prefix map", func(t *testing.T) {
		cfg := config.ModelRoutingConfig{
			PrefixProviders: map[string]string{
				"azure": "unknown-provider",
			},
		}

		_, err := NewMappingProvider(defaultProvider, providers, cfg)
		if err == nil {
			t.Fatal("expected error for unknown provider in prefix map")
		}
	})

	t.Run("unknown provider in fallback", func(t *testing.T) {
		cfg := config.ModelRoutingConfig{
			Fallback: &config.ModelRewriteRule{
				Provider: "unknown-provider",
				Model:    "fallback-model",
			},
		}

		_, err := NewMappingProvider(defaultProvider, providers, cfg)
		if err == nil {
			t.Fatal("expected error for unknown provider in fallback")
		}
	})
}

func TestMappingProvider_Name(t *testing.T) {
	mapper, _ := NewMappingProvider(
		&recordingProvider{name: "default"},
		nil,
		config.ModelRoutingConfig{},
	)
	if got := mapper.Name(); got != "model-mapper" {
		t.Errorf("Name() = %q, want %q", got, "model-mapper")
	}
}

func TestMappingProvider_APIType(t *testing.T) {
	defaultProvider := &recordingProvider{name: "default"}
	mapper, _ := NewMappingProvider(defaultProvider, nil, config.ModelRoutingConfig{})
	if got := mapper.APIType(); got != domain.APITypeOpenAI {
		t.Errorf("APIType() = %v, want %v", got, domain.APITypeOpenAI)
	}
}

func TestMappingProvider_Complete(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	defaultProvider := &recordingProvider{name: "default"}

	t.Run("exact match rewrite", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": openai,
		}, config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{ModelExact: "alias-model", Provider: "openai", Model: "gpt-4"},
			},
		})

		_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "alias-model"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if openai.lastReq == nil || openai.lastReq.Model != "gpt-4" {
			t.Fatalf("expected model gpt-4, got %v", openai.lastReq)
		}
	})

	t.Run("prefix match rewrite", func(t *testing.T) {
		openai.lastReq = nil
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": openai,
		}, config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{ModelPrefix: "claude-", Provider: "openai", Model: "gpt-4o-mini"},
			},
		})

		_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "claude-sonnet"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if openai.lastReq == nil || openai.lastReq.Model != "gpt-4o-mini" {
			t.Fatalf("expected model gpt-4o-mini, got %v", openai.lastReq)
		}
	})

	t.Run("slash prefix routing", func(t *testing.T) {
		openai.lastReq = nil
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": openai,
		}, config.ModelRoutingConfig{
			PrefixProviders: map[string]string{
				"openai": "openai",
			},
		})

		_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "openai/gpt-4o"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if openai.lastReq == nil || openai.lastReq.Model != "gpt-4o" {
			t.Fatalf("expected model gpt-4o (stripped prefix), got %v", openai.lastReq)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		openai.lastReq = nil
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": openai,
		}, config.ModelRoutingConfig{
			Fallback: &config.ModelRewriteRule{
				Provider: "openai",
				Model:    "fallback-model",
			},
		})

		_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "unknown-model"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if openai.lastReq == nil || openai.lastReq.Model != "fallback-model" {
			t.Fatalf("expected model fallback-model, got %v", openai.lastReq)
		}
	})

	t.Run("default provider", func(t *testing.T) {
		defaultProvider.lastReq = nil
		mapper, _ := NewMappingProvider(defaultProvider, nil, config.ModelRoutingConfig{})

		_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "some-model"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if defaultProvider.lastReq == nil || defaultProvider.lastReq.Model != "some-model" {
			t.Fatalf("expected model some-model on default provider, got %v", defaultProvider.lastReq)
		}
	})

	t.Run("response model rewrite", func(t *testing.T) {
		openai.lastReq = nil
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": openai,
		}, config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{
					ModelExact:           "alias-model",
					Provider:             "openai",
					Model:                "gpt-4",
					RewriteResponseModel: true,
				},
			},
		})

		resp, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "alias-model"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The response model should be rewritten back to the original request model
		if resp.Model != "alias-model" {
			t.Fatalf("expected response model alias-model, got %q", resp.Model)
		}
	})

	t.Run("nil request error", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, nil, config.ModelRoutingConfig{})

		_, err := mapper.Complete(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("empty model error", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, nil, config.ModelRoutingConfig{})

		_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: ""})
		if err == nil {
			t.Fatal("expected error for empty model")
		}
	})
}

func TestMappingProvider_Stream(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	defaultProvider := &recordingProvider{name: "default"}

	t.Run("routes to correct provider", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": openai,
		}, config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{ModelExact: "alias-model", Provider: "openai", Model: "gpt-4"},
			},
		})

		ch, err := mapper.Stream(context.Background(), &domain.CanonicalRequest{Model: "alias-model"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Drain the channel
		for range ch {
		}
		if openai.lastReq == nil || openai.lastReq.Model != "gpt-4" {
			t.Fatalf("expected model gpt-4, got %v", openai.lastReq)
		}
	})
}

func TestMappingProvider_ListModels(t *testing.T) {
	defaultProvider := &recordingProvider{name: "default"}

	t.Run("delegates to default provider", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, nil, config.ModelRoutingConfig{})

		models, err := mapper.ListModels(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models.Data) != 1 || models.Data[0].ID != "default" {
			t.Fatalf("expected default model list, got %v", models)
		}
	})
}

func TestMappingProvider_CountTokens(t *testing.T) {
	countProvider := &recordingCountTokensProvider{
		recordingProvider: recordingProvider{name: "openai"},
	}
	defaultProvider := &recordingProvider{name: "default"}

	t.Run("delegates to provider with count support", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
			"openai": countProvider,
		}, config.ModelRoutingConfig{
			Rewrites: []config.ModelRewriteRule{
				{ModelExact: "alias-model", Provider: "openai", Model: "gpt-4"},
			},
		})

		body := []byte(`{"model":"alias-model"}`)
		result, err := mapper.CountTokens(context.Background(), body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != `{"count":100}` {
			t.Fatalf("expected count result, got %s", result)
		}
	})

	t.Run("errors when provider doesn't support count", func(t *testing.T) {
		mapper, _ := NewMappingProvider(defaultProvider, nil, config.ModelRoutingConfig{})

		body := []byte(`{"model":"some-model"}`)
		_, err := mapper.CountTokens(context.Background(), body)
		if err == nil {
			t.Fatal("expected error for unsupported count")
		}
	})
}

func TestMappingProvider_MatchContains(t *testing.T) {
	openai := &recordingProvider{name: "openai"}
	defaultProvider := &recordingProvider{name: "default"}

	mapper, _ := NewMappingProvider(defaultProvider, map[string]ports.Provider{
		"openai": openai,
	}, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			{Match: "turbo", Provider: "openai", Model: "gpt-4-turbo"},
		},
	})

	_, err := mapper.Complete(context.Background(), &domain.CanonicalRequest{Model: "gpt-3.5-turbo-0125"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openai.lastReq == nil || openai.lastReq.Model != "gpt-4-turbo" {
		t.Fatalf("expected model gpt-4-turbo, got %v", openai.lastReq)
	}
}

func TestExtractModelFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{"valid JSON", `{"model":"gpt-4"}`, "gpt-4"},
		{"no model field", `{"other":"value"}`, ""},
		{"invalid JSON", `not json`, ""},
		{"empty body", ``, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractModelFromBody([]byte(tt.body))
			if got != tt.expected {
				t.Errorf("extractModelFromBody() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetProviderName(t *testing.T) {
	mapper := &MappingProvider{}

	tests := []struct {
		model    string
		expected string
	}{
		{"openai/gpt-4", "openai"},
		{"anthropic/claude-3", "anthropic"},
		{"gpt-4", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := mapper.getProviderName(tt.model)
			if got != tt.expected {
				t.Errorf("getProviderName(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}

func TestStripProviderPrefix(t *testing.T) {
	mapper := &MappingProvider{}

	tests := []struct {
		model    string
		expected string
	}{
		{"openai/gpt-4", "gpt-4"},
		{"anthropic/claude-3/opus", "claude-3/opus"},
		{"gpt-4", "gpt-4"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := mapper.stripProviderPrefix(tt.model)
			if got != tt.expected {
				t.Errorf("stripProviderPrefix(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}
