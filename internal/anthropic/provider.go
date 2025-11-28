// Provider implements the domain.Provider interface for the Anthropic API.
package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

const (
	// defaultMaxRetries is the default number of retry attempts for overload errors.
	defaultMaxRetries = 2
	// defaultBaseDelay is the base delay for exponential backoff.
	defaultBaseDelay = 500 * time.Millisecond
	// defaultMaxDelay caps the backoff delay.
	defaultMaxDelay = 5 * time.Second
)

// ProviderOption configures the provider.
type ProviderOption func(*Provider)

// WithProviderBaseURL sets a custom base URL for the API.
func WithProviderBaseURL(baseURL string) ProviderOption {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

// WithProviderHTTPClient sets a custom HTTP client.
func WithProviderHTTPClient(httpClient *http.Client) ProviderOption {
	return func(p *Provider) {
		p.httpClient = httpClient
	}
}

// WithMaxRetries sets the maximum number of retries for overload errors.
func WithMaxRetries(maxRetries int) ProviderOption {
	return func(p *Provider) {
		p.maxRetries = maxRetries
	}
}

// WithLogger sets the logger for the provider.
func WithLogger(logger *slog.Logger) ProviderOption {
	return func(p *Provider) {
		p.logger = logger
	}
}

// Provider implements the domain.Provider interface using our custom Anthropic client.
type Provider struct {
	client     *Client
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	logger     *slog.Logger
}

// NewProvider creates a new Anthropic provider.
func NewProvider(apiKey string, opts ...ProviderOption) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		maxRetries: defaultMaxRetries,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(p)
	}

	// Build client options
	var clientOpts []ClientOption
	if p.baseURL != "" {
		clientOpts = append(clientOpts, WithBaseURL(p.baseURL))
	}
	if p.httpClient != nil {
		clientOpts = append(clientOpts, WithHTTPClient(p.httpClient))
	}

	p.client = NewClient(apiKey, clientOpts...)
	return p
}

// isOverloadedError checks if the error is an Anthropic 529 overloaded error.
func isOverloadedError(err error) bool {
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Type == domain.ErrorTypeOverloaded
	}
	return false
}

// calculateBackoff returns the delay for the given retry attempt using exponential backoff.
func calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(defaultBaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(defaultMaxDelay) {
		delay = float64(defaultMaxDelay)
	}
	return time.Duration(delay)
}

func (p *Provider) Name() string {
	return "anthropic"
}

