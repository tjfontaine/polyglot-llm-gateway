package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

type stubProvider struct {
	lastReq *domain.CanonicalRequest
}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	s.lastReq = req
	return &domain.CanonicalResponse{
		ID:    "resp",
		Model: req.Model,
		Choices: []domain.Choice{
			{Message: domain.Message{Role: "assistant", Content: "ok"}},
		},
	}, nil
}

func (s *stubProvider) Stream(context.Context, *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	return nil, nil
}

func TestHandleMessagesAcceptsContentBlocks(t *testing.T) {
	provider := &stubProvider{}
	handler := NewHandler(provider)

	body := `{
		"model": "claude-3-haiku-20240307",
		"max_tokens": 64,
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "Hello"},
					{"type": "text", "text": " world"}
				]
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/claude/v1/messages", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.HandleMessages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if provider.lastReq == nil {
		t.Fatalf("provider Complete was not called")
	}

	if got := provider.lastReq.Messages[0].Content; got != "Hello world" {
		t.Fatalf("expected merged content 'Hello world', got %q", got)
	}
}

func TestHandleMessagesRejectsUnsupportedBlocks(t *testing.T) {
	provider := &stubProvider{}
	handler := NewHandler(provider)

	body := `{
		"model": "claude-3",
		"messages": [
			{
				"role": "user",
				"content": [{"type": "image"}]
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/claude/v1/messages", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handler.HandleMessages(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
