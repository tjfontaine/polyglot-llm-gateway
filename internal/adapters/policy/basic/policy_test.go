package basic

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

func TestNewPolicy(t *testing.T) {
	policy := NewPolicy()
	if policy == nil {
		t.Fatal("NewPolicy() returned nil")
	}
}

func TestCheckRequest_AlwaysAllows(t *testing.T) {
	policy := NewPolicy()
	ctx := context.Background()

	req := &ports.PolicyRequest{
		TenantID: "test-tenant",
		Model:    "gpt-4",
		Tokens:   1000,
	}

	decision, err := policy.CheckRequest(ctx, req)
	if err != nil {
		t.Fatalf("CheckRequest failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Expected decision.Allow to be true")
	}

	if decision.Reason != "basic policy allows all requests" {
		t.Errorf("Unexpected reason: %s", decision.Reason)
	}
}

func TestCheckRequest_NilRequest(t *testing.T) {
	policy := NewPolicy()
	ctx := context.Background()

	decision, err := policy.CheckRequest(ctx, nil)
	if err != nil {
		t.Fatalf("CheckRequest failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Expected decision.Allow to be true even with nil request")
	}
}

func TestRecordUsage_NoOp(t *testing.T) {
	policy := NewPolicy()
	ctx := context.Background()

	usage := &ports.UsageRecord{
		TenantID:         "test-tenant",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 200,
	}

	err := policy.RecordUsage(ctx, usage)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}
	// No-op, just verify it doesn't error
}

func TestRecordUsage_NilUsage(t *testing.T) {
	policy := NewPolicy()
	ctx := context.Background()

	err := policy.RecordUsage(ctx, nil)
	if err != nil {
		t.Fatalf("RecordUsage failed with nil:  %v", err)
	}
}
