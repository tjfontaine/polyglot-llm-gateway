package openai

import (
	"context"
	"net/http"

	openaiapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/openai"
	openaicodec "github.com/tjfontaine/polyglot-llm-gateway/internal/codec/openai"
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

// Provider implements the domain.Provider interface using our custom OpenAI client.
type Provider struct {
	client     *openaiapi.Client
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// New creates a new OpenAI provider.
func New(apiKey string, opts ...ProviderOption) *Provider {
	p := &Provider{
		apiKey: apiKey,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Build client options
	var clientOpts []openaiapi.ClientOption
	if p.baseURL != "" {
		clientOpts = append(clientOpts, openaiapi.WithBaseURL(p.baseURL))
	}
	if p.httpClient != nil {
		clientOpts = append(clientOpts, openaiapi.WithHTTPClient(p.httpClient))
	}

	p.client = openaiapi.NewClient(apiKey, clientOpts...)
	return p
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) APIType() domain.APIType {
	return domain.APITypeOpenAI
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	// Use codec to convert canonical request to API request
	apiReq := openaicodec.CanonicalToAPIRequest(req)

	opts := &openaiapi.RequestOptions{
		UserAgent: req.UserAgent,
	}

	resp, err := p.client.CreateChatCompletion(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	// Use codec to convert API response to canonical response
	return openaicodec.APIResponseToCanonical(resp), nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	// Use codec to convert canonical request to API request
	apiReq := openaicodec.CanonicalToAPIRequest(req)

	opts := &openaiapi.RequestOptions{
		UserAgent: req.UserAgent,
	}

	stream, err := p.client.StreamChatCompletion(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)
		for result := range stream {
			if result.Err != nil {
				out <- domain.CanonicalEvent{Error: result.Err}
				return
			}

			// Use codec to convert API chunk to canonical event
			event := openaicodec.APIChunkToCanonical(result.Chunk)
			out <- *event
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
		models[i] = domain.Model{
			ID:      m.ID,
			Object:  m.Object,
			OwnedBy: m.OwnedBy,
			Created: m.Created,
		}
	}

	return &domain.ModelList{
		Object: resp.Object,
		Data:   models,
	}, nil
}
