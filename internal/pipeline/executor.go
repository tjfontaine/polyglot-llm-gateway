package pipeline

import (
	"context"
	"fmt"
	"sort"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// Executor orchestrates pipeline stage execution.
// It maintains ordered lists of pre and post stages and executes them sequentially.
type Executor struct {
	preStages  []ports.Stage
	postStages []ports.Stage
}

// ExecutorConfig configures an executor from stage configurations.
type ExecutorConfig struct {
	Stages []StageConfig
}

// StageConfig is the configuration for a single stage.
type StageConfig struct {
	Name  string
	Type  ports.StageType
	Order int
	Stage ports.Stage
}

// NewExecutor creates an executor from configuration.
func NewExecutor(cfg ExecutorConfig) *Executor {
	var preStages, postStages []StageConfig

	for _, s := range cfg.Stages {
		switch s.Type {
		case ports.StagePre:
			preStages = append(preStages, s)
		case ports.StagePost:
			postStages = append(postStages, s)
		}
	}

	// Sort by order
	sort.Slice(preStages, func(i, j int) bool {
		return preStages[i].Order < preStages[j].Order
	})
	sort.Slice(postStages, func(i, j int) bool {
		return postStages[i].Order < postStages[j].Order
	})

	e := &Executor{
		preStages:  make([]ports.Stage, len(preStages)),
		postStages: make([]ports.Stage, len(postStages)),
	}
	for i, s := range preStages {
		e.preStages[i] = s.Stage
	}
	for i, s := range postStages {
		e.postStages[i] = s.Stage
	}

	return e
}

// RunPre executes all pre-stages in order.
// Returns the (possibly mutated) request or an error if denied.
func (e *Executor) RunPre(ctx context.Context, req *domain.CanonicalRequest, meta map[string]any) (*domain.CanonicalRequest, error) {
	if len(e.preStages) == 0 {
		return req, nil
	}

	currentReq := req
	for _, stage := range e.preStages {
		input := &ports.StageInput{
			Phase:    "request",
			Request:  currentReq,
			Metadata: meta,
		}

		output, err := stage.Process(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("pipeline stage %s error: %w", stage.Name(), err)
		}

		switch output.Action {
		case ports.ActionDeny:
			reason := output.DenyReason
			if reason == "" {
				reason = "denied by pipeline stage " + stage.Name()
			}
			return nil, &DeniedError{
				StageName: stage.Name(),
				Reason:    reason,
			}
		case ports.ActionMutate:
			if output.Request != nil {
				currentReq = output.Request
			}
		case ports.ActionAllow:
			// Continue with current request
		}
	}

	return currentReq, nil
}

// RunPost executes all post-stages in order.
// Returns the (possibly mutated) response or an error if denied.
func (e *Executor) RunPost(ctx context.Context, req *domain.CanonicalRequest, resp *domain.CanonicalResponse, meta map[string]any) (*domain.CanonicalResponse, error) {
	if len(e.postStages) == 0 {
		return resp, nil
	}

	currentResp := resp
	for _, stage := range e.postStages {
		input := &ports.StageInput{
			Phase:    "response",
			Request:  req,
			Response: currentResp,
			Metadata: meta,
		}

		output, err := stage.Process(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("pipeline stage %s error: %w", stage.Name(), err)
		}

		switch output.Action {
		case ports.ActionDeny:
			reason := output.DenyReason
			if reason == "" {
				reason = "denied by pipeline stage " + stage.Name()
			}
			return nil, &DeniedError{
				StageName: stage.Name(),
				Reason:    reason,
			}
		case ports.ActionMutate:
			if output.Response != nil {
				currentResp = output.Response
			}
		case ports.ActionAllow:
			// Continue with current response
		}
	}

	return currentResp, nil
}

// HasPreStages returns true if there are any pre-stages configured.
func (e *Executor) HasPreStages() bool {
	return len(e.preStages) > 0
}

// HasPostStages returns true if there are any post-stages configured.
func (e *Executor) HasPostStages() bool {
	return len(e.postStages) > 0
}

// DeniedError is returned when a pipeline stage denies a request or response.
type DeniedError struct {
	StageName string
	Reason    string
}

func (e *DeniedError) Error() string {
	return fmt.Sprintf("pipeline denied by %s: %s", e.StageName, e.Reason)
}

// IsDenied returns true if the error is a pipeline denial.
func IsDenied(err error) bool {
	_, ok := err.(*DeniedError)
	return ok
}

// Ensure Executor implements the interface.
var _ ports.PipelineExecutor = (*Executor)(nil)
