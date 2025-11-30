package frontdoor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// mockProvider implements ports.Provider for testing
type mockProvider struct {
	name              string
	apiType           domain.APIType
	lastReq           *domain.CanonicalRequest
	response          *domain.CanonicalResponse
	countTokensResult []byte
}

func (m *mockProvider) Name() string            { return m.name }
func (m *mockProvider) APIType() domain.APIType { return m.apiType }

func (m *mockProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	m.lastReq = req
	if m.response != nil {
		return m.response, nil
	}
	return &domain.CanonicalResponse{
		Model: req.Model,
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "Test response"}},
		},
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	m.lastReq = req
	ch := make(chan domain.CanonicalEvent)
	close(ch)
	return ch, nil
}

func (m *mockProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{
		Object: "list",
		Data:   []domain.Model{{ID: "test-model", Object: "model"}},
	}, nil
}

func (m *mockProvider) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	if m.countTokensResult != nil {
		return m.countTokensResult, nil
	}
	return []byte(`{"input_tokens": 10}`), nil
}

func TestRegistry_CreateHandlers_WithModelRouting(t *testing.T) {
	registry := NewRegistry()

	openaiProvider := &mockProvider{name: "openai", apiType: domain.APITypeOpenAI}
	anthropicProvider := &mockProvider{name: "anthropic", apiType: domain.APITypeAnthropic}
	routerProvider := &mockProvider{name: "router", apiType: domain.APITypeOpenAI}

	providers := map[string]ports.Provider{
		"openai":    openaiProvider,
		"anthropic": anthropicProvider,
	}

	configs := []config.AppConfig{
		{
			Name:      "test-app",
			Frontdoor: "anthropic",
			Path:      "/test",
			ModelRouting: config.ModelRoutingConfig{
				Rewrites: []config.ModelRewriteRule{
					{
						ModelPrefix: "claude-",
						Provider:    "openai",
						Model:       "gpt-4o",
					},
				},
			},
		},
	}

	handlers, err := registry.CreateHandlers(configs, routerProvider, providers, nil)
	if err != nil {
		t.Fatalf("CreateHandlers failed: %v", err)
	}

	if len(handlers) != 3 { // messages, count_tokens, models
		t.Errorf("expected 3 handlers, got %d", len(handlers))
	}

	// Verify routes were registered
	expectedPaths := []string{"/test/v1/messages", "/test/v1/messages/count_tokens", "/test/v1/models"}
	for _, expected := range expectedPaths {
		found := false
		for _, h := range handlers {
			if h.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected handler for path %s", expected)
		}
	}
}

func TestRegistry_CreateHandlers_WithFallback(t *testing.T) {
	registry := NewRegistry()

	openaiProvider := &mockProvider{name: "openai", apiType: domain.APITypeOpenAI}
	anthropicProvider := &mockProvider{name: "anthropic", apiType: domain.APITypeAnthropic}
	routerProvider := &mockProvider{name: "router", apiType: domain.APITypeOpenAI}

	providers := map[string]ports.Provider{
		"openai":    openaiProvider,
		"anthropic": anthropicProvider,
	}

	// This config only has a fallback, no rewrites
	configs := []config.AppConfig{
		{
			Name:      "test-app",
			Frontdoor: "anthropic",
			Path:      "/test",
			ModelRouting: config.ModelRoutingConfig{
				Fallback: &config.ModelRewriteRule{
					Provider: "openai",
					Model:    "gpt-4o",
				},
			},
		},
	}

	handlers, err := registry.CreateHandlers(configs, routerProvider, providers, nil)
	if err != nil {
		t.Fatalf("CreateHandlers failed: %v", err)
	}

	// With the fix, fallback should trigger ModelMappingProvider creation
	if len(handlers) != 3 { // messages, count_tokens, models
		t.Errorf("expected 3 handlers, got %d", len(handlers))
	}
}

func TestRegistry_CreateHandlers_OpenAIFrontdoor(t *testing.T) {
	registry := NewRegistry()

	openaiProvider := &mockProvider{name: "openai", apiType: domain.APITypeOpenAI}
	routerProvider := &mockProvider{name: "router", apiType: domain.APITypeOpenAI}

	providers := map[string]ports.Provider{
		"openai": openaiProvider,
	}

	configs := []config.AppConfig{
		{
			Name:      "test-app",
			Frontdoor: "openai",
			Path:      "/openai",
		},
	}

	handlers, err := registry.CreateHandlers(configs, routerProvider, providers, nil)
	if err != nil {
		t.Fatalf("CreateHandlers failed: %v", err)
	}

	// OpenAI should register chat/completions and models
	if len(handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handlers))
	}

	expectedPaths := []string{"/openai/v1/chat/completions", "/openai/v1/models"}
	for _, expected := range expectedPaths {
		found := false
		for _, h := range handlers {
			if h.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected handler for path %s", expected)
		}
	}
}

