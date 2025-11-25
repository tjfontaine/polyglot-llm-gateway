// Package anthropic provides shared types and HTTP client for Anthropic API interactions.
// These types are used by both the frontdoor handlers and the upstream provider.
package anthropic

import (
	"encoding/json"
	"fmt"
)

// MessagesRequest represents an Anthropic Messages API request.
type MessagesRequest struct {
	Model         string         `json:"model"`
	Messages      []Message      `json:"messages"`
	MaxTokens     int            `json:"max_tokens"`
	System        SystemMessages `json:"system,omitempty"`
	Temperature   *float32       `json:"temperature,omitempty"`
	TopP          *float32       `json:"top_p,omitempty"`
	TopK          *int           `json:"top_k,omitempty"`
	Stream        bool           `json:"stream,omitempty"`
	StopSequences []string       `json:"stop_sequences,omitempty"`
	Tools         []Tool         `json:"tools,omitempty"`
	ToolChoice    *ToolChoice    `json:"tool_choice,omitempty"`
	Metadata      *Metadata      `json:"metadata,omitempty"`

	// Extended thinking support (beta)
	Thinking *ThinkingConfig `json:"thinking,omitempty"`
}

// Message represents a message in the conversation.
type Message struct {
	Role    string       `json:"role"`
	Content ContentBlock `json:"content"`
}

// ContentBlock can be a string or array of content blocks.
type ContentBlock []ContentPart

// UnmarshalJSON handles both string and array content formats.
func (c *ContentBlock) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*c = ContentBlock{{Type: "text", Text: str}}
		return nil
	}

	// Try array of content parts
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	*c = parts
	return nil
}

// MarshalJSON serializes content block.
func (c ContentBlock) MarshalJSON() ([]byte, error) {
	// If single text block, could simplify to string, but we keep array for consistency
	return json.Marshal([]ContentPart(c))
}

// String returns the concatenated text content.
func (c ContentBlock) String() string {
	var result string
	for _, part := range c {
		if part.Type == "text" || part.Type == "" {
			result += part.Text
		}
	}
	return result
}

// ContentPart represents a single content part in a message.
type ContentPart struct {
	Type string `json:"type"` // "text", "image", "tool_use", "tool_result", "thinking"
	Text string `json:"text,omitempty"`

	// For tool_use blocks
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`

	// For tool_result blocks
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`

	// For image blocks
	Source *ImageSource `json:"source,omitempty"`

	// For thinking blocks (extended thinking beta)
	Thinking string `json:"thinking,omitempty"`
}

// ImageSource represents an image source.
type ImageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// SystemMessages represents the system prompt (can be string or array).
type SystemMessages []SystemBlock

// UnmarshalJSON handles both string and array system formats.
func (s *SystemMessages) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = SystemMessages{{Type: "text", Text: str}}
		return nil
	}

	// Try array of system blocks
	var blocks []SystemBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		return err
	}
	*s = blocks
	return nil
}

// SystemBlock represents a system message block.
type SystemBlock struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	CacheControl *Cache `json:"cache_control,omitempty"`
}

// Cache represents cache control settings.
type Cache struct {
	Type string `json:"type"` // "ephemeral"
}

// Tool represents a tool that the model can use.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
	Type        string `json:"type,omitempty"` // "custom" (default), "computer_20241022", "text_editor_20241022", "bash_20241022"

	// For computer use tools
	DisplayWidthPx  int `json:"display_width_px,omitempty"`
	DisplayHeightPx int `json:"display_height_px,omitempty"`
	DisplayNumber   int `json:"display_number,omitempty"`
}

// ThinkingConfig configures extended thinking behavior.
type ThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // Max tokens for thinking
}

// ToolChoice represents how the model should use tools.
type ToolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool"
	Name string `json:"name,omitempty"`
}

// Metadata represents request metadata.
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// MessagesResponse represents an Anthropic Messages API response.
type MessagesResponse struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Role         string            `json:"role"`
	Content      []ResponseContent `json:"content"`
	Model        string            `json:"model"`
	StopReason   string            `json:"stop_reason"`
	StopSequence *string           `json:"stop_sequence,omitempty"`
	Usage        MessagesUsage     `json:"usage"`
}

// ResponseContent represents content in a response.
type ResponseContent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// MessagesUsage represents token usage in the response.
type MessagesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Streaming types

// StreamEvent represents a streaming event type.
type StreamEvent struct {
	Type string `json:"type"`
}

// MessageStartEvent is sent at the start of a message.
type MessageStartEvent struct {
	Type    string           `json:"type"`
	Message MessagesResponse `json:"message"`
}

// ContentBlockStartEvent is sent at the start of a content block.
type ContentBlockStartEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	ContentBlock ResponseContent `json:"content_block"`
}

// ContentBlockDeltaEvent is sent for content block updates.
type ContentBlockDeltaEvent struct {
	Type  string     `json:"type"`
	Index int        `json:"index"`
	Delta BlockDelta `json:"delta"`
}

// BlockDelta represents the delta in a content block.
type BlockDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// ContentBlockStopEvent is sent at the end of a content block.
type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// MessageDeltaEvent is sent for message-level updates.
type MessageDeltaEvent struct {
	Type  string       `json:"type"`
	Delta MessageDelta `json:"delta"`
	Usage *DeltaUsage  `json:"usage,omitempty"`
}

// MessageDelta represents updates to the message.
type MessageDelta struct {
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// DeltaUsage represents usage in delta events.
type DeltaUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// MessageStopEvent is sent at the end of a message.
type MessageStopEvent struct {
	Type string `json:"type"`
}

// PingEvent is sent periodically to keep connection alive.
type PingEvent struct {
	Type string `json:"type"`
}

// Model types

// Model represents an Anthropic model.
type Model struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// ModelList represents a list of models.
type ModelList struct {
	Data    []Model `json:"data"`
	HasMore bool    `json:"has_more"`
	FirstID string  `json:"first_id,omitempty"`
	LastID  string  `json:"last_id,omitempty"`
}

// ErrorResponse represents an Anthropic API error.
type ErrorResponse struct {
	Type  string    `json:"type"`
	Error *APIError `json:"error"`
}

// APIError contains error details.
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
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
