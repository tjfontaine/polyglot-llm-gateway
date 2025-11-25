package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	anthropicapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
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

func TestToCanonicalRequest(t *testing.T) {
	apiReq := &anthropicapi.MessagesRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []anthropicapi.Message{
			{
				Role:    "user",
				Content: anthropicapi.ContentBlock{{Type: "text", Text: "Hello"}},
			},
		},
		System: anthropicapi.SystemMessages{
			{Type: "text", Text: "You are a helpful assistant."},
		},
		MaxTokens: 100,
	}

	canonReq, err := ToCanonicalRequest(apiReq)
	if err != nil {
		t.Fatalf("ToCanonicalRequest returned error: %v", err)
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
