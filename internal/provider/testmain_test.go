package provider

import (
	"context"
	"os"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

type stubProvider struct {
	name    string
	apiType domain.APIType
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) APIType() domain.APIType { return s.apiType }

func (s *stubProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	return &domain.CanonicalResponse{Model: req.Model}, nil
}

func (s *stubProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	ch := make(chan domain.CanonicalEvent)
	close(ch)
	return ch, nil
}

func (s *stubProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{
		Object: "list",
		Data:   []domain.Model{{ID: "stub", Object: "model", OwnedBy: s.name}},
	}, nil
}

func TestMain(m *testing.M) {
	ClearFactories()
	// Register minimal stub factories to satisfy provider tests without pulling in full dependencies.
	RegisterFactory(ProviderFactory{
		Type:        "openai",
		APIType:     domain.APITypeOpenAI,
		Description: "stub openai",
		Create: func(cfg config.ProviderConfig) (ports.Provider, error) {
			return &stubProvider{name: cfg.Name, apiType: domain.APITypeOpenAI}, nil
		},
		ValidateConfig: func(cfg config.ProviderConfig) error { return nil },
	})
	RegisterFactory(ProviderFactory{
		Type:        "openai-compatible",
		APIType:     domain.APITypeOpenAI,
		Description: "stub openai-compatible",
		Create: func(cfg config.ProviderConfig) (ports.Provider, error) {
			return &stubProvider{name: cfg.Name, apiType: domain.APITypeOpenAI}, nil
		},
		ValidateConfig: func(cfg config.ProviderConfig) error { return nil },
	})
	RegisterFactory(ProviderFactory{
		Type:        "anthropic",
		APIType:     domain.APITypeAnthropic,
		Description: "stub anthropic",
		Create: func(cfg config.ProviderConfig) (ports.Provider, error) {
			return &stubProvider{name: cfg.Name, apiType: domain.APITypeAnthropic}, nil
		},
		ValidateConfig: func(cfg config.ProviderConfig) error { return nil },
	})
	os.Exit(m.Run())
}
