package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	anthropicapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	anthropiccodec "github.com/tjfontaine/polyglot-llm-gateway/internal/codec/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/testutil"
)

func TestListModels(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header to be 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("expected anthropic-version header to be set")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "data": [
    {
      "id": "claude-3-5-sonnet-20241022",
      "type": "model",
      "display_name": "Claude 3.5 Sonnet",
      "created_at": "2024-10-22T00:00:00Z"
    },
    {
      "id": "claude-3-haiku-20240307",
      "type": "model",
      "display_name": "Claude 3 Haiku",
      "created_at": "2024-03-07T00:00:00Z"
    }
  ],
  "first_id": "claude-3-5-sonnet-20241022",
  "last_id": "claude-3-haiku-20240307",
  "has_more": false
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	list, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if list.Object != "list" {
		t.Fatalf("expected object 'list', got %q", list.Object)
	}

	if len(list.Data) != 2 {
		t.Fatalf("expected 2 models, got %d", len(list.Data))
	}

	if list.Data[0].ID != "claude-3-5-sonnet-20241022" || list.Data[0].Object != "model" {
		t.Fatalf("unexpected first model: %+v", list.Data[0])
	}
}

func TestComplete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header to be 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("User-Agent") != "test-client/1.0" {
			t.Errorf("expected User-Agent header to be 'test-client/1.0', got %q", r.Header.Get("User-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-3-haiku-20240307",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	req := &domain.CanonicalRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
		UserAgent: "test-client/1.0",
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if resp.ID != "msg_123" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
	if resp.Model != "claude-3-haiku-20240307" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("unexpected content: %+v", resp.Choices)
	}
}

func TestUserAgentPassthrough(t *testing.T) {
	var receivedUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-3-haiku-20240307",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	req := &domain.CanonicalRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
		UserAgent: "my-custom-agent/2.0",
	}

	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if receivedUA != "my-custom-agent/2.0" {
		t.Errorf("User-Agent not passed through correctly. Expected 'my-custom-agent/2.0', got %q", receivedUA)
	}
}

func TestDefaultUserAgent(t *testing.T) {
	var receivedUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-3-haiku-20240307",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	req := &domain.CanonicalRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
		// No UserAgent set - should use default
	}

	_, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if receivedUA != "polyglot-llm-gateway/1.0" {
		t.Errorf("Default User-Agent not used. Expected 'polyglot-llm-gateway/1.0', got %q", receivedUA)
	}
}

func TestCodecAPIRequestToCanonical(t *testing.T) {
	codec := anthropiccodec.New()

	reqJSON := []byte(`{
		"model": "claude-3-haiku-20240307",
		"messages": [{"role": "user", "content": "Hello"}],
		"system": [{"type": "text", "text": "You are a helpful assistant."}],
		"max_tokens": 100
	}`)

	canonReq, err := codec.DecodeRequest(reqJSON)
	if err != nil {
		t.Fatalf("DecodeRequest returned error: %v", err)
	}

	if canonReq.Model != "claude-3-haiku-20240307" {
		t.Errorf("unexpected model: %s", canonReq.Model)
	}

	// Should have system message first, then user message
	if len(canonReq.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(canonReq.Messages))
	}

	if canonReq.Messages[0].Role != "system" {
		t.Errorf("expected first message to be system, got %s", canonReq.Messages[0].Role)
	}
	if canonReq.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("unexpected system content: %s", canonReq.Messages[0].Content)
	}

	if canonReq.Messages[1].Role != "user" {
		t.Errorf("expected second message to be user, got %s", canonReq.Messages[1].Role)
	}
	if canonReq.Messages[1].Content != "Hello" {
		t.Errorf("unexpected user content: %s", canonReq.Messages[1].Content)
	}
}

func TestCodecEncodeResponse(t *testing.T) {
	codec := anthropiccodec.New()

	canonResp := &domain.CanonicalResponse{
		ID:    "msg_123",
		Model: "claude-3-haiku-20240307",
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "end_turn",
			},
		},
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	respJSON, err := codec.EncodeResponse(canonResp)
	if err != nil {
		t.Fatalf("EncodeResponse returned error: %v", err)
	}

	// Verify it can be decoded back
	decoded, err := codec.DecodeResponse(respJSON)
	if err != nil {
		t.Fatalf("DecodeResponse returned error: %v", err)
	}

	if decoded.ID != canonResp.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, canonResp.ID)
	}
	if decoded.Model != canonResp.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, canonResp.Model)
	}
}

// VCR-based integration tests

