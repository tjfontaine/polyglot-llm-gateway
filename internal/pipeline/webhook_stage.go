package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// WebhookStage calls an external HTTP endpoint for pipeline processing.
type WebhookStage struct {
	name      string
	stageType ports.StageType
	url       string
	timeout   time.Duration
	onError   ports.StageAction // Action to take on error (allow or deny)
	retries   int
	squelch   bool // Post-stage: suppress response if denied
	headers   map[string]string
	client    *http.Client
}

// WebhookStageConfig configures a webhook stage.
type WebhookStageConfig struct {
	Name    string
	Type    ports.StageType
	URL     string
	Timeout time.Duration
	OnError ports.StageAction // "allow" or "deny" (default: deny)
	Retries int
	Squelch bool
	Headers map[string]string
}

// NewWebhookStage creates a new webhook stage.
func NewWebhookStage(cfg WebhookStageConfig) *WebhookStage {
	onError := cfg.OnError
	if onError == "" {
		onError = ports.ActionDeny // Default to fail-closed
	}

	return &WebhookStage{
		name:      cfg.Name,
		stageType: cfg.Type,
		url:       cfg.URL,
		timeout:   cfg.Timeout,
		onError:   onError,
		retries:   cfg.Retries,
		squelch:   cfg.Squelch,
		headers:   cfg.Headers,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Name returns the stage identifier.
func (s *WebhookStage) Name() string {
	return s.name
}

// Type returns when this stage runs.
func (s *WebhookStage) Type() ports.StageType {
	return s.stageType
}

// Squelch returns whether this stage should suppress responses on deny.
func (s *WebhookStage) Squelch() bool {
	return s.squelch
}

// Process executes the webhook call.
func (s *WebhookStage) Process(ctx context.Context, in *ports.StageInput) (*ports.StageOutput, error) {
	var lastErr error

	// Retry loop
	attempts := s.retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		output, err := s.doRequest(ctx, in)
		if err == nil {
			return output, nil
		}
		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}
	}

	// All retries failed - apply onError behavior
	return s.handleError(lastErr)
}

func (s *WebhookStage) doRequest(ctx context.Context, in *ports.StageInput) (*ports.StageOutput, error) {
	// Marshal input
	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal stage input: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add custom headers
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse output
	var output ports.StageOutput
	if err := json.Unmarshal(respBody, &output); err != nil {
		return nil, fmt.Errorf("unmarshal stage output: %w", err)
	}

	// Validate action
	switch output.Action {
	case ports.ActionAllow, ports.ActionDeny, ports.ActionMutate:
		// Valid
	case "":
		output.Action = ports.ActionAllow // Default to allow if not specified
	default:
		return nil, fmt.Errorf("invalid action from webhook: %s", output.Action)
	}

	return &output, nil
}

func (s *WebhookStage) handleError(err error) (*ports.StageOutput, error) {
	switch s.onError {
	case ports.ActionAllow:
		// Fail-open: log and allow
		// TODO: Add structured logging
		return &ports.StageOutput{Action: ports.ActionAllow}, nil
	case ports.ActionDeny:
		// Fail-closed: return error with deny reason
		return &ports.StageOutput{
			Action:     ports.ActionDeny,
			DenyReason: fmt.Sprintf("webhook error: %v", err),
		}, nil
	default:
		// Unknown onError, fail-closed for safety
		return nil, fmt.Errorf("webhook stage %s failed: %w", s.name, err)
	}
}

// Ensure WebhookStage implements the interface.
var _ ports.Stage = (*WebhookStage)(nil)
