// Package anthropic provides a codec for converting between Anthropic API format and canonical format.
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
// For image URL handling with fetching, use CanonicalToAPIRequestWithImageFetching.
func CanonicalToAPIRequest(req *domain.CanonicalRequest) *anthropic.MessagesRequest {
	return canonicalToAPIRequestInternal(req, nil, nil)
}

// CanonicalToAPIRequestWithImageFetching converts a canonical request to Anthropic API format,
// fetching any image URLs and converting them to base64 for Anthropic compatibility.
func CanonicalToAPIRequestWithImageFetching(ctx context.Context, req *domain.CanonicalRequest, fetcher *codec.ImageFetcher) (*anthropic.MessagesRequest, error) {
	var fetchErr error
	result := canonicalToAPIRequestInternal(req, fetcher, &fetchErr)
	if fetchErr != nil {
		return nil, fetchErr
	}
	return result, nil
}

// canonicalToAPIRequestInternal is the internal implementation that optionally fetches images.
func canonicalToAPIRequestInternal(req *domain.CanonicalRequest, fetcher *codec.ImageFetcher, fetchErr *error) *anthropic.MessagesRequest {
	var systemBlocks anthropic.SystemMessages
	var messages []anthropic.Message

	// Handle system prompt if set separately
	if req.SystemPrompt != "" {
		systemBlocks = append(systemBlocks, anthropic.SystemBlock{
			Type: "text",
			Text: req.SystemPrompt,
		})
	}

	// Handle instructions (Responses API) as system message
	if req.Instructions != "" && req.SystemPrompt == "" {
		systemBlocks = append(systemBlocks, anthropic.SystemBlock{
			Type: "text",
			Text: req.Instructions,
		})
	}

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			systemBlocks = append(systemBlocks, anthropic.SystemBlock{
				Type: "text",
				Text: m.Content,
			})
		case "user":
			// Check if this is a tool result message
			if m.ToolCallID != "" {
				messages = append(messages, anthropic.Message{
					Role: "user",
					Content: anthropic.ContentBlock{{
						Type:      "tool_result",
						ToolUseID: m.ToolCallID,
						Content:   m.Content,
					}},
				})
			} else if m.HasRichContent() {
				// Handle multimodal content (images, etc.)
				content := convertRichContentToAnthropic(m.RichContent, fetcher, fetchErr)
				messages = append(messages, anthropic.Message{
					Role:    m.Role,
					Content: content,
				})
			} else {
				messages = append(messages, anthropic.Message{
					Role:    m.Role,
					Content: anthropic.ContentBlock{{Type: "text", Text: m.Content}},
				})
			}
		case "assistant":
			// Check if assistant message has tool calls
			if len(m.ToolCalls) > 0 {
				var content anthropic.ContentBlock
				if m.Content != "" {
					content = append(content, anthropic.ContentPart{Type: "text", Text: m.Content})
				}
				for _, tc := range m.ToolCalls {
					content = append(content, anthropic.ContentPart{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Function.Name,
						Input: tc.Function.Arguments, // Note: Anthropic expects parsed JSON, not string
					})
				}
				messages = append(messages, anthropic.Message{
					Role:    m.Role,
					Content: content,
				})
			} else {
				messages = append(messages, anthropic.Message{
					Role:    m.Role,
					Content: anthropic.ContentBlock{{Type: "text", Text: m.Content}},
				})
			}
		case "tool":
			// OpenAI tool role maps to user with tool_result in Anthropic
			messages = append(messages, anthropic.Message{
				Role: "user",
				Content: anthropic.ContentBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
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
		apiReq.MaxTokens = 4096 // Default - increased from 1024
	}

	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}

	if req.TopP > 0 {
		apiReq.TopP = &req.TopP
	}

	// Convert stop sequences
	if len(req.Stop) > 0 {
		apiReq.StopSequences = req.Stop
	}

	// Convert tool choice
	if req.ToolChoice != nil {
		switch tc := req.ToolChoice.(type) {
		case string:
			switch tc {
			case "auto":
				apiReq.ToolChoice = &anthropic.ToolChoice{Type: "auto"}
			case "none":
				// Anthropic doesn't have "none", just don't send tools
			case "required":
				apiReq.ToolChoice = &anthropic.ToolChoice{Type: "any"}
			}
		case map[string]interface{}:
			if name, ok := tc["function"].(map[string]interface{})["name"].(string); ok {
				apiReq.ToolChoice = &anthropic.ToolChoice{Type: "tool", Name: name}
			}
		}
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
	return APIResponseToCanonicalWithRateLimits(apiResp, nil)
}

// APIResponseToCanonicalWithRateLimits converts an Anthropic API response to canonical format,
// including rate limit information from the upstream response.
func APIResponseToCanonicalWithRateLimits(apiResp *anthropic.MessagesResponse, rateLimits *anthropic.RateLimitHeaders) *domain.CanonicalResponse {
	content := ""
	var toolCalls []domain.ToolCall

	for _, c := range apiResp.Content {
		switch c.Type {
		case "text":
			content += c.Text
		case "tool_use":
			// Convert Anthropic tool_use to OpenAI-style tool call
			tc := domain.ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: domain.ToolCallFunction{
					Name: c.Name,
				},
			}
			// Convert input to JSON string (Anthropic sends parsed object, OpenAI expects string)
			if c.Input != nil {
				if inputBytes, err := json.Marshal(c.Input); err == nil {
					tc.Function.Arguments = string(inputBytes)
				}
			}
			toolCalls = append(toolCalls, tc)
		}
	}

	// Map Anthropic stop_reason to OpenAI finish_reason
	finishReason := mapAnthropicStopReason(apiResp.StopReason)

	msg := domain.Message{
		Role:      apiResp.Role,
		Content:   content,
		ToolCalls: toolCalls,
	}

	resp := &domain.CanonicalResponse{
		ID:      apiResp.ID,
		Object:  "chat.completion", // Map to OpenAI-compatible object type
		Created: 0,                 // Anthropic doesn't return created timestamp
		Model:   apiResp.Model,
		Choices: []domain.Choice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: finishReason,
			},
		},
		Usage: domain.Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
		SourceAPIType: domain.APITypeAnthropic,
	}

	// Add rate limit info if present
	if rateLimits != nil {
		resp.RateLimits = &domain.RateLimitInfo{
			RequestsLimit:     rateLimits.RequestsLimit,
			RequestsRemaining: rateLimits.RequestsRemaining,
			RequestsReset:     rateLimits.RequestsReset,
			TokensLimit:       rateLimits.TokensLimit,
			TokensRemaining:   rateLimits.TokensRemaining,
			TokensReset:       rateLimits.TokensReset,
		}
	}

	return resp
}

