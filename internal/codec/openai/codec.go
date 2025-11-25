// Package openai provides a codec for converting between OpenAI API format and canonical format.
package openai

import (
	"encoding/json"
	"fmt"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/openai"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/codec"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// Codec implements codec.Codec for OpenAI API format.
type Codec struct{}

// New creates a new OpenAI codec.
func New() *Codec {
	return &Codec{}
}

// Name returns the codec name.
func (c *Codec) Name() string {
	return "openai"
}

// DecodeRequest converts OpenAI API request JSON to canonical format.
func (c *Codec) DecodeRequest(data []byte) (*domain.CanonicalRequest, error) {
	var apiReq openai.ChatCompletionRequest
	if err := json.Unmarshal(data, &apiReq); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI request: %w", err)
	}
	return APIRequestToCanonical(&apiReq), nil
}

// EncodeRequest converts canonical request to OpenAI API request JSON.
func (c *Codec) EncodeRequest(req *domain.CanonicalRequest) ([]byte, error) {
	apiReq := CanonicalToAPIRequest(req)
	return json.Marshal(apiReq)
}

// DecodeResponse converts OpenAI API response JSON to canonical format.
func (c *Codec) DecodeResponse(data []byte) (*domain.CanonicalResponse, error) {
	var apiResp openai.ChatCompletionResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}
	return APIResponseToCanonical(&apiResp), nil
}

// EncodeResponse converts canonical response to OpenAI API response JSON.
func (c *Codec) EncodeResponse(resp *domain.CanonicalResponse) ([]byte, error) {
	apiResp := CanonicalToAPIResponse(resp)
	return json.Marshal(apiResp)
}

// DecodeStreamChunk converts an OpenAI SSE data payload to a canonical event.
func (c *Codec) DecodeStreamChunk(data []byte) (*domain.CanonicalEvent, error) {
	var chunk openai.ChatCompletionChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI chunk: %w", err)
	}
	return APIChunkToCanonical(&chunk), nil
}

// EncodeStreamChunk converts a canonical event to OpenAI SSE chunk JSON.
func (c *Codec) EncodeStreamChunk(event *domain.CanonicalEvent, metadata *codec.StreamMetadata) ([]byte, error) {
	chunk := CanonicalToAPIChunk(event, metadata)
	return json.Marshal(chunk)
}

// APIRequestToCanonical converts an OpenAI API request to canonical format.
func APIRequestToCanonical(apiReq *openai.ChatCompletionRequest) *domain.CanonicalRequest {
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

// CanonicalToAPIRequest converts a canonical request to OpenAI API format.
func CanonicalToAPIRequest(req *domain.CanonicalRequest) *openai.ChatCompletionRequest {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		}
	}

	apiReq := &openai.ChatCompletionRequest{
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

	// Convert tools
	if len(req.Tools) > 0 {
		apiReq.Tools = make([]openai.Tool, len(req.Tools))
		for i, t := range req.Tools {
			apiReq.Tools[i] = openai.Tool{
				Type: t.Type,
				Function: openai.FunctionTool{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	return apiReq
}

// APIResponseToCanonical converts an OpenAI API response to canonical format.
func APIResponseToCanonical(apiResp *openai.ChatCompletionResponse) *domain.CanonicalResponse {
	choices := make([]domain.Choice, len(apiResp.Choices))
	for i, c := range apiResp.Choices {
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
		ID:      apiResp.ID,
		Object:  apiResp.Object,
		Created: apiResp.Created,
		Model:   apiResp.Model,
		Choices: choices,
		Usage: domain.Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
	}
}

// CanonicalToAPIResponse converts a canonical response to OpenAI API format.
func CanonicalToAPIResponse(resp *domain.CanonicalResponse) *openai.ChatCompletionResponse {
	choices := make([]openai.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = openai.Choice{
			Index: c.Index,
			Message: openai.ChatCompletionMessage{
				Role:    c.Message.Role,
				Content: c.Message.Content,
				Name:    c.Message.Name,
			},
			FinishReason: c.FinishReason,
		}
	}

	return &openai.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: openai.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

// APIChunkToCanonical converts an OpenAI streaming chunk to a canonical event.
func APIChunkToCanonical(chunk *openai.ChatCompletionChunk) *domain.CanonicalEvent {
	event := &domain.CanonicalEvent{}

	if len(chunk.Choices) > 0 {
		choice := chunk.Choices[0]
		event.Role = choice.Delta.Role
		event.ContentDelta = choice.Delta.Content

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
	}

	// Handle usage in final chunk
	if chunk.Usage != nil {
		event.Usage = &domain.Usage{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		}
	}

	return event
}

// CanonicalToAPIChunk converts a canonical event to an OpenAI streaming chunk.
func CanonicalToAPIChunk(event *domain.CanonicalEvent, metadata *codec.StreamMetadata) *openai.ChatCompletionChunk {
	chunk := &openai.ChatCompletionChunk{
		Object: "chat.completion.chunk",
		Choices: []openai.ChunkChoice{
			{
				Index: 0,
				Delta: openai.ChunkDelta{
					Role:    event.Role,
					Content: event.ContentDelta,
				},
			},
		},
	}

	if metadata != nil {
		chunk.ID = metadata.ID
		chunk.Model = metadata.Model
		chunk.Created = metadata.Created
	}

	// Handle tool calls
	if event.ToolCall != nil {
		chunk.Choices[0].Delta.ToolCalls = []openai.ToolCallChunk{
			{
				Index: event.ToolCall.Index,
				ID:    event.ToolCall.ID,
				Type:  event.ToolCall.Type,
				Function: &openai.FunctionCallChunk{
					Name:      event.ToolCall.Function.Name,
					Arguments: event.ToolCall.Function.Arguments,
				},
			},
		}
	}

	// Handle usage
	if event.Usage != nil {
		chunk.Usage = &openai.Usage{
			PromptTokens:     event.Usage.PromptTokens,
			CompletionTokens: event.Usage.CompletionTokens,
			TotalTokens:      event.Usage.TotalTokens,
		}
	}

	return chunk
}

// Ensure Codec implements the interface
var _ codec.Codec = (*Codec)(nil)
