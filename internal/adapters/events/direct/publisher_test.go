package direct

import (
	"context"
	"testing"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/storage/sqlite"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

func TestNewPublisher(t *testing.T) {
	// Use real SQLite in-memory for testing
	store, _ := sqlite.NewProvider(":memory:")
	defer store.Close()

	publisher, err := NewPublisher(store)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	if publisher == nil {
		t.Fatal("NewPublisher returned nil")
	}
}

func TestNewPublisher_NilStorage(t *testing.T) {
	_, err := NewPublisher(nil)
	if err == nil {
		t.Error("Expected error for nil storage")
	}
	if err.Error() != "storage provider required" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPublish(t *testing.T) {
	store, _ := sqlite.NewProvider(":memory:")
	defer store.Close()

	publisher, _ := NewPublisher(store)
	ctx := context.Background()

	event := &domain.LifecycleEvent{
		Type:          domain.LifecycleEventStarted,
		InteractionID: "test-interaction-123",
		Timestamp:     time.Now(),
		Data:          domain.LifecycleStartedData{},
	}

	err := publisher.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	// Event published successfully - storage integration tested elsewhere
}

func TestClose(t *testing.T) {
	store, _ := sqlite.NewProvider(":memory:")
	publisher, _ := NewPublisher(store)

	err := publisher.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Clean up
	store.Close()
}