func TestProvider_Complete_VCR(t *testing.T) {
	// Skip if no API key and in record mode
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

	p := New(apiKey, WithHTTPClient(client))

	req := &domain.CanonicalRequest{
		Model: "claude-3-opus-20240229",
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1024,
		UserAgent: "test-agent/1.0",
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
	if resp.Model == "" {
		t.Error("Expected model in response")
	}
}

func TestProvider_CountTokens_VCR(t *testing.T) {
	// Skip if no API key and in record mode
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("VCR_MODE") == "record" {
		t.Skip("Skipping test: ANTHROPIC_API_KEY not set")
	}

	// Initialize VCR
	recorder, cleanup := testutil.NewVCRRecorder(t, "anthropic_count_tokens")
	defer cleanup()

	// Create provider with VCR client
	client := testutil.VCRHTTPClient(recorder)

	// Use a dummy key for replay mode if not set
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}

	p := New(apiKey, WithHTTPClient(client))

	// Create count_tokens request body
	reqBody := anthropicapi.CountTokensRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []anthropicapi.Message{
			{
				Role:    "user",
				Content: []anthropicapi.ContentPart{{Type: "text", Text: "Hello, how are you?"}},
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	respBody, err := p.CountTokens(context.Background(), body)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}

	var resp anthropicapi.CountTokensResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.InputTokens == 0 {
		t.Error("Expected non-zero input_tokens")
	}
}

func TestCountTokens_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages/count_tokens" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header to be 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-beta") != "token-counting-2024-11-01" {
			t.Errorf("expected anthropic-beta header to be 'token-counting-2024-11-01', got %q", r.Header.Get("anthropic-beta"))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"input_tokens": 25}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	reqBody := anthropicapi.CountTokensRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []anthropicapi.Message{
			{
				Role:    "user",
				Content: []anthropicapi.ContentPart{{Type: "text", Text: "Test message"}},
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	respBody, err := p.CountTokens(context.Background(), body)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}

	var resp anthropicapi.CountTokensResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.InputTokens != 25 {
		t.Errorf("expected 25 input_tokens, got %d", resp.InputTokens)
	}
}

// Tests for 529 retry/backoff logic

func TestComplete_529Retry_Success(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts return 529 overloaded
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable) // 529 maps to 503
			fmt.Fprintln(w, `{"type": "error", "error": {"type": "overloaded_error", "message": "Overloaded"}}`)
			return
		}
		// Third attempt succeeds
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-3-haiku-20240307",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL), WithMaxRetries(2))

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []domain.Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 100,
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if resp.ID != "msg_123" {
		t.Errorf("Unexpected response ID: %s", resp.ID)
	}
}

func TestComplete_529Retry_ExhaustedRetries(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Always return 529 overloaded
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, `{"type": "error", "error": {"type": "overloaded_error", "message": "Overloaded"}}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL), WithMaxRetries(2))

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []domain.Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 100,
	}

	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error after exhausting retries")
	}

	// Should have tried 3 times (initial + 2 retries)
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Error should mention overloaded
	if !contains(err.Error(), "overloaded") {
		t.Errorf("Expected error to mention 'overloaded', got: %v", err)
	}
}

func TestComplete_NonRetryableError(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Return authentication error (not retryable)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, `{"type": "error", "error": {"type": "authentication_error", "message": "Invalid API key"}}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL), WithMaxRetries(2))

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []domain.Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 100,
	}

	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for authentication failure")
	}

	// Should NOT retry for non-retryable errors
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", attempts)
	}
}

func TestComplete_RateLimitHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set rate limit headers
		w.Header().Set("anthropic-ratelimit-requests-limit", "100")
		w.Header().Set("anthropic-ratelimit-requests-remaining", "95")
		w.Header().Set("anthropic-ratelimit-requests-reset", "2024-01-01T00:00:00Z")
		w.Header().Set("anthropic-ratelimit-tokens-limit", "100000")
		w.Header().Set("anthropic-ratelimit-tokens-remaining", "99000")
		w.Header().Set("anthropic-ratelimit-tokens-reset", "2024-01-01T00:01:00Z")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-3-haiku-20240307",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []domain.Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 100,
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	// Verify rate limit info is captured
	if resp.RateLimits == nil {
		t.Fatal("Expected RateLimits to be set")
	}

	if resp.RateLimits.RequestsLimit != 100 {
		t.Errorf("Expected RequestsLimit=100, got %d", resp.RateLimits.RequestsLimit)
	}
	if resp.RateLimits.RequestsRemaining != 95 {
		t.Errorf("Expected RequestsRemaining=95, got %d", resp.RateLimits.RequestsRemaining)
	}
	if resp.RateLimits.TokensLimit != 100000 {
		t.Errorf("Expected TokensLimit=100000, got %d", resp.RateLimits.TokensLimit)
	}
	if resp.RateLimits.TokensRemaining != 99000 {
		t.Errorf("Expected TokensRemaining=99000, got %d", resp.RateLimits.TokensRemaining)
	}
}

