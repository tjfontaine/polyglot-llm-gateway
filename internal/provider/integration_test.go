package provider

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	routerpkg "github.com/tjfontaine/polyglot-llm-gateway/internal/router"
)

// TestIntegration_FullRoutingChain tests the full routing chain from
// ModelMappingProvider -> Router -> specific Provider
func TestIntegration_FullRoutingChain(t *testing.T) {
	// Create mock providers
	openaiCalls := 0
	anthropicCalls := 0

	openaiProvider := &trackingProvider{
		name:    "openai",
		apiType: domain.APITypeOpenAI,
		onComplete: func(req *domain.CanonicalRequest) {
			openaiCalls++
		},
	}
	anthropicProvider := &trackingProvider{
		name:    "anthropic",
		apiType: domain.APITypeAnthropic,
		onComplete: func(req *domain.CanonicalRequest) {
			anthropicCalls++
		},
	}

	providers := map[string]ports.Provider{
		"openai":    openaiProvider,
		"anthropic": anthropicProvider,
	}

	// Create policy router
	router := routerpkg.NewProviderRouter(providers, config.RoutingConfig{
		Rules: []config.RoutingRule{
			{ModelPrefix: "claude", Provider: "anthropic"},
			{ModelPrefix: "gpt", Provider: "openai"},
		},
		DefaultProvider: "openai",
	})

	// Create model mapping provider with rewrites
	mapper, err := NewModelMappingProvider(router, providers, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			// Redirect haiku to openai with a different model
			{
				ModelPrefix:          "claude-haiku",
				Provider:             "openai",
				Model:                "gpt-4o-mini",
				RewriteResponseModel: true,
			},
		},
		// All other requests go to router
	})
	if err != nil {
		t.Fatalf("failed to create mapper: %v", err)
	}

	tests := []struct {
		name              string
		requestModel      string
		expectedProvider  string
		expectedModel     string
		expectedRespModel string
	}{
		{
			name:              "haiku redirected to openai with model rewrite",
			requestModel:      "claude-haiku-20240307",
			expectedProvider:  "openai",
			expectedModel:     "gpt-4o-mini",
			expectedRespModel: "claude-haiku-20240307", // Response model rewritten back
		},
		{
			name:              "sonnet goes through router to anthropic",
			requestModel:      "claude-sonnet-20240307",
			expectedProvider:  "anthropic",
			expectedModel:     "claude-sonnet-20240307",
			expectedRespModel: "claude-sonnet-20240307",
		},
		{
			name:              "gpt goes through router to openai",
			requestModel:      "gpt-4o",
			expectedProvider:  "openai",
			expectedModel:     "gpt-4o",
			expectedRespModel: "gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openaiCalls = 0
			anthropicCalls = 0
			openaiProvider.lastReq = nil
			anthropicProvider.lastReq = nil

			req := &domain.CanonicalRequest{
				Model: tt.requestModel,
				Messages: []domain.Message{
					{Role: "user", Content: "Hello"},
				},
			}

			resp, err := mapper.Complete(context.Background(), req)
			if err != nil {
				t.Fatalf("Complete() error = %v", err)
			}

			// Check which provider was called
			switch tt.expectedProvider {
			case "openai":
				if openaiCalls != 1 {
					t.Errorf("expected 1 openai call, got %d", openaiCalls)
				}
				if anthropicCalls != 0 {
					t.Errorf("expected 0 anthropic calls, got %d", anthropicCalls)
				}
				if openaiProvider.lastReq == nil {
					t.Fatal("openai provider was not called")
				}
				if openaiProvider.lastReq.Model != tt.expectedModel {
					t.Errorf("expected request model %s, got %s", tt.expectedModel, openaiProvider.lastReq.Model)
				}
			case "anthropic":
				if anthropicCalls != 1 {
					t.Errorf("expected 1 anthropic call, got %d", anthropicCalls)
				}
				if openaiCalls != 0 {
					t.Errorf("expected 0 openai calls, got %d", openaiCalls)
				}
				if anthropicProvider.lastReq == nil {
					t.Fatal("anthropic provider was not called")
				}
				if anthropicProvider.lastReq.Model != tt.expectedModel {
					t.Errorf("expected request model %s, got %s", tt.expectedModel, anthropicProvider.lastReq.Model)
				}
			}

			// Check response model
			if resp.Model != tt.expectedRespModel {
				t.Errorf("expected response model %s, got %s", tt.expectedRespModel, resp.Model)
			}
		})
	}
}

