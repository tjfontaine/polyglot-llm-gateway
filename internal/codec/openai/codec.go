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
		msg := domain.Message{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}

		// Convert tool calls from OpenAI format
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]domain.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = domain.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: domain.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		messages[i] = msg
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

	if apiReq.TopP != nil {
		req.TopP = *apiReq.TopP
	}

	// Convert stop sequences
	if len(apiReq.Stop) > 0 {
		req.Stop = apiReq.Stop
	}

	// Convert tool choice
	req.ToolChoice = apiReq.ToolChoice

	// Convert response format
	if apiReq.ResponseFormat != nil {
		req.ResponseFormat = &domain.ResponseFormat{
			Type: apiReq.ResponseFormat.Type,
		}
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
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	// Handle system prompt if set separately
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Handle instructions (Responses API) as system message
	if req.Instructions != "" && req.SystemPrompt == "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "system",
			Content: req.Instructions,
		})
	}

	for _, m := range req.Messages {
		msg := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}

		// Convert tool calls to OpenAI format
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]openai.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		messages = append(messages, msg)
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

	if req.TopP > 0 {
		apiReq.TopP = &req.TopP
	}

	// Convert stop sequences
	if len(req.Stop) > 0 {
		apiReq.Stop = req.Stop
	}

	// Convert tool choice
	apiReq.ToolChoice = req.ToolChoice

	// Convert response format
	if req.ResponseFormat != nil {
		apiReq.ResponseFormat = &openai.ResponseFormat{
			Type: req.ResponseFormat.Type,
		}
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
		msg := domain.Message{
			Role:    c.Message.Role,
			Content: c.Message.Content,
			Name:    c.Message.Name,
		}

		// Convert tool calls from OpenAI format
		if len(c.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]domain.ToolCall, len(c.Message.ToolCalls))
			for j, tc := range c.Message.ToolCalls {
				msg.ToolCalls[j] = domain.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: domain.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choices[i] = domain.Choice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
			Logprobs:     c.Logprobs,
		}
	}

	return &domain.CanonicalResponse{
		ID:                apiResp.ID,
		Object:            apiResp.Object,
		Created:           apiResp.Created,
		Model:             apiResp.Model,
		Choices:           choices,
		SystemFingerprint: apiResp.SystemFingerprint,
		Usage: domain.Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
		SourceAPIType: domain.APITypeOpenAI,
	}
}

// CanonicalToAPIResponse converts a canonical response to OpenAI API format.
func CanonicalToAPIResponse(resp *domain.CanonicalResponse) *openai.ChatCompletionResponse {
	choices := make([]openai.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		msg := openai.ChatCompletionMessage{
			Role:    c.Message.Role,
			Content: c.Message.Content,
			Name:    c.Message.Name,
		}

		// Convert tool calls to OpenAI format
		if len(c.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]openai.ToolCall, len(c.Message.ToolCalls))
			for j, tc := range c.Message.ToolCalls {
				msg.ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choices[i] = openai.Choice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
		}
	}

	return &openai.ChatCompletionResponse{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		SystemFingerprint: resp.SystemFingerprint,
		Choices:           choices,
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
