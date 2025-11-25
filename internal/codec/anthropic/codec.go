// Package anthropic provides a codec for converting between Anthropic API format and canonical format.
package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/codec"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// Codec implements codec.Codec for Anthropic API format.
type Codec struct{}

// New creates a new Anthropic codec.
func New() *Codec {
	return &Codec{}
}

// Name returns the codec name.
func (c *Codec) Name() string {
	return "anthropic"
}

// DecodeRequest converts Anthropic API request JSON to canonical format.
func (c *Codec) DecodeRequest(data []byte) (*domain.CanonicalRequest, error) {
	var apiReq anthropic.MessagesRequest
	if err := json.Unmarshal(data, &apiReq); err != nil {
		return nil, fmt.Errorf("failed to decode Anthropic request: %w", err)
	}
	return APIRequestToCanonical(&apiReq)
}

// EncodeRequest converts canonical request to Anthropic API request JSON.
func (c *Codec) EncodeRequest(req *domain.CanonicalRequest) ([]byte, error) {
	apiReq := CanonicalToAPIRequest(req)
	return json.Marshal(apiReq)
}

// DecodeResponse converts Anthropic API response JSON to canonical format.
func (c *Codec) DecodeResponse(data []byte) (*domain.CanonicalResponse, error) {
	var apiResp anthropic.MessagesResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Anthropic response: %w", err)
	}
	return APIResponseToCanonical(&apiResp), nil
}

// EncodeResponse converts canonical response to Anthropic API response JSON.
func (c *Codec) EncodeResponse(resp *domain.CanonicalResponse) ([]byte, error) {
	apiResp := CanonicalToAPIResponse(resp)
	return json.Marshal(apiResp)
}

// DecodeStreamChunk converts an Anthropic SSE data payload to a canonical event.
// Note: Anthropic streaming has multiple event types, so the caller should handle
// event type routing. This decodes content_block_delta events.
func (c *Codec) DecodeStreamChunk(data []byte) (*domain.CanonicalEvent, error) {
	// First determine the event type
	var eventType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &eventType); err != nil {
		return nil, fmt.Errorf("failed to decode event type: %w", err)
	}

	switch eventType.Type {
	case "content_block_delta":
		var delta anthropic.ContentBlockDeltaEvent
		if err := json.Unmarshal(data, &delta); err != nil {
			return nil, fmt.Errorf("failed to decode content_block_delta: %w", err)
		}
		return &domain.CanonicalEvent{
			ContentDelta: delta.Delta.Text,
		}, nil

	case "message_start":
		var start anthropic.MessageStartEvent
		if err := json.Unmarshal(data, &start); err != nil {
			return nil, fmt.Errorf("failed to decode message_start: %w", err)
		}
		return &domain.CanonicalEvent{
			Role: start.Message.Role,
			Usage: &domain.Usage{
				PromptTokens: start.Message.Usage.InputTokens,
			},
		}, nil

	case "message_delta":
		var delta anthropic.MessageDeltaEvent
		if err := json.Unmarshal(data, &delta); err != nil {
			return nil, fmt.Errorf("failed to decode message_delta: %w", err)
		}
		event := &domain.CanonicalEvent{}
		if delta.Usage != nil {
			event.Usage = &domain.Usage{
				CompletionTokens: delta.Usage.OutputTokens,
			}
		}
		return event, nil

	case "message_stop", "content_block_start", "content_block_stop", "ping":
		// These events don't carry content we need to forward
		return &domain.CanonicalEvent{}, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType.Type)
	}
}

// EncodeStreamChunk converts a canonical event to Anthropic SSE chunk JSON.
func (c *Codec) EncodeStreamChunk(event *domain.CanonicalEvent, metadata *codec.StreamMetadata) ([]byte, error) {
	chunk := CanonicalToAPIChunk(event)
	return json.Marshal(chunk)
}

// APIRequestToCanonical converts an Anthropic API request to canonical format.
func APIRequestToCanonical(apiReq *anthropic.MessagesRequest) (*domain.CanonicalRequest, error) {
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
func collapseContentBlocks(blocks anthropic.ContentBlock) (string, error) {
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

// CanonicalToAPIRequest converts a canonical request to Anthropic API format.
func CanonicalToAPIRequest(req *domain.CanonicalRequest) *anthropic.MessagesRequest {
	var systemBlocks anthropic.SystemMessages
	var messages []anthropic.Message

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			systemBlocks = append(systemBlocks, anthropic.SystemBlock{
				Type: "text",
				Text: m.Content,
			})
		case "user", "assistant":
			messages = append(messages, anthropic.Message{
				Role:    m.Role,
				Content: anthropic.ContentBlock{{Type: "text", Text: m.Content}},
			})
		}
	}

	apiReq := &anthropic.MessagesRequest{
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

	// Convert tools
	if len(req.Tools) > 0 {
		apiReq.Tools = make([]anthropic.Tool, len(req.Tools))
		for i, t := range req.Tools {
			apiReq.Tools[i] = anthropic.Tool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			}
		}
	}

	return apiReq
}

// APIResponseToCanonical converts an Anthropic API response to canonical format.
func APIResponseToCanonical(apiResp *anthropic.MessagesResponse) *domain.CanonicalResponse {
	content := ""
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &domain.CanonicalResponse{
		ID:      apiResp.ID,
		Object:  "chat.completion", // Map to OpenAI-compatible object type
		Created: 0,                 // Anthropic doesn't return created timestamp
		Model:   apiResp.Model,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    apiResp.Role,
					Content: content,
				},
				FinishReason: apiResp.StopReason,
			},
		},
		Usage: domain.Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}
}

// CanonicalToAPIResponse converts a canonical response to Anthropic API format.
func CanonicalToAPIResponse(resp *domain.CanonicalResponse) *anthropic.MessagesResponse {
	content := ""
	finishReason := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		finishReason = resp.Choices[0].FinishReason
	}

	return &anthropic.MessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      resp.Model,
		StopReason: finishReason,
		Content: []anthropic.ResponseContent{
			{
				Type: "text",
				Text: content,
			},
		},
		Usage: anthropic.MessagesUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}

// CanonicalToAPIChunk converts a canonical event to an Anthropic streaming chunk.
func CanonicalToAPIChunk(event *domain.CanonicalEvent) *anthropic.ContentBlockDeltaEvent {
	return &anthropic.ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: 0,
		Delta: anthropic.BlockDelta{
			Type: "text_delta",
			Text: event.ContentDelta,
		},
	}
}

// Ensure Codec implements the interface
var _ codec.Codec = (*Codec)(nil)
