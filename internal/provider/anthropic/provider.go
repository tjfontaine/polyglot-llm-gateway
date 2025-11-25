package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	anthropicapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	anthropiccodec "github.com/tjfontaine/polyglot-llm-gateway/internal/codec/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ProviderOption configures the provider.
type ProviderOption func(*Provider)

// WithBaseURL sets a custom base URL for the API.
func WithBaseURL(baseURL string) ProviderOption {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ProviderOption {
	return func(p *Provider) {
		p.httpClient = httpClient
	}
}

// Provider implements the domain.Provider interface using our custom Anthropic client.
type Provider struct {
	client     *anthropicapi.Client
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// New creates a new Anthropic provider.
func New(apiKey string, opts ...ProviderOption) *Provider {
	p := &Provider{
		apiKey: apiKey,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Build client options
	var clientOpts []anthropicapi.ClientOption
	if p.baseURL != "" {
		clientOpts = append(clientOpts, anthropicapi.WithBaseURL(p.baseURL))
	}
	if p.httpClient != nil {
		clientOpts = append(clientOpts, anthropicapi.WithHTTPClient(p.httpClient))
	}

	p.client = anthropicapi.NewClient(apiKey, clientOpts...)
	return p
}

func (p *Provider) Name() string {
	return "anthropic"
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	// Use codec to convert canonical request to API request
	apiReq := anthropiccodec.CanonicalToAPIRequest(req)

	opts := &anthropicapi.RequestOptions{
		UserAgent: req.UserAgent,
	}

	resp, err := p.client.CreateMessage(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	// Use codec to convert API response to canonical response
	return anthropiccodec.APIResponseToCanonical(resp), nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	// Use codec to convert canonical request to API request
	apiReq := anthropiccodec.CanonicalToAPIRequest(req)

	opts := &anthropicapi.RequestOptions{
		UserAgent: req.UserAgent,
	}

	stream, err := p.client.StreamMessage(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)

		var inputTokens, outputTokens int

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

			case "content_block_delta":
				event, err := result.ParseContentBlockDelta()
				if err != nil {
					out <- domain.CanonicalEvent{Error: fmt.Errorf("parse content_block_delta: %w", err)}
					return
				}
				if event.Delta.Type == "text_delta" {
					out <- domain.CanonicalEvent{ContentDelta: event.Delta.Text}
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

			case "message_stop":
				// Send final usage
				if inputTokens > 0 || outputTokens > 0 {
					out <- domain.CanonicalEvent{
						Usage: &domain.Usage{
							PromptTokens:     inputTokens,
							CompletionTokens: outputTokens,
							TotalTokens:      inputTokens + outputTokens,
						},
					}
				}
				return

			case "ping", "content_block_start", "content_block_stop":
				// Ignore these events
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
func ToCanonicalRequest(apiReq *anthropicapi.MessagesRequest) (*domain.CanonicalRequest, error) {
	return anthropiccodec.APIRequestToCanonical(apiReq)
}
