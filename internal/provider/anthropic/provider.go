package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	anthropicapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
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
	apiReq := toAPIRequest(req)

	opts := &anthropicapi.RequestOptions{
		UserAgent: req.UserAgent,
	}

	resp, err := p.client.CreateMessage(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	return toCanonicalResponse(resp), nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	apiReq := toAPIRequest(req)

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

// toAPIRequest converts a canonical request to an Anthropic API request.
func toAPIRequest(req *domain.CanonicalRequest) *anthropicapi.MessagesRequest {
	var systemBlocks anthropicapi.SystemMessages
	var messages []anthropicapi.Message

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			systemBlocks = append(systemBlocks, anthropicapi.SystemBlock{
				Type: "text",
				Text: m.Content,
			})
		case "user", "assistant":
			messages = append(messages, anthropicapi.Message{
				Role:    m.Role,
				Content: anthropicapi.ContentBlock{{Type: "text", Text: m.Content}},
			})
		}
	}

	apiReq := &anthropicapi.MessagesRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if len(systemBlocks) > 0 {
		apiReq.System = systemBlocks
	}

	// Set max tokens (required for Anthropic)
	if req.MaxTokens > 0 {
		apiReq.MaxTokens = req.MaxTokens
	} else {
		apiReq.MaxTokens = 1024 // Default
	}

	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		apiReq.Tools = make([]anthropicapi.Tool, len(req.Tools))
		for i, t := range req.Tools {
			apiReq.Tools[i] = anthropicapi.Tool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			}
		}
	}

	return apiReq
}

// toCanonicalResponse converts an Anthropic API response to a canonical response.
func toCanonicalResponse(resp *anthropicapi.MessagesResponse) *domain.CanonicalResponse {
	content := ""
	if len(resp.Content) > 0 {
		for _, c := range resp.Content {
			if c.Type == "text" {
				content += c.Text
			}
		}
	}

	return &domain.CanonicalResponse{
		ID:      resp.ID,
		Object:  "chat.completion", // Map to OpenAI-compatible object type
		Created: 0,                 // Anthropic doesn't return created timestamp
		Model:   resp.Model,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    resp.Role,
					Content: content,
				},
				FinishReason: resp.StopReason,
			},
		},
		Usage: domain.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// ToAPIRequest is exported for use by frontdoor handlers that want to
// directly work with Anthropic API types.
func ToAPIRequest(req *domain.CanonicalRequest) *anthropicapi.MessagesRequest {
	return toAPIRequest(req)
}

// ToCanonicalResponse is exported for use by frontdoor handlers.
func ToCanonicalResponse(resp *anthropicapi.MessagesResponse) *domain.CanonicalResponse {
	return toCanonicalResponse(resp)
}

// ToCanonicalRequest converts an Anthropic API request to a canonical request.
// This is useful for frontdoor handlers that receive Anthropic format requests.
func ToCanonicalRequest(apiReq *anthropicapi.MessagesRequest) (*domain.CanonicalRequest, error) {
	messages := make([]domain.Message, 0, len(apiReq.Messages)+len(apiReq.System))

	// Add system messages first
	for _, sys := range apiReq.System {
		if sys.Type != "" && sys.Type != "text" {
			return nil, fmt.Errorf("unsupported system block type: %s", sys.Type)
		}
		messages = append(messages, domain.Message{
			Role:    "system",
			Content: sys.Text,
		})
	}

	// Add conversation messages
	for idx, msg := range apiReq.Messages {
		content, err := collapseContentBlocks(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("message %d: %w", idx, err)
		}
		messages = append(messages, domain.Message{
			Role:    msg.Role,
			Content: content,
		})
	}

	req := &domain.CanonicalRequest{
		Model:     apiReq.Model,
		Messages:  messages,
		Stream:    apiReq.Stream,
		MaxTokens: apiReq.MaxTokens,
	}

	if apiReq.Temperature != nil {
		req.Temperature = *apiReq.Temperature
	}

	// Convert tools
	if len(apiReq.Tools) > 0 {
		req.Tools = make([]domain.ToolDefinition, len(apiReq.Tools))
		for i, t := range apiReq.Tools {
			req.Tools[i] = domain.ToolDefinition{
				Type: "function",
				Function: domain.FunctionDef{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			}
		}
	}

	return req, nil
}

// collapseContentBlocks validates and collapses content blocks into a single string.
// Returns an error if unsupported content types are found.
func collapseContentBlocks(blocks anthropicapi.ContentBlock) (string, error) {
	if len(blocks) == 0 {
		return "", fmt.Errorf("content is required")
	}

	var result string
	for _, block := range blocks {
		blockType := block.Type
		if blockType == "" {
			blockType = "text"
		}
		if blockType != "text" {
			return "", fmt.Errorf("unsupported content block type: %s", blockType)
		}
		result += block.Text
	}

	return result, nil
}

// APIRequest is the Anthropic API request type, exported for frontdoor use.
type APIRequest = anthropicapi.MessagesRequest

// APIResponse is the Anthropic API response type, exported for frontdoor use.
type APIResponse = anthropicapi.MessagesResponse

// APIMessage is the Anthropic API message type, exported for frontdoor use.
type APIMessage = anthropicapi.Message

// APIContentBlock is the Anthropic API content block type, exported for frontdoor use.
type APIContentBlock = anthropicapi.ContentBlock

// APIContentPart is the Anthropic API content part type, exported for frontdoor use.
type APIContentPart = anthropicapi.ContentPart

// APISystemMessages is the Anthropic API system messages type, exported for frontdoor use.
type APISystemMessages = anthropicapi.SystemMessages

// APISystemBlock is the Anthropic API system block type, exported for frontdoor use.
type APISystemBlock = anthropicapi.SystemBlock

// FormatStreamingError formats an error for SSE streaming.
func FormatStreamingError(err error) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":"server_error","message":%q}}`, err.Error())
}
