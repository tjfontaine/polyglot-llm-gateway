package conversation

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/memory"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tenant"
)

func TestRecordPersistsWithCancelledContext(t *testing.T) {
	store := memory.New()

	req := &domain.CanonicalRequest{
		Model: "test-model",
		Messages: []domain.Message{
			{Role: "user", Content: "hi"},
		},
	}
	resp := &domain.CanonicalResponse{
		Model: "test-model",
		Choices: []domain.Choice{
			{
				Message: domain.Message{
					Role:    "assistant",
					Content: "hello",
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "tenant", &tenant.Tenant{ID: "tenant-1"})
	cancel() // simulate client disconnect

	convID := Record(ctx, store, "", req, resp, nil)

	conv, err := store.GetConversation(context.Background(), convID)
	if err != nil {
		t.Fatalf("expected conversation to be stored, got error: %v", err)
	}

	if conv.TenantID != "tenant-1" {
		t.Fatalf("expected tenant ID tenant-1, got %s", conv.TenantID)
	}

	if len(conv.Messages) != 2 {
		t.Fatalf("expected 2 messages to be stored, got %d", len(conv.Messages))
	}
}
