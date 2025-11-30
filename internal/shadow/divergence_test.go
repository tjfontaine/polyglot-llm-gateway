package shadow

import (
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

func TestDetectDivergences_NilInputs(t *testing.T) {
	// Nil primary
	divs := DetectDivergences(nil, &domain.ShadowResult{})
	if divs != nil {
		t.Errorf("expected nil for nil primary, got %v", divs)
	}

	// Nil shadow
	divs = DetectDivergences(&domain.CanonicalResponse{}, nil)
	if divs != nil {
		t.Errorf("expected nil for nil shadow, got %v", divs)
	}

	// Nil shadow response
	divs = DetectDivergences(&domain.CanonicalResponse{}, &domain.ShadowResult{})
	if divs != nil {
		t.Errorf("expected nil for nil shadow response, got %v", divs)
	}
}

func TestDetectDivergences_ChoiceCountDifference(t *testing.T) {
	primary := &domain.CanonicalResponse{
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "Hello"}},
			{Message: domain.Message{Role: "assistant", Content: "World"}},
		},
	}
	shadow := &domain.ShadowResult{
		Response: &domain.ShadowResponse{
			Canonical: []byte(`{"choices":[{"message":{"role":"assistant","content":"Hi"}}]}`),
		},
	}

	divs := DetectDivergences(primary, shadow)
	if len(divs) == 0 {
		t.Error("expected divergences for different choice counts")
	}

	found := false
	for _, d := range divs {
		if d.Type == domain.DivergenceArrayLength && d.Path == "choices" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected DivergenceArrayLength for choices")
	}
}

func TestDetectDivergences_FinishReasonDifference(t *testing.T) {
	primary := &domain.CanonicalResponse{
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "Hello"}, FinishReason: "stop"},
		},
	}
	shadow := &domain.ShadowResult{
		Response: &domain.ShadowResponse{
			Canonical: []byte(`{"choices":[{"message":{"role":"assistant","content":"Hi"},"finish_reason":"length"}]}`),
		},
	}

	divs := DetectDivergences(primary, shadow)

	found := false
	for _, d := range divs {
		if d.Type == domain.DivergenceTypeMismatch && d.Path == "choices[0].finish_reason" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected DivergenceTypeMismatch for finish_reason")
	}
}

func TestDetectDivergences_RoleDifference(t *testing.T) {
	primary := &domain.CanonicalResponse{
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "Hello"}},
		},
	}
	shadow := &domain.ShadowResult{
		Response: &domain.ShadowResponse{
			Canonical: []byte(`{"choices":[{"message":{"role":"user","content":"Hello"}}]}`),
		},
	}

	divs := DetectDivergences(primary, shadow)

	found := false
	for _, d := range divs {
		if d.Type == domain.DivergenceTypeMismatch && d.Path == "choices[0].message.role" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected DivergenceTypeMismatch for message role")
	}
}

func TestDetectDivergences_NoStructuralDivergence(t *testing.T) {
	primary := &domain.CanonicalResponse{
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "Hello"}, FinishReason: "stop"},
		},
	}
	shadow := &domain.ShadowResult{
		Response: &domain.ShadowResponse{
			// Same structure, different content (content differences are expected and not flagged)
			Canonical: []byte(`{"choices":[{"message":{"role":"assistant","content":"Different content"},"finish_reason":"stop"}]}`),
		},
	}

	divs := DetectDivergences(primary, shadow)
	if len(divs) != 0 {
		t.Errorf("expected no divergences for same structure, got %v", divs)
	}
}

func TestHasMessageContent(t *testing.T) {
	tests := []struct {
		name     string
		msg      domain.Message
		expected bool
	}{
		{
			name:     "empty message",
			msg:      domain.Message{},
			expected: false,
		},
		{
			name:     "text content",
			msg:      domain.Message{Content: "hello"},
			expected: true,
		},
		{
			name:     "empty text",
			msg:      domain.Message{Content: ""},
			expected: false,
		},
		{
			name: "rich content with text",
			msg: domain.Message{
				RichContent: &domain.MessageContent{Text: "hello"},
			},
			expected: true,
		},
		{
			name: "rich content with parts",
			msg: domain.Message{
				RichContent: &domain.MessageContent{
					Parts: []domain.ContentPart{{Type: domain.ContentTypeText, Text: "hello"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasMessageContent(&tt.msg)
			if result != tt.expected {
				t.Errorf("hasMessageContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}
