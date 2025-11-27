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
			minTokens: 10,
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
			minTokens: 15,
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
			minTokens: 15,
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
			minTokens: 10,
			maxTokens: 25,
		},
		{
			name: "camelCase words",
			req: &domain.TokenCountRequest{
				Model: "gpt-4o",
				Messages: []domain.Message{
					{Role: "user", Content: "getCustomerById calculateTotalPrice"},
				},
			},
			minTokens: 10,
			maxTokens: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.CountTokens(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("CountTokens() error = %v", err)
			}

			// tiktoken provides accurate counts, not estimates
			if resp.Estimated {
				t.Error("expected Estimated to be false for tiktoken-based counter")
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
		// Current models
		{"gpt-4o", true},
		{"gpt-4-turbo", true},
		{"gpt-3.5-turbo", true},
		{"gpt-4.1", true},
		{"gpt-4.1-mini", true},
		// GPT-5 family
		{"gpt-5", true},
		{"gpt-5-mini", true},
		{"gpt-5-nano", true},
		{"gpt-5-turbo", true},
		// GPT-5.1+ (newer point releases)
		{"gpt-5.1", true},
		{"gpt-5.1-mini", true},
		{"gpt-5.1-turbo", true},
		{"gpt-5.2-preview", true},
		// Future GPT models
		{"gpt-6", true},
		{"gpt-6-preview", true},
		{"gpt-6-mini", true},
		// O-series reasoning models
		{"o1", true},
		{"o1-preview", true},
		{"o1-mini", true},
		{"o3", true},
		{"o3-mini", true},
		{"o4-mini", true},
		{"o4-reasoning", true},
		{"o5-preview", true},
		// Embedding models
		{"text-embedding-ada-002", true},
		{"text-embedding-3-large", true},
		// Legacy models
		{"text-davinci-003", true},
		{"davinci", true},
		{"curie", true},
		{"babbage", true},
		{"ada", true},
		// Non-OpenAI models (should not match)
		{"claude-3-sonnet", false},
		{"unknown-model", false},
		{"llama-3", false},
		{"gemini-pro", false},
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

func TestOpenAICounter_CountText(t *testing.T) {
	c := NewOpenAICounter()

	tests := []struct {
		text      string
		minTokens int
		maxTokens int
	}{
		{"", 0, 0},
		{"hello", 1, 1},
		{"Hello, World!", 3, 5},
		{"The quick brown fox", 4, 5},
		{"supercalifragilisticexpialidocious", 7, 12},
		{"12345", 2, 3},
		{"getUserById", 4, 6}, // camelCase splits into subwords
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, err := c.CountText("gpt-4o", tt.text)
			if err != nil {
				t.Fatalf("CountText() error = %v", err)
			}
			if got < tt.minTokens || got > tt.maxTokens {
				t.Errorf("CountText(%q) = %d, want between %d and %d",
					tt.text, got, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestOpenAICounter_DifferentModels(t *testing.T) {
	c := NewOpenAICounter()

	text := "Hello, how are you?"

	tests := []struct {
		model    string
		encoding string // for reference
	}{
		{"gpt-4o", "o200k_base"},
		{"gpt-3.5-turbo", "cl100k_base"},
		{"gpt-4-turbo", "cl100k_base"},
		{"gpt-5", "o200k_base"},
		{"o1-preview", "o200k_base"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			count, err := c.CountText(tt.model, text)
			if err != nil {
				t.Fatalf("CountText() error = %v", err)
			}
			if count <= 0 {
				t.Error("expected positive token count")
			}
			// Different encodings produce slightly different token counts
			// o200k_base and cl100k_base both produce ~6 tokens for this text
			if count < 5 || count > 7 {
				t.Errorf("CountText(%q, %q) = %d, want between 5 and 7", tt.model, text, count)
			}
		})
	}
}

func TestOpenAICounter_FutureModels(t *testing.T) {
	c := NewOpenAICounter()

	// Test that future models (gpt-5, gpt-5.1, gpt-6, etc.) work with correct encoding
	futureModels := []string{
		// GPT-5 family (explicitly supported in tiktoken-go/tokenizer)
		"gpt-5",
		"gpt-5-mini",
		"gpt-5-nano",
		"gpt-5-turbo",
		"gpt-5-turbo-preview",
		// GPT-5.1+ (should use o200k_base fallback)
		"gpt-5.1",
		"gpt-5.1-mini",
		"gpt-5.1-turbo",
		"gpt-5.2-preview",
		// GPT-6 and beyond (future-proofing)
		"gpt-6",
		"gpt-6-mini",
		"gpt-6-turbo",
		// O-series reasoning models
		"o4-preview",
		"o4-mini",
		"o5-reasoning",
	}

	text := "The quick brown fox jumps over the lazy dog."

	for _, model := range futureModels {
		t.Run(model, func(t *testing.T) {
			// Should support the model
			if !c.SupportsModel(model) {
				t.Errorf("SupportsModel(%q) = false, want true", model)
			}

			// Should be able to count tokens using appropriate encoding
			count, err := c.CountText(model, text)
			if err != nil {
				t.Fatalf("CountText() error = %v", err)
			}

			// Should get a reasonable count (o200k_base encoding for newer models)
			// "The quick brown fox jumps over the lazy dog." = ~10 tokens
			if count < 8 || count > 12 {
				t.Errorf("CountText(%q, %q) = %d, expected ~10 tokens", model, text, count)
			}
		})
	}
}

func TestOpenAICounter_CountTokensWithFutureModel(t *testing.T) {
	c := NewOpenAICounter()

	req := &domain.TokenCountRequest{
		Model: "gpt-5-turbo",
		Messages: []domain.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
	}

	resp, err := c.CountTokens(context.Background(), req)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}

	// Should return accurate counts (not estimated)
	if resp.Estimated {
		t.Error("expected Estimated to be false for tiktoken-based counter")
	}

	// Should have reasonable token count
	if resp.InputTokens < 10 || resp.InputTokens > 25 {
		t.Errorf("CountTokens() = %d, expected between 10 and 25", resp.InputTokens)
	}

	if resp.Model != "gpt-5-turbo" {
		t.Errorf("Model = %q, want %q", resp.Model, "gpt-5-turbo")
	}
}

func TestOpenAICounter_GPT51Models(t *testing.T) {
	c := NewOpenAICounter()

	// Specifically test GPT-5.1 and its variants
	tests := []struct {
		model string
	}{
		{"gpt-5.1"},
		{"gpt-5.1-mini"},
		{"gpt-5.1-turbo"},
		{"gpt-5.1-turbo-preview"},
		{"gpt-5.1-0125"},  // Date-stamped variant
		{"gpt-5.2"},
		{"gpt-5.2-mini"},
		{"gpt-5.3-preview"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			// Should support the model
			if !c.SupportsModel(tt.model) {
				t.Errorf("SupportsModel(%q) = false, want true", tt.model)
			}

			// Count tokens for a known text
			req := &domain.TokenCountRequest{
				Model: tt.model,
				Messages: []domain.Message{
					{Role: "user", Content: "What is the meaning of life?"},
				},
			}

			resp, err := c.CountTokens(context.Background(), req)
			if err != nil {
				t.Fatalf("CountTokens() error = %v", err)
			}

			// Should not be estimated (tiktoken-go provides accurate counts)
			if resp.Estimated {
				t.Error("expected Estimated to be false")
			}

			// Should have reasonable token count (o200k_base encoding)
			if resp.InputTokens < 10 || resp.InputTokens > 20 {
				t.Errorf("CountTokens() = %d, expected between 10 and 20", resp.InputTokens)
			}

			// CountText should also work
			count, err := c.CountText(tt.model, "Hello world")
			if err != nil {
				t.Fatalf("CountText() error = %v", err)
			}
			if count < 1 || count > 3 {
				t.Errorf("CountText(%q, %q) = %d, expected 1-3", tt.model, "Hello world", count)
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