// mapAnthropicStopReason maps Anthropic stop_reason to OpenAI finish_reason
func mapAnthropicStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	default:
		return stopReason
	}
}

// mapOpenAIFinishReason maps OpenAI finish_reason to Anthropic stop_reason
func mapOpenAIFinishReason(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "end_turn" // No direct equivalent
	default:
		return finishReason
	}
}

// CanonicalToAPIResponse converts a canonical response to Anthropic API format.
func CanonicalToAPIResponse(resp *domain.CanonicalResponse) *anthropic.MessagesResponse {
	var contentBlocks []anthropic.ResponseContent
	finishReason := ""

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		finishReason = mapOpenAIFinishReason(choice.FinishReason)

		// Add text content if present
		if choice.Message.Content != "" {
			contentBlocks = append(contentBlocks, anthropic.ResponseContent{
				Type: "text",
				Text: choice.Message.Content,
			})
		}

		// Convert tool calls to Anthropic tool_use blocks
		for _, tc := range choice.Message.ToolCalls {
			var input any
			// Parse arguments JSON string back to object
			if tc.Function.Arguments != "" {
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
			}
			contentBlocks = append(contentBlocks, anthropic.ResponseContent{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}
	}

	// Ensure at least one content block
	if len(contentBlocks) == 0 {
		contentBlocks = []anthropic.ResponseContent{{Type: "text", Text: ""}}
	}

	return &anthropic.MessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      resp.Model,
		StopReason: finishReason,
		Content:    contentBlocks,
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

// convertRichContentToAnthropic converts domain.MessageContent to Anthropic content blocks.
// If a fetcher is provided, image URLs will be fetched and converted to base64.
func convertRichContentToAnthropic(content *domain.MessageContent, fetcher *codec.ImageFetcher, fetchErr *error) anthropic.ContentBlock {
	if content == nil || content.IsSimpleText() {
		text := ""
		if content != nil {
			text = content.Text
		}
		return anthropic.ContentBlock{{Type: "text", Text: text}}
	}

	var result anthropic.ContentBlock
	for _, part := range content.Parts {
		switch part.Type {
		case domain.ContentTypeText:
			result = append(result, anthropic.ContentPart{
				Type: "text",
				Text: part.Text,
			})

		case domain.ContentTypeImage:
			// Already base64 encoded image
			if part.Source != nil {
				result = append(result, anthropic.ContentPart{
					Type: "image",
					Source: &anthropic.ImageSource{
						Type:      part.Source.Type,
						MediaType: part.Source.MediaType,
						Data:      part.Source.Data,
					},
				})
			}

		case domain.ContentTypeImageURL:
			// Image URL that needs to be fetched and converted
			if part.ImageURL != nil {
				if fetcher != nil {
					// Fetch and convert the image
					source, err := fetcher.FetchAndConvert(context.Background(), part.ImageURL.URL)
					if err != nil {
						if fetchErr != nil {
							*fetchErr = fmt.Errorf("failed to fetch image from %s: %w", part.ImageURL.URL, err)
						}
						// Still add a placeholder so the message structure is preserved
						continue
					}
					result = append(result, anthropic.ContentPart{
						Type: "image",
						Source: &anthropic.ImageSource{
							Type:      source.Type,
							MediaType: source.MediaType,
							Data:      source.Data,
						},
					})
				} else {
					// Check if it's a data URL (already base64)
					if strings.HasPrefix(part.ImageURL.URL, "data:") {
						source, err := parseDataURL(part.ImageURL.URL)
						if err == nil {
							result = append(result, anthropic.ContentPart{
								Type: "image",
								Source: &anthropic.ImageSource{
									Type:      source.Type,
									MediaType: source.MediaType,
									Data:      source.Data,
								},
							})
						}
					}
					// If not a data URL and no fetcher, we can't handle it
					// The image will be skipped
				}
			}

		case domain.ContentTypeToolUse:
			result = append(result, anthropic.ContentPart{
				Type:  "tool_use",
				ID:    part.ID,
				Name:  part.Name,
				Input: part.Input,
			})

		case domain.ContentTypeToolResult:
			result = append(result, anthropic.ContentPart{
				Type:      "tool_result",
				ToolUseID: part.ToolUseID,
				Content:   part.Content,
				IsError:   part.IsError,
			})
		}
	}

	return result
}

// parseDataURL parses a data URL and extracts the base64 content.
func parseDataURL(url string) (*domain.ImageSource, error) {
	if !strings.HasPrefix(url, "data:") {
		return nil, fmt.Errorf("not a data URL")
	}

	// Remove "data:" prefix
	content := url[5:]

	// Find the comma separator
	commaIdx := strings.Index(content, ",")
	if commaIdx == -1 {
		return nil, fmt.Errorf("invalid data URL: missing comma separator")
	}

	metadata := content[:commaIdx]
	data := content[commaIdx+1:]

	// Parse metadata
	parts := strings.Split(metadata, ";")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid data URL: missing media type")
	}

	mediaType := parts[0]

	// Normalize image/jpg to image/jpeg
	if mediaType == "image/jpg" {
		mediaType = "image/jpeg"
	}

	return &domain.ImageSource{
		Type:      "base64",
		MediaType: mediaType,
		Data:      data,
	}, nil
}

// Ensure Codec implements the interface
var _ codec.Codec = (*Codec)(nil)
