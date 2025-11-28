// Provider implements the domain.Provider interface for the OpenAI API.
package openai

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
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

// WithResponsesAPI enables using the Responses API instead of Chat Completions.
func WithResponsesAPI(enable bool) ProviderOption {
	return func(p *Provider) {
		p.useResponsesAPI = enable
	}
}

// Provider implements the domain.Provider interface using our custom OpenAI client.
type Provider struct {
	client          *Client
	apiKey          string
	baseURL         string
	httpClient      *http.Client
	useResponsesAPI bool
}

// NewProvider creates a new OpenAI provider.
func NewProvider(apiKey string, opts ...ProviderOption) *Provider {
	p := &Provider{
		apiKey: apiKey,
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

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) APIType() domain.APIType {
	return domain.APITypeOpenAI
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	opts := &RequestOptions{
		UserAgent: req.UserAgent,
	}

	if p.useResponsesAPI {
		return p.completeWithResponses(ctx, req, opts)
	}

	// Use codec to convert canonical request to API request
	apiReq := CanonicalToAPIRequest(req)

	resp, err := p.client.CreateChatCompletion(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	// Use codec to convert API response to canonical response
	return APIResponseToCanonical(resp), nil
}

func (p *Provider) completeWithResponses(ctx context.Context, req *domain.CanonicalRequest, opts *RequestOptions) (*domain.CanonicalResponse, error) {
	apiReq := canonicalToResponsesRequest(req)

	resp, err := p.client.CreateResponse(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	return responsesResponseToCanonical(resp), nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	opts := &RequestOptions{
		UserAgent: req.UserAgent,
	}

	if p.useResponsesAPI {
		return p.streamWithResponses(ctx, req, opts)
	}

	// Use codec to convert canonical request to API request
	apiReq := CanonicalToAPIRequest(req)

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
			event := APIChunkToCanonical(result.Chunk)
			out <- *event
		}
	}()

	return out, nil
}

func (p *Provider) streamWithResponses(ctx context.Context, req *domain.CanonicalRequest, opts *RequestOptions) (<-chan domain.CanonicalEvent, error) {
	apiReq := canonicalToResponsesRequest(req)
	apiReq.Stream = true

	stream, err := p.client.StreamResponse(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)
		var currentModel string
		var currentResponseID string

		for result := range stream {
			if result.Err != nil {
				out <- domain.CanonicalEvent{Error: result.Err}
				return
			}

			event := result.Event
			if event == nil {
				continue
			}

			// Parse different event types (per OpenAI Responses API Spec v1.1)
			switch event.Type {
			case "response.created":
				// Per spec: {"id": "resp_123", "created": 1732000000, "model": "gpt-5"}
				var data struct {
					ID    string `json:"id"`
					Model string `json:"model"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					currentModel = data.Model
					currentResponseID = data.ID
				}

			case "response.output_item.delta":
				// Per spec: {"item_index": 0, "delta": {"content": "Hello"}} or {"delta": {"arguments": "..."}}
				var data struct {
					ItemIndex int `json:"item_index"`
					Delta     struct {
						Content   string `json:"content"`
						Arguments string `json:"arguments"`
					} `json:"delta"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					if data.Delta.Content != "" {
						out <- domain.CanonicalEvent{
							ContentDelta: data.Delta.Content,
							Model:        currentModel,
							ResponseID:   currentResponseID,
						}
					}
					// TODO: Handle arguments delta for tool calls if needed
				}

			case "response.done":
				// Per spec: {"usage": {...}, "finish_reason": "stop"}
				var data struct {
					Usage *struct {
						InputTokens  int `json:"input_tokens"`
						OutputTokens int `json:"output_tokens"`
						TotalTokens  int `json:"total_tokens"`
					} `json:"usage"`
					FinishReason string `json:"finish_reason"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					ev := domain.CanonicalEvent{
						Model:        currentModel,
						ResponseID:   currentResponseID,
						FinishReason: data.FinishReason,
					}
					if data.Usage != nil {
						ev.Usage = &domain.Usage{
							PromptTokens:     data.Usage.InputTokens,
							CompletionTokens: data.Usage.OutputTokens,
							TotalTokens:      data.Usage.TotalTokens,
						}
					}
					out <- ev
				}

			// Legacy event handling for backwards compatibility
			case "response.in_progress":
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					currentModel = data.Response.Model
					currentResponseID = data.Response.ID
				}

			case "response.output_text.delta":
				var data struct {
					Delta string `json:"delta"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil && data.Delta != "" {
					out <- domain.CanonicalEvent{
						ContentDelta: data.Delta,
						Model:        currentModel,
						ResponseID:   currentResponseID,
					}
				}

			case "response.completed":
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					if data.Response.Usage != nil {
						out <- domain.CanonicalEvent{
							Model:      data.Response.Model,
							ResponseID: data.Response.ID,
							Usage: &domain.Usage{
								PromptTokens:     data.Response.Usage.InputTokens,
								CompletionTokens: data.Response.Usage.OutputTokens,
								TotalTokens:      data.Response.Usage.TotalTokens,
							},
						}
					}
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

// canonicalToResponsesRequest converts a canonical request to OpenAI Responses API format.
func canonicalToResponsesRequest(req *domain.CanonicalRequest) *ResponsesRequest {
	// Build input items from messages
	var input any

	// If there's a single user message and no system prompt, use simple text input
	if len(req.Messages) == 1 && req.Messages[0].Role == "user" && req.SystemPrompt == "" && req.Instructions == "" {
		input = req.Messages[0].Content
	} else {
		// Build input items from messages
		items := make([]ResponsesInputItem, 0, len(req.Messages))
		for _, msg := range req.Messages {
			item := ResponsesInputItem{
				Type: "message",
				Role: msg.Role,
				Content: []ResponsesContentPart{
					{Type: "input_text", Text: msg.Content},
				},
			}
			items = append(items, item)
		}
		input = items
	}

	apiReq := &ResponsesRequest{
		Model: req.Model,
		Input: input,
	}

	// Set instructions from system prompt
	if req.Instructions != "" {
		apiReq.Instructions = req.Instructions
	} else if req.SystemPrompt != "" {
		apiReq.Instructions = req.SystemPrompt
	}

	if req.MaxTokens > 0 {
		apiReq.MaxOutputTokens = req.MaxTokens
	}

	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}

	if req.TopP > 0 {
		apiReq.TopP = &req.TopP
	}

	apiReq.ToolChoice = req.ToolChoice

	// Convert tools
	if len(req.Tools) > 0 {
		apiReq.Tools = make([]ResponsesTool, len(req.Tools))
		for i, t := range req.Tools {
			apiReq.Tools[i] = ResponsesTool{
				Type: t.Type,
				Function: FunctionTool{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	return apiReq
}

// responsesResponseToCanonical converts an OpenAI Responses API response to canonical format.
func responsesResponseToCanonical(resp *ResponsesResponse) *domain.CanonicalResponse {
	choices := make([]domain.Choice, 0)

	for _, item := range resp.Output {
		if item.Type == "message" {
			var content string
			for _, part := range item.Content {
				if part.Type == "output_text" {
					content += part.Text
				}
			}

			choices = append(choices, domain.Choice{
				Index: len(choices),
				Message: domain.Message{
					Role:    item.Role,
					Content: content,
				},
				FinishReason: "stop",
			})
		} else if item.Type == "function_call" {
			// Handle function calls
			if len(choices) == 0 {
				choices = append(choices, domain.Choice{
					Index: 0,
					Message: domain.Message{
						Role: "assistant",
					},
					FinishReason: "tool_calls",
				})
			}

			choices[0].Message.ToolCalls = append(choices[0].Message.ToolCalls, domain.ToolCall{
				ID:   item.CallID,
				Type: "function",
				Function: domain.ToolCallFunction{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		}
	}

	// Ensure we have at least one choice
	if len(choices) == 0 {
		choices = append(choices, domain.Choice{
			Index:        0,
			Message:      domain.Message{Role: "assistant"},
			FinishReason: "stop",
		})
	}

	canonResp := &domain.CanonicalResponse{
		ID:            resp.ID,
		Object:        "chat.completion",
		Created:       resp.CreatedAt,
		Model:         resp.Model,
		Choices:       choices,
		SourceAPIType: domain.APITypeOpenAI,
	}

	if resp.Usage != nil {
		canonResp.Usage = domain.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return canonResp
}
