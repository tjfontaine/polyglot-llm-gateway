// Package basic provides a basic quality policy with no rate limiting.
package basic

import (
	"context"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// Policy implements ports.QualityPolicy with no restrictions.
// This is the default policy for single-instance deployments.
type Policy struct{}

// NewPolicy creates a new basic policy.
func NewPolicy() *Policy {
	return &Policy{}
}

// CheckRequest always allows requests (no rate limiting).
func (p *Policy) CheckRequest(ctx context.Context, req *ports.PolicyRequest) (*ports.PolicyDecision, error) {
	return &ports.PolicyDecision{
		Allow:  true,
		Reason: "basic policy allows all requests",
	}, nil
}

// RecordUsage records usage but does nothing with it.
func (p *Policy) RecordUsage(ctx context.Context, usage *ports.UsageRecord) error {
	// No-op: basic policy doesn't enforce quotas
	return nil
}
