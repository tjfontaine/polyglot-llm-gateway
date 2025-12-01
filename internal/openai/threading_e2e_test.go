package openai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/memory"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/sqldb"
)

func TestResponsesThreadingPersistsAcrossProviders(t *testing.T) {
	var requests []map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		defer r.Body.Close()

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to parse request payload: %v", err)
		}
		requests = append(requests, payload)

		resp := ResponsesResponse{
			ID:        fmt.Sprintf("resp_%d", len(requests)),
			Object:    "response",
			CreatedAt: time.Now().Unix(),
			Status:    "completed",
			Model:     "gpt-4o-mini",
			Output: []ResponsesOutputItem{{
				Type: "message",
				Role: "assistant",
				Content: []ResponsesContentPart{{
					Type: "output_text",
					Text: "hello",
				}},
			}},
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	dbPath := t.TempDir() + "/threads.db"
	threadStore, err := sqldb.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer threadStore.Close()

	newProvider := func() *Provider {
		p := NewProvider(
			"sk-test",
			WithProviderBaseURL(ts.URL),
			WithProviderHTTPClient(ts.Client()),
			WithResponsesAPI(true),
			WithResponsesThreadKeyPath("metadata.user_id"),
			WithResponsesThreadPersistence(true),
		)
		p.SetThreadStore(threadStore)
		return p
	}

	buildReq := func(userID string) *domain.CanonicalRequest {
		rawReq := []byte(fmt.Sprintf(`{"model":"gpt-4o-mini","input":"hi","metadata":{"user_id":"%s"}}`, userID))
		return &domain.CanonicalRequest{
			Model:         "gpt-4o-mini",
			Messages:      []domain.Message{{Role: "user", Content: "hi"}},
			RawRequest:    rawReq,
			Metadata:      map[string]string{"user_id": userID},
			SourceAPIType: domain.APITypeOpenAI,
		}
	}

	// First request seeds thread state
	p1 := newProvider()
	resp1, err := p1.Complete(context.Background(), buildReq("alice"))
	if err != nil {
		t.Fatalf("first completion failed: %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("expected one request, got %d", len(requests))
	}
	if _, ok := requests[0]["previous_response_id"]; ok {
		t.Fatalf("first request unexpectedly included previous_response_id")
	}

	// Simulate restart with a fresh provider; thread state should come from SQLite
	p2 := newProvider()
	resp2, err := p2.Complete(context.Background(), buildReq("alice"))
	if err != nil {
		t.Fatalf("second completion failed: %v", err)
	}

	if resp1.ID == "" || resp2.ID == "" {
		t.Fatalf("expected non-empty response IDs (got %q and %q)", resp1.ID, resp2.ID)
	}

	if len(requests) != 2 {
		t.Fatalf("expected two requests, got %d", len(requests))
	}

	prev, ok := requests[1]["previous_response_id"].(string)
	if !ok {
		t.Fatalf("second request missing previous_response_id: %+v", requests[1])
	}
	if prev != resp1.ID {
		t.Fatalf("previous_response_id = %s, want %s", prev, resp1.ID)
	}
}

func TestResponsesAuditEventsLogged(t *testing.T) {
	ctx := context.Background()
	store := memory.New()

	var requests [][]byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		defer r.Body.Close()
		requests = append(requests, body)

		resp := ResponsesResponse{
			ID:        fmt.Sprintf("resp_%d", len(requests)),
			Object:    "response",
			CreatedAt: time.Now().Unix(),
			Status:    "completed",
			Model:     "gpt-4o-mini",
			Output: []ResponsesOutputItem{{
				Type: "message",
				Role: "assistant",
				Content: []ResponsesContentPart{{
					Type: "output_text",
					Text: "hello",
				}},
			}},
			Usage: &ResponsesUsage{
				InputTokens:  4,
				OutputTokens: 6,
				TotalTokens:  10,
			},
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	provider := NewProvider(
		"sk-test",
		WithProviderBaseURL(ts.URL),
		WithProviderHTTPClient(ts.Client()),
		WithResponsesAPI(true),
		WithResponsesThreadKeyPath("metadata.user_id"),
		WithResponsesThreadPersistence(true),
		WithThreadStateStore(store),
		WithEventStore(store),
	)

	rawReq := []byte(`{"model":"gpt-4o-mini","input":"hi","metadata":{"user_id":"audit-user"}}`)
	req := &domain.CanonicalRequest{
		Model:         "gpt-4o-mini",
		Messages:      []domain.Message{{Role: "user", Content: "hi"}},
		RawRequest:    rawReq,
		Metadata:      map[string]string{"user_id": "audit-user", "interaction_id": "int_audit"},
		SourceAPIType: domain.APITypeOpenAI,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}
	if resp.ID == "" {
		t.Fatalf("expected response id to be set")
	}

	events, err := store.ListInteractionEvents(ctx, "int_audit", storage.InteractionListOptions{})
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected events to be recorded")
	}

	var stages []string
	for _, evt := range events {
		stages = append(stages, evt.Stage)
	}

	wantStages := []string{"thread_resolve", "provider_encode", "provider_decode", "thread_update"}
	for _, want := range wantStages {
		found := false
		for _, stage := range stages {
			if stage == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected stage %q to be recorded; got %v", want, stages)
		}
	}

	hash := sha256.Sum256([]byte("sk-test:audit-user"))
	expectedThreadKey := fmt.Sprintf("%x", hash[:])
	for _, evt := range events {
		if evt.Stage == "thread_resolve" || evt.Stage == "thread_update" {
			if evt.ThreadKey != expectedThreadKey {
				t.Fatalf("unexpected thread key %q, want %q", evt.ThreadKey, expectedThreadKey)
			}
		}
	}

	// thread state should be persisted in the backing store
	if stored, _ := store.GetThreadState(expectedThreadKey); stored != resp.ID {
		t.Fatalf("thread state persisted as %q, want %q", stored, resp.ID)
	}
}
