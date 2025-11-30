package domain

import "encoding/json"

// APIType identifies the API format for frontdoors and providers.
type APIType string

const (
	APITypeOpenAI    APIType = "openai"
	APITypeAnthropic APIType = "anthropic"
	APITypeResponses APIType = "responses" // OpenAI Responses API
)

// Message represents a chat message with support for both simple text
// and multimodal content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"` // Simple text content (for backward compat)
	Name    string `json:"name,omitempty"`

	// RichContent holds multimodal content (images, tool calls, etc.)
	// When set, this takes precedence over Content field.
	RichContent *MessageContent `json:"rich_content,omitempty"`

	// ToolCalls for assistant messages that invoke tools (OpenAI style)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID for tool messages providing results (OpenAI style)
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// GetContent returns the text content of the message.
// If RichContent is set, returns concatenated text parts.
func (m *Message) GetContent() string {
	if m.RichContent != nil {
		return m.RichContent.String()
	}
	return m.Content
}

// HasRichContent returns true if the message has multimodal content.
func (m *Message) HasRichContent() bool {
	return m.RichContent != nil && !m.RichContent.IsSimpleText()
}

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction represents the function details in a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition represents a tool that the model can call.
type ToolDefinition struct {
	// Name is the identifier for the tool (required by OpenAI Responses API).
	Name     string      `json:"name,omitempty"`
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef describes the function signature.
type FunctionDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"` // JSON Schema
}

// CanonicalRequest is the superset of all supported features.
type CanonicalRequest struct {
	TenantID    string            `json:"tenant_id"`
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Stream      bool              `json:"stream"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	Tools       []ToolDefinition  `json:"tools,omitempty"`
	ToolChoice  any               `json:"tool_choice,omitempty"` // "auto", "none", "required", or specific tool
	Metadata    map[string]string `json:"metadata,omitempty"`

	// System prompt (for APIs that support separate system instructions)
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Response format configuration
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Stop sequences
	Stop []string `json:"stop,omitempty"`

	// For Responses API: instructions override system prompt
	Instructions string `json:"instructions,omitempty"`

	// For Responses API: continue from previous response
	PreviousResponseID string `json:"previous_response_id,omitempty"`

	// UserAgent is the User-Agent header from the incoming request.
	// Providers should forward this to upstream APIs for traceability.
	UserAgent string `json:"-"`

	// SourceAPIType identifies the original API format of the incoming request.
	// Used to enable pass-through optimization when frontdoor matches provider.
	SourceAPIType APIType `json:"-"`

	// RawRequest contains the original request body for pass-through mode.
	// Only populated when SourceAPIType matches the target provider type.
	RawRequest json.RawMessage `json:"-"`
}

// ResponseFormat specifies the format of model output.
type ResponseFormat struct {
	Type       string `json:"type"`                  // "text", "json_object", "json_schema"
	JSONSchema any    `json:"json_schema,omitempty"` // Schema for json_schema type
}

// ToolCallChunk represents a partial tool execution.
type ToolCallChunk struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CanonicalResponse represents a complete non-streaming response.
type CanonicalResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`

	// SourceAPIType identifies the API format of the response from the provider.
	SourceAPIType APIType `json:"-"`

	// RawResponse contains the original response body for pass-through mode.
	RawResponse json.RawMessage `json:"-"`

	// SystemFingerprint from the provider (OpenAI specific)
	SystemFingerprint string `json:"system_fingerprint,omitempty"`

	// RateLimits contains rate limit information from the upstream provider.
	RateLimits *RateLimitInfo `json:"-"`

	// ProviderModel tracks the actual model used by the provider (for logging/observability)
	// This is preserved even when Model is rewritten by model mapping
	ProviderModel string `json:"-"`

	// ProviderRequestBody contains the actual request body sent to the upstream provider API.
	// Captured for debugging and visibility into the complete transformation flow.
	ProviderRequestBody json.RawMessage `json:"-"`
}

// RateLimitInfo contains rate limit information from upstream providers.
type RateLimitInfo struct {
	// Request limits
	RequestsLimit     int    `json:"requests_limit,omitempty"`
	RequestsRemaining int    `json:"requests_remaining,omitempty"`
	RequestsReset     string `json:"requests_reset,omitempty"` // Duration or timestamp

	// Token limits
	TokensLimit     int    `json:"tokens_limit,omitempty"`
	TokensRemaining int    `json:"tokens_remaining,omitempty"`
	TokensReset     string `json:"tokens_reset,omitempty"` // Duration or timestamp
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Logprobs     any     `json:"logprobs,omitempty"`
}

// StreamEventType identifies the type of streaming event.
type StreamEventType string

const (
	// Generic events
	EventTypeContentDelta StreamEventType = "content_delta"
	EventTypeContentDone  StreamEventType = "content_done"
	EventTypeError        StreamEventType = "error"
	EventTypeDone         StreamEventType = "done"

	// Message lifecycle events
	EventTypeMessageStart StreamEventType = "message_start"
	EventTypeMessageDelta StreamEventType = "message_delta"
	EventTypeMessageStop  StreamEventType = "message_stop"

	// Content block events (Anthropic style)
	EventTypeContentBlockStart StreamEventType = "content_block_start"
	EventTypeContentBlockDelta StreamEventType = "content_block_delta"
	EventTypeContentBlockStop  StreamEventType = "content_block_stop"

	// Responses API events (per OpenAI Responses API Spec v1.1)
	EventTypeResponseCreated         StreamEventType = "response.created"
	EventTypeResponseOutputItemAdd   StreamEventType = "response.output_item.added"
	EventTypeResponseOutputItemDelta StreamEventType = "response.output_item.delta"
	EventTypeResponseOutputItemDone  StreamEventType = "response.output_item.done"
	EventTypeResponseDone            StreamEventType = "response.done"
	EventTypeResponseFailed          StreamEventType = "response.failed"
)

// CanonicalEvent represents a streaming event.
type CanonicalEvent struct {
	// Type identifies the event type for structured event handling.
	Type StreamEventType

	// Role for message start events
	Role string

	// ContentDelta for text content streaming
	ContentDelta string

	// ToolCall for streaming tool call data
	ToolCall *ToolCallChunk

	// Usage for token count updates
	Usage *Usage

	// Error for error events
	Error error

	// Index for content block events (which block this relates to)
	Index int

	// ContentBlock for content block start events
	ContentBlock *ContentPart

	// FinishReason for message/response completion
	FinishReason string

	// ID for response events
	ResponseID string

	// Model for message start events (may be rewritten for client display)
	Model string

	// ProviderModel tracks the actual model used by the provider (for logging/observability)
	// This is preserved even when Model is rewritten by model mapping
	ProviderModel string

	// RawEvent for pass-through mode (original SSE data)
	RawEvent json.RawMessage
}

// Model describes a model entry exposed via the frontdoor.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
	Created int64  `json:"created,omitempty"`
}

// ModelList is the canonical model listing response.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}
