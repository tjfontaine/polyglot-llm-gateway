package openai

import (
	"context"
	"fmt"
	"net/http"

	openaiapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/openai"
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

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	apiReq := toAPIRequest(req)

	opts := &openaiapi.RequestOptions{
		UserAgent: req.UserAgent,
	}

	resp, err := p.client.CreateChatCompletion(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	return toCanonicalResponse(resp), nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	apiReq := toAPIRequest(req)

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

			chunk := result.Chunk
			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				event := domain.CanonicalEvent{
					Role:         choice.Delta.Role,
					ContentDelta: choice.Delta.Content,
				}

				// Handle tool calls
				if len(choice.Delta.ToolCalls) > 0 {
					tc := choice.Delta.ToolCalls[0]
					event.ToolCall = &domain.ToolCallChunk{
						Index: tc.Index,
						ID:    tc.ID,
						Type:  tc.Type,
					}
					if tc.Function != nil {
						event.ToolCall.Function.Name = tc.Function.Name
						event.ToolCall.Function.Arguments = tc.Function.Arguments
					}
				}

				out <- event
			}

			// Handle usage in final chunk
			if chunk.Usage != nil {
				out <- domain.CanonicalEvent{
					Usage: &domain.Usage{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
					},
				}
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

// toAPIRequest converts a canonical request to an OpenAI API request.
func toAPIRequest(req *domain.CanonicalRequest) *openaiapi.ChatCompletionRequest {
	messages := make([]openaiapi.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiapi.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		}
	}

	apiReq := &openaiapi.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.MaxTokens > 0 {
		// Newer models prefer max_completion_tokens
		apiReq.MaxCompletionTokens = req.MaxTokens
	}

	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		apiReq.Tools = make([]openaiapi.Tool, len(req.Tools))
		for i, t := range req.Tools {
			apiReq.Tools[i] = openaiapi.Tool{
				Type: t.Type,
				Function: openaiapi.FunctionTool{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	return apiReq
}

// toCanonicalResponse converts an OpenAI API response to a canonical response.
func toCanonicalResponse(resp *openaiapi.ChatCompletionResponse) *domain.CanonicalResponse {
	choices := make([]domain.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = domain.Choice{
			Index: c.Index,
			Message: domain.Message{
				Role:    c.Message.Role,
				Content: c.Message.Content,
				Name:    c.Message.Name,
			},
			FinishReason: c.FinishReason,
		}
	}

	return &domain.CanonicalResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: domain.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

// ToAPIRequest is exported for use by frontdoor handlers that want to
// directly decode into OpenAI API types.
func ToAPIRequest(req *domain.CanonicalRequest) *openaiapi.ChatCompletionRequest {
	return toAPIRequest(req)
}

// ToCanonicalResponse is exported for use by frontdoor handlers.
func ToCanonicalResponse(resp *openaiapi.ChatCompletionResponse) *domain.CanonicalResponse {
	return toCanonicalResponse(resp)
}

// ToCanonicalRequest converts an OpenAI API request to a canonical request.
// This is useful for frontdoor handlers that receive OpenAI format requests.
func ToCanonicalRequest(apiReq *openaiapi.ChatCompletionRequest) *domain.CanonicalRequest {
	messages := make([]domain.Message, len(apiReq.Messages))
	for i, m := range apiReq.Messages {
		messages[i] = domain.Message{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		}
	}

	req := &domain.CanonicalRequest{
		Model:    apiReq.Model,
		Messages: messages,
		Stream:   apiReq.Stream,
	}

	// Prefer max_completion_tokens over max_tokens
	if apiReq.MaxCompletionTokens > 0 {
		req.MaxTokens = apiReq.MaxCompletionTokens
	} else if apiReq.MaxTokens > 0 {
		req.MaxTokens = apiReq.MaxTokens
	}

	if apiReq.Temperature != nil {
		req.Temperature = *apiReq.Temperature
	}

	// Convert tools
	if len(apiReq.Tools) > 0 {
		req.Tools = make([]domain.ToolDefinition, len(apiReq.Tools))
		for i, t := range apiReq.Tools {
			req.Tools[i] = domain.ToolDefinition{
				Type: t.Type,
				Function: domain.FunctionDef{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	return req
}

// APIRequest is the OpenAI API request type, exported for frontdoor use.
type APIRequest = openaiapi.ChatCompletionRequest

// APIResponse is the OpenAI API response type, exported for frontdoor use.
type APIResponse = openaiapi.ChatCompletionResponse

// APIChunk is the OpenAI API streaming chunk type, exported for frontdoor use.
type APIChunk = openaiapi.ChatCompletionChunk

// FormatStreamingError formats an error for SSE streaming.
func FormatStreamingError(err error) string {
	return fmt.Sprintf(`{"error":{"message":%q,"type":"server_error"}}`, err.Error())
}
