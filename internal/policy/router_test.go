package policy

import (
	"context"
	"testing"

	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
)

// mockProvider implements domain.Provider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	return &domain.CanonicalResponse{
		ID:    "test-id",
		Model: req.Model,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    "assistant",
					Content: "test response from " + m.name,
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	ch := make(chan domain.CanonicalEvent, 1)
	ch <- domain.CanonicalEvent{
		ContentDelta: "test stream from " + m.name,
	}
	close(ch)
	return ch, nil
}

func TestRouter_Route(t *testing.T) {
	// Create mock providers
	openaiProvider := &mockProvider{name: "openai"}
	anthropicProvider := &mockProvider{name: "anthropic"}

	providers := map[string]domain.Provider{
		"openai":    openaiProvider,
		"anthropic": anthropicProvider,
	}

	routingConfig := config.RoutingConfig{
		Rules: []config.RoutingRule{
			{ModelPrefix: "claude", Provider: "anthropic"},
			{ModelPrefix: "gpt", Provider: "openai"},
			{ModelExact: "gemini-pro", Provider: "openai"},
		},
		DefaultProvider: "openai",
	}

	router := NewRouter(providers, routingConfig)

	tests := []struct {
		name         string
		model        string
		wantProvider string
		wantError    bool
	}{
		{
			name:         "claude model routes to anthropic",
			model:        "claude-3-sonnet",
			wantProvider: "anthropic",
			wantError:    false,
		},
		{
			name:         "gpt model routes to openai",
			model:        "gpt-4",
			wantProvider: "openai",
			wantError:    false,
		},
		{
			name:         "exact match takes precedence",
			model:        "gemini-pro",
			wantProvider: "openai",
			wantError:    false,
		},
		{
			name:         "unknown model uses default",
			model:        "unknown-model",
			wantProvider: "openai",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &domain.CanonicalRequest{
				Model: tt.model,
			}

			provider, err := router.Route(req)

			if tt.wantError {
				if err == nil {
					t.Error("Route() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Route() unexpected error: %v", err)
				return
			}

			if provider.Name() != tt.wantProvider {
				t.Errorf("Route() provider = %v, want %v", provider.Name(), tt.wantProvider)
			}
		})
	}
}

func TestRouter_Complete(t *testing.T) {
	openaiProvider := &mockProvider{name: "openai"}
	anthropicProvider := &mockProvider{name: "anthropic"}

	providers := map[string]domain.Provider{
		"openai":    openaiProvider,
		"anthropic": anthropicProvider,
	}

	routingConfig := config.RoutingConfig{
		Rules: []config.RoutingRule{
			{ModelPrefix: "claude", Provider: "anthropic"},
		},
		DefaultProvider: "openai",
	}

	router := NewRouter(providers, routingConfig)

	tests := []struct {
		name              string
		model             string
		expectedInContent string
	}{
		{
			name:              "routes to openai",
			model:             "gpt-4",
			expectedInContent: "openai",
		},
		{
			name:              "routes to anthropic",
			model:             "claude-3",
			expectedInContent: "anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &domain.CanonicalRequest{
				Model: tt.model,
				Messages: []domain.Message{
					{Role: "user", Content: "test"},
				},
			}

			resp, err := router.Complete(context.Background(), req)
			if err != nil {
				t.Fatalf("Complete() error = %v", err)
			}

			if resp.Model != tt.model {
				t.Errorf("Complete() model = %v, want %v", resp.Model, tt.model)
			}

			if len(resp.Choices) == 0 {
				t.Fatal("Complete() returned no choices")
			}

			content := resp.Choices[0].Message.Content
			if len(content) == 0 {
				t.Error("Complete() returned empty content")
			}
		})
	}
}

func TestRouter_Stream(t *testing.T) {
	openaiProvider := &mockProvider{name: "openai"}

	providers := map[string]domain.Provider{
		"openai": openaiProvider,
	}

	routingConfig := config.RoutingConfig{
		DefaultProvider: "openai",
	}

	router := NewRouter(providers, routingConfig)

	req := &domain.CanonicalRequest{
		Model: "gpt-4",
		Messages: []domain.Message{
			{Role: "user", Content: "test"},
		},
	}

	events, err := router.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	// Read from channel
	eventCount := 0
	for event := range events {
		eventCount++
		if event.Error != nil {
			t.Errorf("Stream() event error = %v", event.Error)
		}
		if event.ContentDelta == "" {
			t.Error("Stream() event has empty content delta")
		}
	}

	if eventCount == 0 {
		t.Error("Stream() returned no events")
	}
}
