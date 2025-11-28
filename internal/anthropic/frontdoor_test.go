package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

type stubProvider struct {
	lastReq    *domain.CanonicalRequest
	listCalled bool
}

func (s *stubProvider) Name() string            { return "stub" }
func (s *stubProvider) APIType() domain.APIType { return domain.APITypeAnthropic }

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

func (s *stubProvider) ListModels(context.Context) (*domain.ModelList, error) {
	s.listCalled = true
	return &domain.ModelList{Object: "list", Data: []domain.Model{{ID: "provider-model"}}}, nil
}

func TestHandleMessagesAcceptsContentBlocks(t *testing.T) {
	provider := &stubProvider{}
	handler := NewFrontdoorHandler(provider, nil, "test-app", nil)

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
	handler := NewFrontdoorHandler(provider, nil, "test-app", nil)

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

func TestHandleListModelsUsesConfiguredModels(t *testing.T) {
	provider := &stubProvider{}
	handler := NewFrontdoorHandler(provider, nil, "test-app", []config.ModelListItem{{
		ID:      "cfg-model",
		Object:  "model",
		OwnedBy: "owner",
	}})

	req := httptest.NewRequest(http.MethodGet, "/claude/v1/models", nil)
	rr := httptest.NewRecorder()

	handler.HandleListModels(rr, req)

	if provider.listCalled {
		t.Fatalf("expected provider ListModels not to be called when configured models exist")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var list domain.ModelList
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(list.Data) != 1 || list.Data[0].ID != "cfg-model" || list.Data[0].OwnedBy != "owner" {
		t.Fatalf("unexpected list response: %+v", list)
	}
}

func TestHandleListModelsFallsBackToProvider(t *testing.T) {
	provider := &stubProvider{}
	handler := NewFrontdoorHandler(provider, nil, "test-app", nil)

	req := httptest.NewRequest(http.MethodGet, "/claude/v1/models", nil)
	rr := httptest.NewRecorder()

	handler.HandleListModels(rr, req)

	if !provider.listCalled {
		t.Fatalf("expected provider ListModels to be called when no configured models")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var list domain.ModelList
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(list.Data) != 1 || list.Data[0].ID != "provider-model" {
		t.Fatalf("unexpected provider list response: %+v", list)
	}
}
