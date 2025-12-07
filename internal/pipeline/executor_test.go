package pipeline

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// mockStage is a test helper that records calls and returns configured responses.
type mockStage struct {
	name      string
	stageType ports.StageType
	output    *ports.StageOutput
	err       error
	calls     []*ports.StageInput
}

func (s *mockStage) Name() string          { return s.name }
func (s *mockStage) Type() ports.StageType { return s.stageType }

func (s *mockStage) Process(ctx context.Context, in *ports.StageInput) (*ports.StageOutput, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	if s.output != nil {
		return s.output, nil
	}
	return &ports.StageOutput{Action: ports.ActionAllow}, nil
}

func TestExecutor_RunPre_Empty(t *testing.T) {
	e := NewExecutor(ExecutorConfig{})
	req := &domain.CanonicalRequest{Model: "gpt-4"}

	result, err := e.RunPre(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != req {
		t.Error("expected same request when no stages")
	}
}

func TestExecutor_RunPre_Allow(t *testing.T) {
	stage := &mockStage{
		name:      "test-stage",
		stageType: ports.StagePre,
		output:    &ports.StageOutput{Action: ports.ActionAllow},
	}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{{
			Name:  "test-stage",
			Type:  ports.StagePre,
			Order: 1,
			Stage: stage,
		}},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	result, err := e.RunPre(context.Background(), req, map[string]any{"app": "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != req {
		t.Error("expected same request on allow")
	}
	if len(stage.calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(stage.calls))
	}
	if stage.calls[0].Phase != "request" {
		t.Errorf("expected phase 'request', got %q", stage.calls[0].Phase)
	}
}

func TestExecutor_RunPre_Deny(t *testing.T) {
	stage := &mockStage{
		name:      "deny-stage",
		stageType: ports.StagePre,
		output: &ports.StageOutput{
			Action:     ports.ActionDeny,
			DenyReason: "blocked by policy",
		},
	}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{{
			Name:  "deny-stage",
			Type:  ports.StagePre,
			Order: 1,
			Stage: stage,
		}},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	_, err := e.RunPre(context.Background(), req, nil)

	if err == nil {
		t.Fatal("expected error on deny")
	}
	if !IsDenied(err) {
		t.Errorf("expected DeniedError, got %T", err)
	}

	denied := err.(*DeniedError)
	if denied.StageName != "deny-stage" {
		t.Errorf("expected stage name 'deny-stage', got %q", denied.StageName)
	}
	if denied.Reason != "blocked by policy" {
		t.Errorf("unexpected reason: %s", denied.Reason)
	}
}

func TestExecutor_RunPre_Mutate(t *testing.T) {
	mutatedReq := &domain.CanonicalRequest{Model: "gpt-4-turbo"}
	stage := &mockStage{
		name:      "mutate-stage",
		stageType: ports.StagePre,
		output: &ports.StageOutput{
			Action:  ports.ActionMutate,
			Request: mutatedReq,
		},
	}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{{
			Name:  "mutate-stage",
			Type:  ports.StagePre,
			Order: 1,
			Stage: stage,
		}},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	result, err := e.RunPre(context.Background(), req, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mutatedReq {
		t.Error("expected mutated request")
	}
	if result.Model != "gpt-4-turbo" {
		t.Errorf("expected model 'gpt-4-turbo', got %q", result.Model)
	}
}

func TestExecutor_RunPre_OrderedExecution(t *testing.T) {
	var callOrder []string

	stage1 := &mockStage{
		name:      "first",
		stageType: ports.StagePre,
	}
	stage1Wrapper := &orderTrackingStage{mockStage: stage1, callOrder: &callOrder}

	stage2 := &mockStage{
		name:      "second",
		stageType: ports.StagePre,
	}
	stage2Wrapper := &orderTrackingStage{mockStage: stage2, callOrder: &callOrder}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{
			{Name: "second", Type: ports.StagePre, Order: 2, Stage: stage2Wrapper},
			{Name: "first", Type: ports.StagePre, Order: 1, Stage: stage1Wrapper},
		},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	_, err := e.RunPre(context.Background(), req, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(callOrder) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(callOrder))
	}
	if callOrder[0] != "first" || callOrder[1] != "second" {
		t.Errorf("unexpected order: %v", callOrder)
	}
}

type orderTrackingStage struct {
	*mockStage
	callOrder *[]string
}

func (s *orderTrackingStage) Process(ctx context.Context, in *ports.StageInput) (*ports.StageOutput, error) {
	*s.callOrder = append(*s.callOrder, s.name)
	return s.mockStage.Process(ctx, in)
}

func TestExecutor_RunPost_Allow(t *testing.T) {
	stage := &mockStage{
		name:      "post-stage",
		stageType: ports.StagePost,
		output:    &ports.StageOutput{Action: ports.ActionAllow},
	}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{{
			Name:  "post-stage",
			Type:  ports.StagePost,
			Order: 1,
			Stage: stage,
		}},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	resp := &domain.CanonicalResponse{
		ID:    "resp-123",
		Model: "gpt-4",
		Choices: []domain.Choice{{
			Index:   0,
			Message: domain.Message{Role: "assistant", Content: "Hello"},
		}},
	}
	result, err := e.RunPost(context.Background(), req, resp, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != resp {
		t.Error("expected same response on allow")
	}
	if len(stage.calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(stage.calls))
	}
	if stage.calls[0].Phase != "response" {
		t.Errorf("expected phase 'response', got %q", stage.calls[0].Phase)
	}
	if stage.calls[0].Response != resp {
		t.Error("expected response to be passed to stage")
	}
}

func TestExecutor_RunPost_Mutate(t *testing.T) {
	mutatedResp := &domain.CanonicalResponse{
		ID:    "resp-456",
		Model: "gpt-4",
		Choices: []domain.Choice{{
			Index:   0,
			Message: domain.Message{Role: "assistant", Content: "Filtered response"},
		}},
	}
	stage := &mockStage{
		name:      "filter-stage",
		stageType: ports.StagePost,
		output: &ports.StageOutput{
			Action:   ports.ActionMutate,
			Response: mutatedResp,
		},
	}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{{
			Name:  "filter-stage",
			Type:  ports.StagePost,
			Order: 1,
			Stage: stage,
		}},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	resp := &domain.CanonicalResponse{
		ID:    "resp-123",
		Model: "gpt-4",
		Choices: []domain.Choice{{
			Index:   0,
			Message: domain.Message{Role: "assistant", Content: "Original response"},
		}},
	}
	result, err := e.RunPost(context.Background(), req, resp, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mutatedResp {
		t.Error("expected mutated response")
	}
	if len(result.Choices) == 0 || result.Choices[0].Message.Content != "Filtered response" {
		t.Error("expected mutated content")
	}
}

func TestExecutor_StageTypeSeparation(t *testing.T) {
	preStage := &mockStage{name: "pre", stageType: ports.StagePre}
	postStage := &mockStage{name: "post", stageType: ports.StagePost}

	e := NewExecutor(ExecutorConfig{
		Stages: []StageConfig{
			{Name: "pre", Type: ports.StagePre, Order: 1, Stage: preStage},
			{Name: "post", Type: ports.StagePost, Order: 1, Stage: postStage},
		},
	})

	req := &domain.CanonicalRequest{Model: "gpt-4"}
	resp := &domain.CanonicalResponse{ID: "resp-123", Model: "gpt-4"}

	// Pre should only call pre-stage
	_, _ = e.RunPre(context.Background(), req, nil)
	if len(preStage.calls) != 1 {
		t.Errorf("expected 1 pre call, got %d", len(preStage.calls))
	}
	if len(postStage.calls) != 0 {
		t.Errorf("expected 0 post calls during pre, got %d", len(postStage.calls))
	}

	// Post should only call post-stage
	_, _ = e.RunPost(context.Background(), req, resp, nil)
	if len(postStage.calls) != 1 {
		t.Errorf("expected 1 post call, got %d", len(postStage.calls))
	}
}