func TestRegistry_CreateHandlers_UnknownFrontdoor(t *testing.T) {
	registry := NewRegistry()

	routerProvider := &mockProvider{name: "router", apiType: domain.APITypeOpenAI}

	configs := []config.AppConfig{
		{
			Name:      "test-app",
			Frontdoor: "unknown",
			Path:      "/unknown",
		},
	}

	_, err := registry.CreateHandlers(configs, routerProvider, nil, nil)
	if err == nil {
		t.Error("expected error for unknown frontdoor")
	}
	if !strings.Contains(err.Error(), "unknown frontdoor type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistry_CreateHandlers_UnknownProviderInRewrite(t *testing.T) {
	registry := NewRegistry()

	routerProvider := &mockProvider{name: "router", apiType: domain.APITypeOpenAI}

	providers := map[string]ports.Provider{}

	configs := []config.AppConfig{
		{
			Name:      "test-app",
			Frontdoor: "anthropic",
			Path:      "/test",
			ModelRouting: config.ModelRoutingConfig{
				Rewrites: []config.ModelRewriteRule{
					{
						ModelPrefix: "claude-",
						Provider:    "nonexistent",
						Model:       "some-model",
					},
				},
			},
		},
	}

	_, err := registry.CreateHandlers(configs, routerProvider, providers, nil)
	if err == nil {
		t.Error("expected error for unknown provider in rewrite")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistry_ResponsesHandlers(t *testing.T) {
	registry := NewRegistry()

	mockProv := &mockProvider{name: "test", apiType: domain.APITypeOpenAI}

	handlers := registry.CreateResponsesHandlers("/test", nil, mockProv)

	// Should have response create, get, cancel, and thread handlers
	// Note: threads/{thread_id}/messages has both POST and GET handlers
	expectedPaths := map[string][]string{
		"/test/v1/responses":                      {http.MethodPost},
		"/test/v1/responses/{response_id}":        {http.MethodGet},
		"/test/v1/responses/{response_id}/cancel": {http.MethodPost},
		"/test/v1/threads":                        {http.MethodPost},
		"/test/v1/threads/{thread_id}":            {http.MethodGet},
		"/test/v1/threads/{thread_id}/messages":   {http.MethodPost, http.MethodGet},
		"/test/v1/threads/{thread_id}/runs":       {http.MethodPost},
	}

	// Count expected handlers (sum of all methods)
	expectedCount := 0
	for _, methods := range expectedPaths {
		expectedCount += len(methods)
	}

	if len(handlers) != expectedCount {
		t.Errorf("expected %d handlers, got %d", expectedCount, len(handlers))
		for _, h := range handlers {
			t.Logf("  registered: %s %s", h.Method, h.Path)
		}
	}

	for expectedPath, expectedMethods := range expectedPaths {
		for _, expectedMethod := range expectedMethods {
			found := false
			for _, h := range handlers {
				if h.Path == expectedPath && h.Method == expectedMethod {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected handler for %s %s", expectedMethod, expectedPath)
			}
		}
	}
}

// Integration test that simulates the full request flow
func TestRegistry_IntegrationWithModelRewrite(t *testing.T) {
	registry := NewRegistry()

	openaiProvider := &mockProvider{
		name:    "openai",
		apiType: domain.APITypeOpenAI,
		response: &domain.CanonicalResponse{
			ID:    "msg_123",
			Model: "gpt-4o",
			Choices: []domain.Choice{
				{Message: domain.Message{Role: "assistant", Content: "Hello from OpenAI"}},
			},
		},
	}
	anthropicProvider := &mockProvider{name: "anthropic", apiType: domain.APITypeAnthropic}
	routerProvider := &mockProvider{name: "router", apiType: domain.APITypeOpenAI}

	providers := map[string]ports.Provider{
		"openai":    openaiProvider,
		"anthropic": anthropicProvider,
	}

	// Configure to rewrite claude requests to openai
	configs := []config.AppConfig{
		{
			Name:      "claude-proxy",
			Frontdoor: "anthropic",
			Path:      "/claude",
			ModelRouting: config.ModelRoutingConfig{
				Rewrites: []config.ModelRewriteRule{
					{
						ModelPrefix:          "claude-",
						Provider:             "openai",
						Model:                "gpt-4o",
						RewriteResponseModel: true,
					},
				},
				Fallback: &config.ModelRewriteRule{
					Provider: "openai",
					Model:    "gpt-4o-mini",
				},
			},
		},
	}

	handlers, err := registry.CreateHandlers(configs, routerProvider, providers, nil)
	if err != nil {
		t.Fatalf("CreateHandlers failed: %v", err)
	}

	// Find the messages handler
	var messagesHandler func(http.ResponseWriter, *http.Request)
	for _, h := range handlers {
		if h.Path == "/claude/v1/messages" && h.Method == http.MethodPost {
			messagesHandler = h.Handler
			break
		}
	}

	if messagesHandler == nil {
		t.Fatal("messages handler not found")
	}

	// Create a test request
	reqBody := `{
		"model": "claude-3-haiku-20240307",
		"messages": [{"role": "user", "content": "Hello"}],
		"max_tokens": 100
	}`

	req := httptest.NewRequest(http.MethodPost, "/claude/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	messagesHandler(w, req)

	// The request should have been routed to openai with rewritten model
	if openaiProvider.lastReq == nil {
		t.Fatal("request was not routed to openai provider")
	}

	if openaiProvider.lastReq.Model != "gpt-4o" {
		t.Errorf("expected model to be rewritten to 'gpt-4o', got %s", openaiProvider.lastReq.Model)
	}
}
