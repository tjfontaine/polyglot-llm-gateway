package sqldb

import (
"context"
"testing"

"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

func TestSQLDBStore_CreateConversation(t *testing.T) {
	store, err := NewSQLite("file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
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

func TestSQLDBStore_AddMessage(t *testing.T) {
	store, err := NewSQLite("file:memdb2?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
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

func TestSQLDBStore_ListConversations(t *testing.T) {
	store, err := NewSQLite("file:memdb3?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
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

	// List conversations
	opts := storage.ListOptions{
		TenantID: "tenant-1",
		Limit:    10,
	}
	conversations, err := store.ListConversations(context.Background(), opts)
	if err != nil {
		t.Fatalf("ListConversations() error = %v", err)
	}

	if len(conversations) != 5 {
		t.Errorf("Conversations count = %d, want 5", len(conversations))
	}
}

func TestSQLDBStore_DeleteConversation(t *testing.T) {
	store, err := NewSQLite("file:memdb4?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	defer store.Close()

	conv := &storage.Conversation{
		ID:       "test-conv-delete",
		TenantID: "tenant-1",
	}

	err = store.CreateConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	err = store.DeleteConversation(context.Background(), "test-conv-delete")
	if err != nil {
		t.Fatalf("DeleteConversation() error = %v", err)
	}

	// Verify conversation was deleted
	_, err = store.GetConversation(context.Background(), "test-conv-delete")
	if err == nil {
		t.Error("Expected error getting deleted conversation")
	}
}

func TestSQLDBStore_ThreadState(t *testing.T) {
	store, err := NewSQLite("file:memdb5?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	defer store.Close()

	// Set thread state
	err = store.SetThreadState("thread-1", "response-1")
	if err != nil {
		t.Fatalf("SetThreadState() error = %v", err)
	}

	// Get thread state
	respID, err := store.GetThreadState("thread-1")
	if err != nil {
		t.Fatalf("GetThreadState() error = %v", err)
	}

	if respID != "response-1" {
		t.Errorf("Response ID = %v, want response-1", respID)
	}

	// Update thread state
	err = store.SetThreadState("thread-1", "response-2")
	if err != nil {
		t.Fatalf("SetThreadState() error = %v", err)
	}

	respID, err = store.GetThreadState("thread-1")
	if err != nil {
		t.Fatalf("GetThreadState() error = %v", err)
	}

	if respID != "response-2" {
		t.Errorf("Response ID = %v, want response-2", respID)
	}
}

func TestSQLDBStore_DialectAccessor(t *testing.T) {
	store, err := NewSQLite("file:memdb6?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	defer store.Close()

	// Test Dialect accessor
	d := store.Dialect()
	if d.Name() != "sqlite" {
		t.Errorf("Dialect name = %v, want sqlite", d.Name())
	}

	// Test DB accessor
	db := store.DB()
	if db == nil {
		t.Error("DB() returned nil")
	}
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		Driver: "sqlite",
		DSN:    "file:memdb7?mode=memory&cache=shared",
	}

	store, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	if store.Dialect().Name() != "sqlite" {
		t.Errorf("Dialect name = %v, want sqlite", store.Dialect().Name())
	}
}

func TestNew_UnsupportedDriver(t *testing.T) {
	cfg := Config{
		Driver: "unsupported",
		DSN:    "test",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}
