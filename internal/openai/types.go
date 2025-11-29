// Package openai provides shared types and HTTP client for OpenAI API interactions.
// These types are used by both the frontdoor handlers and the upstream provider.
package openai

import (
	"encoding/json"
	"strings"

	domainerrors "github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ChatCompletionRequest represents an OpenAI chat completion request.
type ChatCompletionRequest struct {
	Model               string                  `json:"model"`
	Messages            []ChatCompletionMessage `json:"messages"`
	MaxTokens           int                     `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                     `json:"max_completion_tokens,omitempty"`
	Temperature         *float32                `json:"temperature,omitempty"`
	TopP                *float32                `json:"top_p,omitempty"`
	N                   int                     `json:"n,omitempty"`
	Stream              bool                    `json:"stream,omitempty"`
	StreamOptions       *StreamOptions          `json:"stream_options,omitempty"`
	Stop                []string                `json:"stop,omitempty"`
	PresencePenalty     float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty    float32                 `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]int          `json:"logit_bias,omitempty"`
	User                string                  `json:"user,omitempty"`
	Tools               []Tool                  `json:"tools,omitempty"`
	ToolChoice          any                     `json:"tool_choice,omitempty"`
	ResponseFormat      *ResponseFormat         `json:"response_format,omitempty"`
	Seed                *int                    `json:"seed,omitempty"`
}

// StreamOptions configures streaming behavior.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatCompletionMessage represents a message in the chat completion request/response.
type ChatCompletionMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool represents a tool that the model can call.
type Tool struct {
	Type     string       `json:"type"`
	Function FunctionTool `json:"function"`
}

// FunctionTool describes a function tool.
type FunctionTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ToolCall represents a tool call made by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatCompletionResponse represents an OpenAI chat completion response.
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage,omitempty"`

	// RawBody contains the original response JSON for debugging
	RawBody json.RawMessage `json:"-"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
	Logprobs     *Logprobs             `json:"logprobs,omitempty"`
}

// Logprobs contains log probability information.
type Logprobs struct {
	Content []LogprobContent `json:"content,omitempty"`
}

// LogprobContent contains token log probabilities.
type LogprobContent struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk represents a streaming chunk.
type ChatCompletionChunk struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
	Choices           []ChunkChoice `json:"choices"`
	Usage             *Usage        `json:"usage,omitempty"`
}

// ChunkChoice represents a choice in a streaming chunk.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
	Logprobs     *Logprobs  `json:"logprobs,omitempty"`
}

// ChunkDelta represents the delta content in a streaming chunk.
type ChunkDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []ToolCallChunk `json:"tool_calls,omitempty"`
}

// ToolCallChunk represents a partial tool call in streaming.
type ToolCallChunk struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function *FunctionCallChunk `json:"function,omitempty"`
}

// FunctionCallChunk represents a partial function call.
type FunctionCallChunk struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Model represents an OpenAI model.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelList represents a list of models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// ErrorResponse represents an OpenAI API error response.
type ErrorResponse struct {
	Error *APIError `json:"error"`
}

// APIError contains error details.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// ToCanonical converts the OpenAI API error to a canonical domain error.
func (e *APIError) ToCanonical() *domainerrors.APIError {
	errType, code := mapOpenAIErrorType(e.Type, e.Code, e.Message)
	return &domainerrors.APIError{
		Type:      errType,
		Code:      code,
		Message:   e.Message,
		Param:     e.Param,
		SourceAPI: domainerrors.APITypeOpenAI,
	}
}

// mapOpenAIErrorType maps OpenAI error types/codes to domain error types.
func mapOpenAIErrorType(errType, errCode, message string) (domainerrors.ErrorType, domainerrors.ErrorCode) {
	// First check specific error codes
	switch errCode {
	case "context_length_exceeded":
		return domainerrors.ErrorTypeContextLength, domainerrors.ErrorCodeContextLengthExceeded
	case "rate_limit_exceeded":
		return domainerrors.ErrorTypeRateLimit, domainerrors.ErrorCodeRateLimitExceeded
	case "invalid_api_key":
		return domainerrors.ErrorTypeAuthentication, domainerrors.ErrorCodeInvalidAPIKey
	case "model_not_found":
		return domainerrors.ErrorTypeNotFound, domainerrors.ErrorCodeModelNotFound
	}

	// Check message for patterns
	msgLower := strings.ToLower(message)
	if strings.Contains(msgLower, "max_tokens") || strings.Contains(msgLower, "maximum tokens") {
		if strings.Contains(msgLower, "truncated") || strings.Contains(msgLower, "could not finish") ||
			strings.Contains(msgLower, "output limit") {
			return domainerrors.ErrorTypeMaxTokens, domainerrors.ErrorCodeOutputTruncated
		}
		return domainerrors.ErrorTypeMaxTokens, domainerrors.ErrorCodeMaxTokensExceeded
	}
	if strings.Contains(msgLower, "context length") || strings.Contains(msgLower, "context window") {
		return domainerrors.ErrorTypeContextLength, domainerrors.ErrorCodeContextLengthExceeded
	}

	// Map by error type
	switch errType {
	case "invalid_request_error":
		return domainerrors.ErrorTypeInvalidRequest, ""
	case "authentication_error":
		return domainerrors.ErrorTypeAuthentication, domainerrors.ErrorCodeInvalidAPIKey
	case "permission_denied":
		return domainerrors.ErrorTypePermission, ""
	case "not_found":
		return domainerrors.ErrorTypeNotFound, domainerrors.ErrorCodeModelNotFound
	case "rate_limit_error", "rate_limit_exceeded":
		return domainerrors.ErrorTypeRateLimit, domainerrors.ErrorCodeRateLimitExceeded
	case "service_unavailable":
		return domainerrors.ErrorTypeOverloaded, ""
	case "server_error":
		return domainerrors.ErrorTypeServer, ""
	default:
		return domainerrors.ErrorTypeServer, ""
	}
}

// ParseErrorResponse attempts to parse an error response from JSON.
func ParseErrorResponse(data []byte) (*APIError, error) {
	var errResp ErrorResponse
	if err := json.Unmarshal(data, &errResp); err != nil {
		return nil, err
	}
	if errResp.Error == nil {
		return nil, nil
	}
	return errResp.Error, nil
}

// ========== OpenAI Responses API Types ==========

// ResponsesRequest represents a request to OpenAI's Responses API.
type ResponsesRequest struct {
	Model              string              `json:"model"`
	Input              any                 `json:"input"` // string, []InputItem, or []Message
	Instructions       string              `json:"instructions,omitempty"`
	Tools              []ResponsesTool     `json:"tools,omitempty"`
	ToolChoice         any                 `json:"tool_choice,omitempty"`
	MaxOutputTokens    int                 `json:"max_output_tokens,omitempty"`
	Temperature        *float32            `json:"temperature,omitempty"`
	TopP               *float32            `json:"top_p,omitempty"`
	Stream             bool                `json:"stream,omitempty"`
	StreamOptions      *StreamOptions      `json:"stream_options,omitempty"`
	Metadata           map[string]string   `json:"metadata,omitempty"`
	Store              *bool               `json:"store,omitempty"`
	TruncationStrategy *TruncationStrategy `json:"truncation_strategy,omitempty"`
	ResponseFormat     *ResponseFormat     `json:"response_format,omitempty"`
}

// ResponsesTool represents a tool in the Responses API.
type ResponsesTool struct {
	Type     string       `json:"type"` // "function"
	Function FunctionTool `json:"function,omitempty"`
}

// TruncationStrategy defines how to handle long conversations.
type TruncationStrategy struct {
	Type         string `json:"type"` // "auto", "last_messages"
	LastMessages int    `json:"last_messages,omitempty"`
}

// ResponsesInputItem represents an input item in the Responses API.
type ResponsesInputItem struct {
	Type    string                 `json:"type"` // "message", "item_reference"
	Role    string                 `json:"role,omitempty"`
	Content []ResponsesContentPart `json:"content,omitempty"`
	ID      string                 `json:"id,omitempty"` // for item_reference
}

// ResponsesContentPart represents a content part in the Responses API.
type ResponsesContentPart struct {
	Type     string    `json:"type"` // "input_text", "input_image", "output_text"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL in content.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// ResponsesResponse represents a response from OpenAI's Responses API.
type ResponsesResponse struct {
	ID                 string                `json:"id"`
	Object             string                `json:"object"` // "response"
	CreatedAt          int64                 `json:"created_at"`
	Status             string                `json:"status"` // "completed", "failed", "in_progress", "cancelled"
	Model              string                `json:"model"`
	Output             []ResponsesOutputItem `json:"output"`
	Usage              *ResponsesUsage       `json:"usage,omitempty"`
	Error              *ResponsesError       `json:"error,omitempty"`
	Metadata           map[string]string     `json:"metadata,omitempty"`
	PreviousResponseID string                `json:"previous_response_id,omitempty"`

	// RawBody contains the original response JSON for debugging
	RawBody json.RawMessage `json:"-"`
}

// ResponsesOutputItem represents an output item from the Responses API.
type ResponsesOutputItem struct {
	Type      string                 `json:"type"` // "message", "function_call", "function_call_output"
	ID        string                 `json:"id,omitempty"`
	Role      string                 `json:"role,omitempty"`
	Content   []ResponsesContentPart `json:"content,omitempty"`
	Status    string                 `json:"status,omitempty"`
	CallID    string                 `json:"call_id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Arguments string                 `json:"arguments,omitempty"`
	Output    string                 `json:"output,omitempty"`
}

