package tokens

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

func TestEstimator_CountTokens(t *testing.T) {
	e := NewEstimator()

	tests := []struct {
		name           string
		req            *domain.TokenCountRequest
		minTokens      int
		maxTokens      int
	}{
		{
			name: "simple message",
			req: &domain.TokenCountRequest{
				Model: "test-model",
				Messages: []domain.Message{
					{Role: "user", Content: "Hello, how are you?"},
				},
			},
			minTokens: 5,
			maxTokens: 15,
		},
		{
			name: "with system message",
			req: &domain.TokenCountRequest{
				Model:  "test-model",
				System: "You are a helpful assistant.",
				Messages: []domain.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			minTokens: 8,
			maxTokens: 20,
		},
		{
			name: "multiple messages",
			req: &domain.TokenCountRequest{
				Model: "test-model",
				Messages: []domain.Message{
					{Role: "user", Content: "What is 2+2?"},
					{Role: "assistant", Content: "2+2 equals 4."},
					{Role: "user", Content: "Thanks!"},
				},
			},
			minTokens: 10,
			maxTokens: 30,
		},
		{
			name: "with tools",
			req: &domain.TokenCountRequest{
				Model: "test-model",
				Messages: []domain.Message{
					{Role: "user", Content: "Calculate something"},
				},
				Tools: []domain.TokenCountTool{
					{
						Name:        "calculator",
						Description: "A simple calculator",
					},
				},
			},
			minTokens: 10,
			maxTokens: 40,
		},
		{
			name: "empty request",
			req: &domain.TokenCountRequest{
				Model:    "test-model",
				Messages: []domain.Message{},
			},
			minTokens: 0,
			maxTokens: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := e.CountTokens(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("CountTokens() error = %v", err)
			}

			if !resp.Estimated {
				t.Error("expected Estimated to be true for estimator")
			}

			if resp.InputTokens < tt.minTokens || resp.InputTokens > tt.maxTokens {
				t.Errorf("CountTokens() = %d, want between %d and %d", 
					resp.InputTokens, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestEstimator_SupportsModel(t *testing.T) {
	e := NewEstimator()

	// Estimator should support all models as a fallback
	models := []string{"gpt-4", "claude-3", "unknown-model", ""}
	for _, model := range models {
		if !e.SupportsModel(model) {
			t.Errorf("SupportsModel(%q) = false, want true", model)
		}
	}
}

func TestOpenAICounter_CountTokens(t *testing.T) {
	c := NewOpenAICounter()

	tests := []struct {
		name           string
		req            *domain.TokenCountRequest
		minTokens      int
		maxTokens      int
	}{
		{
			name: "simple message",
			req: &domain.TokenCountRequest{
				Model: "gpt-4o",
				Messages: []domain.Message{
					{Role: "user", Content: "Hello, how are you today?"},
				},
			},
			minTokens: 8,
			maxTokens: 20,
		},
		{
			name: "code snippet",
			req: &domain.TokenCountRequest{
				Model: "gpt-4o",
				Messages: []domain.Message{
					{Role: "user", Content: "def hello(): print('Hello, World!')"},
				},
			},
			minTokens: 10,
			maxTokens: 30,
		},
		{
			name: "common words",
			req: &domain.TokenCountRequest{
				Model: "gpt-4o",
				Messages: []domain.Message{
					{Role: "user", Content: "The quick brown fox jumps over the lazy dog."},
				},
			},
			minTokens: 12,
			maxTokens: 25,
		},
		{
			name: "numbers",
			req: &domain.TokenCountRequest{
				Model: "gpt-4o",
				Messages: []domain.Message{
					{Role: "user", Content: "123456789 and 987654321"},
				},
			},
			minTokens: 8,
			maxTokens: 20,
		},
		{
			name: "camelCase words",
			req: &domain.TokenCountRequest{
				Model: "gpt-4o",
				Messages: []domain.Message{
					{Role: "user", Content: "getCustomerById calculateTotalPrice"},
				},
			},
			minTokens: 8,
			maxTokens: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.CountTokens(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("CountTokens() error = %v", err)
			}

			if !resp.Estimated {
				t.Error("expected Estimated to be true for OpenAI counter")
			}

			if resp.InputTokens < tt.minTokens || resp.InputTokens > tt.maxTokens {
				t.Errorf("CountTokens() = %d, want between %d and %d", 
					resp.InputTokens, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestOpenAICounter_SupportsModel(t *testing.T) {
	c := NewOpenAICounter()

	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-4o", true},
		{"gpt-4-turbo", true},
		{"gpt-3.5-turbo", true},
		{"o1-preview", true},
		{"o3-mini", true},
		{"text-embedding-ada-002", true},
		{"claude-3-sonnet", false},
		{"unknown-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := c.SupportsModel(tt.model); got != tt.expected {
				t.Errorf("SupportsModel(%q) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestRegistry_CountTokens(t *testing.T) {
	// Create registry with OpenAI counter and fallback estimator
	registry := NewRegistry()
	registry.Register(NewOpenAICounter())

	tests := []struct {
		name        string
		model       string
		wantCounter string // "openai" or "estimator"
	}{
		{"gpt model uses OpenAI counter", "gpt-4o", "openai"},
		{"unknown model uses fallback", "unknown-model", "estimator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &domain.TokenCountRequest{
				Model: tt.model,
				Messages: []domain.Message{
					{Role: "user", Content: "Hello"},
				},
			}

			resp, err := registry.CountTokens(context.Background(), req)
			if err != nil {
				t.Fatalf("CountTokens() error = %v", err)
			}

			// All our current implementations return estimates
			if !resp.Estimated {
				t.Logf("note: model %s returned non-estimated count", tt.model)
			}

			if resp.InputTokens <= 0 {
				t.Error("expected positive token count")
			}
		})
	}
}

func TestRegistry_GetCounter(t *testing.T) {
	registry := NewRegistry()
	openaiCounter := NewOpenAICounter()
	registry.Register(openaiCounter)

	// GPT model should get OpenAI counter
	counter := registry.GetCounter("gpt-4o")
	if _, ok := counter.(*OpenAICounter); !ok {
		t.Error("expected OpenAI counter for gpt-4o")
	}

	// Unknown model should get fallback (Estimator)
	counter = registry.GetCounter("unknown-model")
	if _, ok := counter.(*Estimator); !ok {
		t.Error("expected Estimator fallback for unknown model")
	}
}

func TestModelMatcher(t *testing.T) {
	matcher := NewModelMatcher(
		[]string{"gpt-", "claude-"},
		[]string{"davinci", "curie"},
	)

	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-4", true},
		{"gpt-3.5-turbo", true},
		{"claude-3-opus", true},
		{"davinci", true},
		{"curie", true},
		{"text-davinci-003", false}, // not exact match
		{"llama-2", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := matcher.Matches(tt.model); got != tt.expected {
				t.Errorf("Matches(%q) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestOpenAICounter_estimateTokens(t *testing.T) {
	c := NewOpenAICounter()

	tests := []struct {
		text      string
		minTokens int
		maxTokens int
	}{
		{"", 0, 0},
		{"hello", 1, 2},
		{"Hello, World!", 3, 6},
		{"The quick brown fox", 4, 8},
		{"supercalifragilisticexpialidocious", 3, 10},
		{"12345", 1, 3},
		{"getUserById", 2, 5}, // camelCase
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := c.estimateTokens(tt.text)
			if got < tt.minTokens || got > tt.maxTokens {
				t.Errorf("estimateTokens(%q) = %d, want between %d and %d",
					tt.text, got, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

// Benchmark for token estimation
func BenchmarkOpenAICounter_CountTokens(b *testing.B) {
	c := NewOpenAICounter()
	req := &domain.TokenCountRequest{
		Model: "gpt-4o",
		Messages: []domain.Message{
			{Role: "system", Content: "You are a helpful assistant that provides detailed answers."},
			{Role: "user", Content: "Can you explain quantum computing in simple terms? I'd like to understand the basics of qubits, superposition, and entanglement."},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.CountTokens(context.Background(), req)
	}
}

func BenchmarkEstimator_CountTokens(b *testing.B) {
	e := NewEstimator()
	req := &domain.TokenCountRequest{
		Model: "test-model",
		Messages: []domain.Message{
			{Role: "system", Content: "You are a helpful assistant that provides detailed answers."},
			{Role: "user", Content: "Can you explain quantum computing in simple terms? I'd like to understand the basics of qubits, superposition, and entanglement."},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.CountTokens(context.Background(), req)
	}
}
