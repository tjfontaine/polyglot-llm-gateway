// Package ports defines the core interfaces for the gateway.
// This file contains the pipeline stage interfaces for request/response mutation.
package ports

import (
	"context"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

// StageType determines when the stage runs in the pipeline.
type StageType string

const (
	// StagePre runs before routing/provider call.
	StagePre StageType = "pre"
	// StagePost runs after response is received.
	StagePost StageType = "post"
)

// StageAction is the result action from a pipeline stage.
type StageAction string

const (
	// ActionAllow permits the request/response to continue.
	ActionAllow StageAction = "allow"
	// ActionDeny blocks the request/response.
	ActionDeny StageAction = "deny"
	// ActionMutate allows with mutations applied.
	ActionMutate StageAction = "mutate"
)

// StageInput is the data sent to a pipeline stage.
type StageInput struct {
	// Phase indicates whether this is a "request" or "response" phase.
	Phase string `json:"phase"`
	// Request is the canonical request (always present).
	Request *domain.CanonicalRequest `json:"request"`
	// Response is the canonical response (only present in post-stage).
	Response *domain.CanonicalResponse `json:"response,omitempty"`
	// Metadata contains contextual information about the request.
	Metadata map[string]any `json:"metadata"`
}

// StageOutput is returned from a pipeline stage.
type StageOutput struct {
	// Action indicates what should happen: allow, deny, or mutate.
	Action StageAction `json:"action"`
	// Request is the mutated request (only if Action is mutate and Phase is request).
	Request *domain.CanonicalRequest `json:"request,omitempty"`
	// Response is the mutated response (only if Action is mutate and Phase is response).
	Response *domain.CanonicalResponse `json:"response,omitempty"`
	// DenyReason explains why the request/response was denied.
	DenyReason string `json:"deny_reason,omitempty"`
}

// Stage processes a request or response in the pipeline.
type Stage interface {
	// Name returns the unique identifier for this stage.
	Name() string
	// Type returns when this stage runs (pre or post).
	Type() StageType
	// Process executes the stage logic.
	Process(ctx context.Context, in *StageInput) (*StageOutput, error)
}

// PipelineExecutor orchestrates pipeline stage execution.
type PipelineExecutor interface {
	// RunPre executes all pre-stages in order.
	// Returns the (possibly mutated) request or an error if denied.
	RunPre(ctx context.Context, req *domain.CanonicalRequest, meta map[string]any) (*domain.CanonicalRequest, error)

	// RunPost executes all post-stages in order.
	// Returns the (possibly mutated) response or an error if denied.
	RunPost(ctx context.Context, req *domain.CanonicalRequest, resp *domain.CanonicalResponse, meta map[string]any) (*domain.CanonicalResponse, error)
}
