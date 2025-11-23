package openai

import (
	"context"
	"os"
	"testing"

	"github.com/openai/openai-go/option"
	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
	"github.com/tjfontaine/poly-llm-gateway/internal/testutil"
)

func TestProvider_Complete(t *testing.T) {
	// Skip if no API key and not in replay mode
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("VCR_MODE") == "record" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}

	// Initialize VCR
	recorder, cleanup := testutil.NewVCRRecorder(t, "openai_complete")
	defer cleanup()

	// Create provider with VCR client
	client := testutil.VCRHTTPClient(recorder)

	// Use a dummy key for replay mode if not set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}

	p := New(apiKey, option.WithHTTPClient(client))

	req := &domain.CanonicalRequest{
		Model: "gpt-3.5-turbo",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
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

func TestProvider_Stream(t *testing.T) {
	// Skip if no API key and not in replay mode
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("VCR_MODE") == "record" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}

	// Initialize VCR
	recorder, cleanup := testutil.NewVCRRecorder(t, "openai_stream")
	defer cleanup()

	// Create provider with VCR client
	client := testutil.VCRHTTPClient(recorder)

	// Use a dummy key for replay mode if not set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}

	p := New(apiKey, option.WithHTTPClient(client))

	req := &domain.CanonicalRequest{
		Model: "gpt-3.5-turbo",
		Messages: []domain.Message{
			{Role: "user", Content: "Count to 3"},
		},
	}

	stream, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var content string
	for event := range stream {
		if event.Error != nil {
			t.Fatalf("Stream event error: %v", event.Error)
		}
		content += event.ContentDelta
	}

	if content == "" {
		t.Error("Expected content in stream")
	}
	// Verify content contains "1, 2, 3" (approximate check)
	if len(content) < 3 {
		t.Errorf("Content too short: %s", content)
	}
}

func TestProvider_Error(t *testing.T) {
	// Skip if no API key and not in replay mode
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("VCR_MODE") == "record" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}

	// Initialize VCR
	recorder, cleanup := testutil.NewVCRRecorder(t, "openai_error")
	defer cleanup()

	// Create provider with VCR client
	client := testutil.VCRHTTPClient(recorder)

	// Use a dummy key for replay mode if not set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}

	p := New(apiKey, option.WithHTTPClient(client))

	req := &domain.CanonicalRequest{
		Model: "invalid-model",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Error("Expected error for invalid model")
	}
}