func (p *Provider) APIType() domain.APIType {
	return domain.APITypeAnthropic
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	// Use codec to convert canonical request to API request
	apiReq := CanonicalToAPIRequest(req)

	opts := &RequestOptions{
		UserAgent: req.UserAgent,
	}

	var respWithHeaders *MessagesResponseWithHeaders
	var lastErr error

	// Retry loop for handling 529 overloaded errors
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		respWithHeaders, lastErr = p.client.CreateMessage(ctx, apiReq, opts)
		if lastErr == nil {
			// Success - convert and return (including rate limit headers)
			return APIResponseToCanonicalWithRateLimits(respWithHeaders.Response, respWithHeaders.RateLimits), nil
		}

		// Check if this is a retryable overload error
		if !isOverloadedError(lastErr) {
			// Non-retryable error, return immediately
			return nil, lastErr
		}

		// Don't retry if this was our last attempt
		if attempt == p.maxRetries {
			break
		}

		// Calculate backoff delay
		delay := calculateBackoff(attempt)
		p.logger.Warn("Anthropic overloaded, retrying",
			slog.Int("attempt", attempt+1),
			slog.Int("max_retries", p.maxRetries),
			slog.Duration("backoff", delay),
		)

		// Wait before retrying, respecting context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted, return a 503-mapped overloaded error
	p.logger.Error("Anthropic overloaded after all retries",
		slog.Int("retries", p.maxRetries),
		slog.String("error", lastErr.Error()),
	)
	return nil, domain.ErrOverloaded("Anthropic API is overloaded after retries. Please try again later.").
		WithStatusCode(http.StatusServiceUnavailable)
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	// Use codec to convert canonical request to API request
	apiReq := CanonicalToAPIRequest(req)

	opts := &RequestOptions{
		UserAgent: req.UserAgent,
	}

	var stream <-chan StreamEventResult
	var lastErr error

	// Retry loop for handling 529 overloaded errors
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		stream, lastErr = p.client.StreamMessage(ctx, apiReq, opts)
		if lastErr == nil {
			// Success
			break
		}

		// Check if this is a retryable overload error
		if !isOverloadedError(lastErr) {
			// Non-retryable error, return immediately
			return nil, lastErr
		}

		// Don't retry if this was our last attempt
		if attempt == p.maxRetries {
			break
		}

		// Calculate backoff delay
		delay := calculateBackoff(attempt)
		p.logger.Warn("Anthropic overloaded on stream, retrying",
			slog.Int("attempt", attempt+1),
			slog.Int("max_retries", p.maxRetries),
			slog.Duration("backoff", delay),
		)

		// Wait before retrying, respecting context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// If we exhausted retries
	if lastErr != nil {
		p.logger.Error("Anthropic overloaded on stream after all retries",
			slog.Int("retries", p.maxRetries),
			slog.String("error", lastErr.Error()),
		)
		return nil, domain.ErrOverloaded("Anthropic API is overloaded after retries. Please try again later.").
			WithStatusCode(http.StatusServiceUnavailable)
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)

		var inputTokens, outputTokens int
		var stopReason string

		// Track active tool calls by content block index
		activeToolCalls := make(map[int]*domain.ToolCallChunk)

		for result := range stream {
			if result.Err != nil {
				out <- domain.CanonicalEvent{Error: result.Err}
				return
			}

			switch result.EventType {
			case "message_start":
				event, err := result.ParseMessageStart()
				if err != nil {
					out <- domain.CanonicalEvent{Error: fmt.Errorf("parse message_start: %w", err)}
					return
				}
				// Capture initial usage
				inputTokens = event.Message.Usage.InputTokens
				out <- domain.CanonicalEvent{Role: event.Message.Role}

			case "content_block_start":
				event, err := result.ParseContentBlockStart()
				if err != nil {
					out <- domain.CanonicalEvent{Error: fmt.Errorf("parse content_block_start: %w", err)}
					return
				}

				// Handle tool_use content block start
				if event.ContentBlock.Type == "tool_use" {
					toolCall := &domain.ToolCallChunk{
						Index: event.Index,
						ID:    event.ContentBlock.ID,
						Type:  "function",
					}
					toolCall.Function.Name = event.ContentBlock.Name
					activeToolCalls[event.Index] = toolCall

					// Send initial tool call event
					out <- domain.CanonicalEvent{
						Type:     domain.EventTypeContentBlockStart,
						Index:    event.Index,
						ToolCall: toolCall,
					}
				}

			case "content_block_delta":
				event, err := result.ParseContentBlockDelta()
				if err != nil {
					out <- domain.CanonicalEvent{Error: fmt.Errorf("parse content_block_delta: %w", err)}
					return
				}

				switch event.Delta.Type {
				case "text_delta":
					out <- domain.CanonicalEvent{ContentDelta: event.Delta.Text}

				case "input_json_delta":
					// Tool call argument streaming
					if tc, ok := activeToolCalls[event.Index]; ok {
						tc.Function.Arguments += event.Delta.PartialJSON
						out <- domain.CanonicalEvent{
							Type:     domain.EventTypeContentBlockDelta,
							Index:    event.Index,
							ToolCall: tc,
						}
					}
				}

			case "content_block_stop":
				event, err := result.ParseContentBlockStop()
				if err != nil {
					out <- domain.CanonicalEvent{Error: fmt.Errorf("parse content_block_stop: %w", err)}
					return
				}

				// Finalize tool call if this was a tool_use block
				if tc, ok := activeToolCalls[event.Index]; ok {
					out <- domain.CanonicalEvent{
						Type:     domain.EventTypeContentBlockStop,
						Index:    event.Index,
						ToolCall: tc,
					}
					delete(activeToolCalls, event.Index)
				}

			case "message_delta":
				event, err := result.ParseMessageDelta()
				if err != nil {
					out <- domain.CanonicalEvent{Error: fmt.Errorf("parse message_delta: %w", err)}
					return
				}
				if event.Usage != nil {
					outputTokens = event.Usage.OutputTokens
				}
				// Capture stop reason
				if event.Delta.StopReason != "" {
					stopReason = event.Delta.StopReason
				}

			case "message_stop":
				// Map Anthropic stop_reason to OpenAI finish_reason
				finishReason := MapStopReason(stopReason)

				// Send final event with usage and finish reason
				out <- domain.CanonicalEvent{
					Usage: &domain.Usage{
						PromptTokens:     inputTokens,
						CompletionTokens: outputTokens,
						TotalTokens:      inputTokens + outputTokens,
					},
					FinishReason: finishReason,
				}
				return

			case "ping":
				// Ignore ping events
				continue
			}
		}
	}()

	return out, nil
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	// ListModels doesn't need user agent passthrough as it's typically internal
	resp, err := p.client.ListModels(ctx, nil)
	if err != nil {
		return nil, err
	}

	models := make([]domain.Model, len(resp.Data))
	for i, m := range resp.Data {
		created := int64(0)
		if m.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
				created = t.Unix()
			}
		}
		models[i] = domain.Model{
			ID:      m.ID,
			Object:  m.Type,
			Created: created,
		}
	}

	return &domain.ModelList{
		Object: "list",
		Data:   models,
	}, nil
}

// ToCanonicalRequest converts an Anthropic API request to a canonical request.
// Exposed for use by frontdoor handlers via the codec.
func ToCanonicalRequest(apiReq *MessagesRequest) (*domain.CanonicalRequest, error) {
	return APIRequestToCanonical(apiReq)
}

// CountTokens passes through a count_tokens request to the Anthropic API.
// This method accepts raw JSON bytes and returns raw JSON bytes.
func (p *Provider) CountTokens(ctx context.Context, body []byte) ([]byte, error) {
	var req CountTokensRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal count_tokens request: %w", err)
	}

	resp, err := p.client.CountTokens(ctx, &req, nil)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}
