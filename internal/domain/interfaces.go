package domain

import (
	"context"
)

// Provider defines the interface for LLM providers.
type Provider interface {
	Name() string

	// Complete handles unary requests (non-streaming)
	Complete(ctx context.Context, req *CanonicalRequest) (*CanonicalResponse, error)

	// Stream returns a channel of events.
	// The channel MUST be closed by the provider when done.
	Stream(ctx context.Context, req *CanonicalRequest) (<-chan CanonicalEvent, error)

	// ListModels returns the models supported by the provider/frontdoor pairing.
	ListModels(ctx context.Context) (*ModelList, error)
}
