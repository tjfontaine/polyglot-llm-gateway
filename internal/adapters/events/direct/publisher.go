// Package direct provides a direct event publisher that writes to storage.
package direct

import (
	"context"
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// Publisher implements ports.EventPublisher by writing directly to storage.
// This is the default implementation for single-instance deployments.
type Publisher struct {
	store ports.InteractionStore
}

// NewPublisher creates a new direct event publisher.
func NewPublisher(store ports.StorageProvider) (*Publisher, error) {
	if store == nil {
		return nil, fmt.Errorf("storage provider required")
	}

	interactionStore, ok := store.(ports.InteractionStore)
	if !ok {
		return nil, fmt.Errorf("storage provider must implement InteractionStore")
	}

	return &Publisher{
		store: interactionStore,
	}, nil
}

// Publish writes a lifecycle event directly to storage.
func (p *Publisher) Publish(ctx context.Context, event *domain.LifecycleEvent) error {
	// Convert LifecycleEvent to InteractionEvent for storage
	// The storage layer expects the detailed InteractionEvent type
	storageEvent := &domain.InteractionEvent{
		InteractionID: event.InteractionID,
		Stage:         string(event.Type),
		Direction:     "internal",
		CreatedAt:     event.Timestamp,
	}

	return p.store.AppendInteractionEvent(ctx, storageEvent)
}

// Close is a no-op for direct publisher.
func (p *Publisher) Close() error {
	return nil
}
