package conversation

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/memory"
)

func TestRecordInteraction(t *testing.T) {
	store := memory.New()
	ctx := context.Background()

	canonReq := &domain.CanonicalRequest{
		Model: "gpt-4",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	canonResp := &domain.CanonicalResponse{
		ID:    "resp-123",
		Model: "gpt-4",
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    "assistant",
					Content: "Hi there!",
				},
				FinishReason: "stop",
			},
		},
	}

	params := RecordInteractionParams{
		Store:          store,
		RawRequest:     []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`),
		CanonicalReq:   canonReq,
		RawResponse:    []byte(`{"id": "resp-123", "choices": [{"message": {"role": "assistant", "content": "Hi there!"}}]}`),
		CanonicalResp:  canonResp,
		ClientResponse: []byte(`{"id": "resp-123", "choices": [{"message": {"role": "assistant", "content": "Hi there!"}}]}`),
		RequestHeaders: http.Header{
			"User-Agent":    []string{"test-client/1.0"},
			"X-Request-Id":  []string{"req-abc123"},
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"Bearer secret"}, // Should not be captured
		},
		Frontdoor: domain.APITypeOpenAI,
		Provider:  "openai",
		AppName:   "test-app",
		Duration:  100 * time.Millisecond,
	}

	id := RecordInteraction(ctx, params)
	if id == "" {
		t.Fatal("expected non-empty interaction ID")
	}

	// Verify interaction was stored
	interaction, err := store.GetInteraction(ctx, id)
	if err != nil {
		t.Fatalf("failed to get interaction: %v", err)
	}

	// Verify fields
	if interaction.Frontdoor != domain.APITypeOpenAI {
		t.Errorf("expected frontdoor 'openai', got '%s'", interaction.Frontdoor)
	}
	if interaction.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", interaction.Provider)
	}
	if interaction.AppName != "test-app" {
		t.Errorf("expected app_name 'test-app', got '%s'", interaction.AppName)
	}
	if interaction.RequestedModel != "gpt-4" {
		t.Errorf("expected requested_model 'gpt-4', got '%s'", interaction.RequestedModel)
	}
	if interaction.ServedModel != "gpt-4" {
		t.Errorf("expected served_model 'gpt-4', got '%s'", interaction.ServedModel)
	}
	if interaction.Status != domain.InteractionStatusCompleted {
		t.Errorf("expected status 'completed', got '%s'", interaction.Status)
	}

	// Verify request headers were captured but sensitive ones were filtered
	if interaction.RequestHeaders["User-Agent"] != "test-client/1.0" {
		t.Errorf("expected User-Agent header to be captured")
	}
	if interaction.RequestHeaders["X-Request-Id"] != "req-abc123" {
		t.Errorf("expected X-Request-Id header to be captured")
	}
	if _, ok := interaction.RequestHeaders["Authorization"]; ok {
		t.Error("Authorization header should not be captured")
	}

	// Verify request data
	if interaction.Request == nil {
		t.Fatal("expected request data")
	}
	if len(interaction.Request.Raw) == 0 {
		t.Error("expected raw request to be captured")
	}
	if len(interaction.Request.CanonicalJSON) == 0 {
		t.Error("expected canonical request to be captured")
	}

	// Verify response data
	if interaction.Response == nil {
		t.Fatal("expected response data")
	}
	if len(interaction.Response.Raw) == 0 {
		t.Error("expected raw response to be captured")
	}
	if len(interaction.Response.CanonicalJSON) == 0 {
		t.Error("expected canonical response to be captured")
	}
	if len(interaction.Response.ClientResponse) == 0 {
		t.Error("expected client response to be captured")
	}
}

func TestRecordInteractionWithError(t *testing.T) {
	store := memory.New()
	ctx := context.Background()

	canonReq := &domain.CanonicalRequest{
		Model: "gpt-4",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	params := RecordInteractionParams{
		Store:          store,
		RawRequest:     []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`),
		CanonicalReq:   canonReq,
		RequestHeaders: http.Header{},
		Frontdoor:      domain.APITypeOpenAI,
		Provider:       "openai",
		AppName:        "test-app",
		Error:          &domain.APIError{Type: "rate_limit", Code: "rate_limit_exceeded", Message: "Too many requests"},
		Duration:       50 * time.Millisecond,
	}

	id := RecordInteraction(ctx, params)
	if id == "" {
		t.Fatal("expected non-empty interaction ID")
	}

	// Verify interaction was stored
	interaction, err := store.GetInteraction(ctx, id)
	if err != nil {
		t.Fatalf("failed to get interaction: %v", err)
	}

	// Verify error handling
	if interaction.Status != domain.InteractionStatusFailed {
		t.Errorf("expected status 'failed', got '%s'", interaction.Status)
	}
	if interaction.Error == nil {
		t.Fatal("expected error data")
	}
	if interaction.Error.Type != "rate_limit" {
		t.Errorf("expected error type 'rate_limit', got '%s'", interaction.Error.Type)
	}
	if interaction.Error.Code != "rate_limit_exceeded" {
		t.Errorf("expected error code 'rate_limit_exceeded', got '%s'", interaction.Error.Code)
	}
}

func TestRecordInteractionStreaming(t *testing.T) {
	store := memory.New()
	ctx := context.Background()

	canonReq := &domain.CanonicalRequest{
		Model:  "gpt-4",
		Stream: true,
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	canonResp := &domain.CanonicalResponse{
		Model: "gpt-4",
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    "assistant",
					Content: "Hi there! How can I help you today?",
				},
				FinishReason: "stop",
			},
		},
	}

	params := RecordInteractionParams{
		Store:          store,
		RawRequest:     []byte(`{"model": "gpt-4", "stream": true, "messages": [{"role": "user", "content": "Hello"}]}`),
		CanonicalReq:   canonReq,
		CanonicalResp:  canonResp,
		RequestHeaders: http.Header{},
		Frontdoor:      domain.APITypeAnthropic,
		Provider:       "anthropic",
		AppName:        "test-app",
		Streaming:      true,
		Duration:       200 * time.Millisecond,
		FinishReason:   "stop",
	}

	id := RecordInteraction(ctx, params)
	if id == "" {
		t.Fatal("expected non-empty interaction ID")
	}

	// Verify interaction was stored
	interaction, err := store.GetInteraction(ctx, id)
	if err != nil {
		t.Fatalf("failed to get interaction: %v", err)
	}

	// Verify streaming flag
	if !interaction.Streaming {
		t.Error("expected streaming to be true")
	}
	if interaction.Frontdoor != domain.APITypeAnthropic {
		t.Errorf("expected frontdoor 'anthropic', got '%s'", interaction.Frontdoor)
	}
}