// ResponsesUsage represents usage in the Responses API.
type ResponsesUsage struct {
	InputTokens        int           `json:"input_tokens"`
	OutputTokens       int           `json:"output_tokens"`
	TotalTokens        int           `json:"total_tokens"`
	InputTokenDetails  *TokenDetails `json:"input_token_details,omitempty"`
	OutputTokenDetails *TokenDetails `json:"output_token_details,omitempty"`
}

// TokenDetails provides detailed token counts.
type TokenDetails struct {
	CachedTokens    int `json:"cached_tokens,omitempty"`
	AudioTokens     int `json:"audio_tokens,omitempty"`
	TextTokens      int `json:"text_tokens,omitempty"`
	ImageTokens     int `json:"image_tokens,omitempty"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// ResponsesError represents an error in the Responses API.
type ResponsesError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ResponsesStreamChunk represents a streaming chunk from the Responses API.
type ResponsesStreamChunk struct {
	Type string          `json:"type"` // event type
	Data json.RawMessage `json:"data,omitempty"`

	// For response.created, response.completed, etc.
	Response *ResponsesResponse `json:"response,omitempty"`

	// For output_item events
	OutputIndex int                  `json:"output_index,omitempty"`
	Item        *ResponsesOutputItem `json:"item,omitempty"`

	// For content_part events
	ItemID       string                `json:"item_id,omitempty"`
	ContentIndex int                   `json:"content_index,omitempty"`
	Part         *ResponsesContentPart `json:"part,omitempty"`

	// For text delta events
	Delta string `json:"delta,omitempty"`
	Text  string `json:"text,omitempty"`
}
