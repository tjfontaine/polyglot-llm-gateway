package sqlite

import (
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

func TestNewProvider(t *testing.T) {
	// Use in-memory SQLite for testing
	provider, err := NewProvider(":memory:")
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil")
	}

	// Verify it implements StorageProvider
	var _ ports.StorageProvider = provider

	// Clean up
	provider.Close()
}

func TestNewProvider_InvalidPath(t *testing.T) {
	// Try to create invalid path
	_, err := NewProvider("/invalid/path/that/does/not/exist/test.db")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestProvider_ImplementsInterfaces(t *testing.T) {
	provider, _ := NewProvider(":memory:")
	defer provider.Close()

	// Verify all required interfaces are implemented
	var _ ports.ConversationStore = provider
	var _ ports.InteractionStore = provider
	var _ ports.ResponseStore = provider
	var _ ports.ShadowStore = provider

	// ThreadStateStore methods should be available through embedded Store
	_ = provider.Store
}

func TestProvider_Close(t *testing.T) {
	provider, _ := NewProvider(":memory:")

	err := provider.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
