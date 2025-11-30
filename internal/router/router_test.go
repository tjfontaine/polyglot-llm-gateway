package router

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

type stubProvider struct {
	name string
}

func (s *stubProvider) Name() string { return s.name }
func (s *stubProvider) APIType() domain.APIType {
	return domain.APITypeOpenAI
}
func (s *stubProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	return &domain.CanonicalResponse{Model: req.Model}, nil
}
func (s *stubProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	ch := make(chan domain.CanonicalEvent)
	close(ch)
	return ch, nil
}
func (s *stubProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{Object: "list", Data: []domain.Model{{ID: "stub"}}}, nil
}

type stubCountTokensProvider struct{ stubProvider }

func (s *stubCountTokensProvider) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	return []byte(`{"count":1}`), nil
}

func TestRouterDecide(t *testing.T) {
	providers := map[string]ports.Provider{
		"openai":    &stubProvider{name: "openai"},
		"anthropic": &stubProvider{name: "anthropic"},
	}
	rt := New(providers, config.RoutingConfig{
		Rules: []config.RoutingRule{
			{ModelPrefix: "claude", Provider: "anthropic"},
		},
		DefaultProvider: "openai",
	})

	tests := []struct {
		name         string
		model        string
		wantProvider string
	}{
		{"prefix match", "claude-3", "anthropic"},
		{"default", "gpt-4o", "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := rt.Decide(&domain.CanonicalRequest{Model: tt.model})
			if err != nil {
				t.Fatalf("Decide() error = %v", err)
			}
			if dec.ProviderName != tt.wantProvider {
				t.Fatalf("provider = %s, want %s", dec.ProviderName, tt.wantProvider)
			}
			if dec.UpstreamModel != tt.model {
				t.Fatalf("upstream model = %s, want %s", dec.UpstreamModel, tt.model)
			}
		})
	}
}

func TestRouterCountTokensDelegates(t *testing.T) {
	providers := map[string]ports.Provider{
		"openai": &stubCountTokensProvider{stubProvider{name: "openai"}},
	}
	rt := New(providers, config.RoutingConfig{
		DefaultProvider: "openai",
	})

	_, err := rt.CountTokens(context.Background(), []byte(`{"model":"gpt-4o"}`))
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}
}

func TestRouterCountTokensUnsupported(t *testing.T) {
	providers := map[string]ports.Provider{
		"openai": &stubProvider{name: "openai"},
	}
	rt := New(providers, config.RoutingConfig{
		DefaultProvider: "openai",
	})

	if _, err := rt.CountTokens(context.Background(), []byte(`{"model":"gpt-4o"}`)); err == nil {
		t.Fatalf("expected error for unsupported count_tokens")
	}
}