func TestComplete_ToolCalls(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [
    {"type": "text", "text": "I'll check the weather for you."},
    {"type": "tool_use", "id": "toolu_123", "name": "get_weather", "input": {"location": "San Francisco"}}
  ],
  "model": "claude-3-haiku-20240307",
  "stop_reason": "tool_use",
  "usage": {"input_tokens": 50, "output_tokens": 30}
}`)
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []domain.Message{{Role: "user", Content: "What's the weather in SF?"}},
		MaxTokens: 100,
		Tools: []domain.ToolDefinition{{
			Type: "function",
			Function: domain.FunctionDef{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]string{"type": "string"},
					},
				},
			},
		}},
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Expected at least one choice")
	}

	choice := resp.Choices[0]

	// Should have tool_calls finish reason
	if choice.FinishReason != "tool_calls" {
		t.Errorf("Expected finish_reason='tool_calls', got %s", choice.FinishReason)
	}

	// Should have text content
	if !contains(choice.Message.Content, "weather") {
		t.Errorf("Expected content to contain 'weather', got: %s", choice.Message.Content)
	}

	// Should have tool call
	if len(choice.Message.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(choice.Message.ToolCalls))
	}

	tc := choice.Message.ToolCalls[0]
	if tc.ID != "toolu_123" {
		t.Errorf("Expected tool call ID='toolu_123', got %s", tc.ID)
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Expected function name='get_weather', got %s", tc.Function.Name)
	}
}

func TestStream_ToolCalls(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, _ := w.(http.Flusher)

		// Simulate streaming events for a tool call
		events := []string{
			`event: message_start
data: {"type": "message_start", "message": {"id": "msg_123", "type": "message", "role": "assistant", "content": [], "model": "claude-3-haiku-20240307", "usage": {"input_tokens": 50}}}`,

			`event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "tool_use", "id": "toolu_123", "name": "get_weather"}}`,

			`event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "input_json_delta", "partial_json": "{\"location\":"}}`,

			`event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "input_json_delta", "partial_json": "\"SF\"}"}}`,

			`event: content_block_stop
data: {"type": "content_block_stop", "index": 0}`,

			`event: message_delta
data: {"type": "message_delta", "delta": {"stop_reason": "tool_use"}, "usage": {"output_tokens": 30}}`,

			`event: message_stop
data: {"type": "message_stop"}`,
		}

		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
			flusher.Flush()
		}
	}))
	defer ts.Close()

	p := New("test-key", WithBaseURL(ts.URL))

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []domain.Message{{Role: "user", Content: "What's the weather?"}},
		MaxTokens: 100,
		Stream:    true,
	}

	events, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	var toolCallEvents []domain.CanonicalEvent
	var finishReason string

	for event := range events {
		if event.Error != nil {
			t.Fatalf("Stream event error: %v", event.Error)
		}
		if event.ToolCall != nil {
			toolCallEvents = append(toolCallEvents, event)
		}
		if event.FinishReason != "" {
			finishReason = event.FinishReason
		}
	}

	// Should have tool call events
	if len(toolCallEvents) < 2 {
		t.Errorf("Expected at least 2 tool call events, got %d", len(toolCallEvents))
	}

	// Check first event is content_block_start
	if len(toolCallEvents) > 0 && toolCallEvents[0].Type != domain.EventTypeContentBlockStart {
		t.Errorf("Expected first event to be content_block_start, got %v", toolCallEvents[0].Type)
	}

	// Check tool call ID and name
	if len(toolCallEvents) > 0 {
		tc := toolCallEvents[0].ToolCall
		if tc.ID != "toolu_123" {
			t.Errorf("Expected tool call ID='toolu_123', got %s", tc.ID)
		}
		if tc.Function.Name != "get_weather" {
			t.Errorf("Expected function name='get_weather', got %s", tc.Function.Name)
		}
	}

	// Check finish reason is mapped correctly
	if finishReason != "tool_calls" {
		t.Errorf("Expected finish_reason='tool_calls', got %s", finishReason)
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"stop_sequence", "stop"},
		{"tool_use", "tool_calls"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapStopReason(tt.input)
			if result != tt.expected {
				t.Errorf("mapStopReason(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
