package memory

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

func TestMemoryStore_CreateConversation(t *testing.T) {
	store := New()

	conv := &storage.Conversation{
		ID:       "test-conv-1",
		TenantID: "tenant-1",
		Metadata: map[string]string{"key": "value"},
	}

	err := store.CreateConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	// Verify conversation was created
	retrieved, err := store.GetConversation(context.Background(), "test-conv-1")
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}

	if retrieved.ID != conv.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, conv.ID)
	}
	if retrieved.TenantID != conv.TenantID {
		t.Errorf("TenantID = %v, want %v", retrieved.TenantID, conv.TenantID)
	}
}

func TestMemoryStore_AddMessage(t *testing.T) {
	store := New()

	conv := &storage.Conversation{
		ID:       "test-conv-2",
		TenantID: "tenant-1",
	}

	err := store.CreateConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	msg := &storage.Message{
		ID:      "msg-1",
		Role:    "user",
		Content: "Hello",
	}

	err = store.AddMessage(context.Background(), "test-conv-2", msg)
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	// Verify message was added
	retrieved, err := store.GetConversation(context.Background(), "test-conv-2")
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}

	if len(retrieved.Messages) != 1 {
		t.Fatalf("Messages count = %d, want 1", len(retrieved.Messages))
	}

	if retrieved.Messages[0].Content != "Hello" {
		t.Errorf("Message content = %v, want Hello", retrieved.Messages[0].Content)
	}
}

func TestMemoryStore_ListConversations(t *testing.T) {
	store := New()

	// Create multiple conversations
	for i := 0; i < 5; i++ {
		conv := &storage.Conversation{
			ID:       "conv-" + string(rune('0'+i)),
			TenantID: "tenant-1",
		}
		if err := store.CreateConversation(context.Background(), conv); err != nil {
			t.Fatalf("CreateConversation() error = %v", err)
		}
	}

	// List with limit
	opts := storage.ListOptions{
		TenantID: "tenant-1",
		Limit:    3,
		Offset:   0,
	}

	convs, err := store.ListConversations(context.Background(), opts)
	if err != nil {
		t.Fatalf("ListConversations() error = %v", err)
	}

	if len(convs) != 3 {
		t.Errorf("ListConversations() count = %d, want 3", len(convs))
	}
}

func TestMemoryStore_DeleteConversation(t *testing.T) {
	store := New()

	conv := &storage.Conversation{
		ID:       "test-conv-3",
		TenantID: "tenant-1",
	}

	err := store.CreateConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	// Delete conversation
	err = store.DeleteConversation(context.Background(), "test-conv-3")
	if err != nil {
		t.Fatalf("DeleteConversation() error = %v", err)
	}

	// Verify it's deleted
	_, err = store.GetConversation(context.Background(), "test-conv-3")
	if err == nil {
		t.Error("GetConversation() expected error for deleted conversation")
	}
}
