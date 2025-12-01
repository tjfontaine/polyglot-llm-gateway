package sqlite

import (
	"context"
	"os"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

func TestSQLiteStore_CreateConversation(t *testing.T) {
	// Use in-memory SQLite with shared cache for testing
	store, err := New("file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	conv := &storage.Conversation{
		ID:       "test-conv-1",
		TenantID: "tenant-1",
		Metadata: map[string]string{"key": "value"},
	}

	err = store.CreateConversation(context.Background(), conv)
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

func TestSQLiteStore_AddMessage(t *testing.T) {
	store, err := New("file:memdb2?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	conv := &storage.Conversation{
		ID:       "test-conv-2",
		TenantID: "tenant-1",
	}

	err = store.CreateConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	msg := &storage.StoredMessage{
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

func TestSQLiteStore_ListConversations(t *testing.T) {
	store, err := New("file:memdb3?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

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

func TestSQLiteStore_DeleteConversation(t *testing.T) {
	store, err := New("file:memdb4?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	conv := &storage.Conversation{
		ID:       "test-conv-3",
		TenantID: "tenant-1",
	}

	err = store.CreateConversation(context.Background(), conv)
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

func TestSQLiteStore_Persistence(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Create store and add data
	store, err := New(tmpfile.Name())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	conv := &storage.Conversation{
		ID:       "persist-test",
		TenantID: "tenant-1",
	}

	err = store.CreateConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	store.Close()

	// Reopen and verify data persisted
	store2, err := New(tmpfile.Name())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store2.Close()

	retrieved, err := store2.GetConversation(context.Background(), "persist-test")
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}

	if retrieved.ID != "persist-test" {
		t.Errorf("ID = %v, want persist-test", retrieved.ID)
	}
}
