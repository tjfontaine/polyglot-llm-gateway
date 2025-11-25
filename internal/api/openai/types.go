// Package openai provides shared types and HTTP client for OpenAI API interactions.
// These types are used by both the frontdoor handlers and the upstream provider.
package openai

import "encoding/json"

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
	Index        int         `json:"index"`
	Delta        ChunkDelta  `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
	Logprobs     *Logprobs   `json:"logprobs,omitempty"`
}

// ChunkDelta represents the delta content in a streaming chunk.
type ChunkDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []ToolCallChunk `json:"tool_calls,omitempty"`
}

// ToolCallChunk represents a partial tool call in streaming.
type ToolCallChunk struct {
	Index    int                   `json:"index"`
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type,omitempty"`
	Function *FunctionCallChunk    `json:"function,omitempty"`
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
