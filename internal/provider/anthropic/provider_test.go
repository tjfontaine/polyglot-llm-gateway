package anthropic

import (
	"context"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
	"github.com/tjfontaine/poly-llm-gateway/internal/testutil"
)

func TestProvider_Complete(t *testing.T) {
	// Skip if no API key and not in replay mode
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("VCR_MODE") == "record" {
		t.Skip("Skipping test: ANTHROPIC_API_KEY not set")
	}

	// Initialize VCR
	recorder, cleanup := testutil.NewVCRRecorder(t, "anthropic_complete")
	defer cleanup()

	// Create provider with VCR client
	client := testutil.VCRHTTPClient(recorder)

	// Use a dummy key for replay mode if not set
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}

	p := New(apiKey, option.WithHTTPClient(client))

	req := &domain.CanonicalRequest{
		Model: "claude-3-opus-20240229",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1024,
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice")
	}
	if resp.Choices[0].Message.Content == "" {
		t.Error("Expected content in response")
	}
}