// TestIntegration_FallbackChain tests that fallback works when no rewrite matches
func TestIntegration_FallbackChain(t *testing.T) {
	openaiProvider := &trackingProvider{
		name:    "openai",
		apiType: domain.APITypeOpenAI,
	}

	providers := map[string]ports.Provider{
		"openai": openaiProvider,
	}

	// Router with no rules (so everything goes to default)
	router := routerpkg.NewProviderRouter(providers, config.RoutingConfig{
		DefaultProvider: "openai",
	})

	// Model mapper with only a fallback (no specific rewrites)
	mapper, err := NewModelMappingProvider(router, providers, config.ModelRoutingConfig{
		Fallback: &config.ModelRewriteRule{
			Provider:             "openai",
			Model:                "gpt-4o-mini",
			RewriteResponseModel: true,
		},
	})
	if err != nil {
		t.Fatalf("failed to create mapper: %v", err)
	}

	req := &domain.CanonicalRequest{
		Model: "any-random-model",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := mapper.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	// Should have been rewritten to gpt-4o-mini
	if openaiProvider.lastReq == nil {
		t.Fatal("openai provider was not called")
	}
	if openaiProvider.lastReq.Model != "gpt-4o-mini" {
		t.Errorf("expected request model gpt-4o-mini, got %s", openaiProvider.lastReq.Model)
	}

	// Response model should be rewritten back to original
	if resp.Model != "any-random-model" {
		t.Errorf("expected response model any-random-model, got %s", resp.Model)
	}
}

// TestIntegration_StreamingWithRewrite tests that streaming works with model rewrites
func TestIntegration_StreamingWithRewrite(t *testing.T) {
	openaiProvider := &trackingProvider{
		name:          "openai",
		apiType:       domain.APITypeOpenAI,
		streamContent: []string{"Hello", " ", "World"},
	}

	providers := map[string]ports.Provider{
		"openai": openaiProvider,
	}

	router := routerpkg.NewProviderRouter(providers, config.RoutingConfig{
		DefaultProvider: "openai",
	})

	mapper, err := NewModelMappingProvider(router, providers, config.ModelRoutingConfig{
		Rewrites: []config.ModelRewriteRule{
			{
				ModelExact: "alias-model",
				Provider:   "openai",
				Model:      "gpt-4o",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create mapper: %v", err)
	}

	req := &domain.CanonicalRequest{
		Model: "alias-model",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	events, err := mapper.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	// Collect all events
	var content string
	for event := range events {
		if event.Error != nil {
			t.Fatalf("stream error: %v", event.Error)
		}
		content += event.ContentDelta
	}

	if content != "Hello World" {
		t.Errorf("expected content 'Hello World', got '%s'", content)
	}

	// Verify the model was rewritten
	if openaiProvider.lastReq == nil {
		t.Fatal("openai provider was not called")
	}
	if openaiProvider.lastReq.Model != "gpt-4o" {
		t.Errorf("expected request model gpt-4o, got %s", openaiProvider.lastReq.Model)
	}
}

// trackingProvider is a test provider that tracks calls
type trackingProvider struct {
	name          string
	apiType       domain.APIType
	lastReq       *domain.CanonicalRequest
	onComplete    func(*domain.CanonicalRequest)
	streamContent []string
}

func (p *trackingProvider) Name() string            { return p.name }
func (p *trackingProvider) APIType() domain.APIType { return p.apiType }

func (p *trackingProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	p.lastReq = req
	if p.onComplete != nil {
		p.onComplete(req)
	}
	return &domain.CanonicalResponse{
		Model: req.Model,
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "Response from " + p.name}},
		},
	}, nil
}

func (p *trackingProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	p.lastReq = req
	ch := make(chan domain.CanonicalEvent)
	go func() {
		defer close(ch)
		for _, content := range p.streamContent {
			ch <- domain.CanonicalEvent{ContentDelta: content}
		}
	}()
	return ch, nil
}

func (p *trackingProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{Object: "list"}, nil
}
