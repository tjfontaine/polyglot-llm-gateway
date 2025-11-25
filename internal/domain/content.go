package domain

import "encoding/json"

// ContentType represents the type of content in a message.
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeImageURL   ContentType = "image_url"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
	ContentTypeInputAudio ContentType = "input_audio"
)

// ContentPart represents a single part of message content.
// This supports multimodal content (text, images, tool calls, etc.)
type ContentPart struct {
	Type ContentType `json:"type"`

	// For text content
	Text string `json:"text,omitempty"`

	// For image content (base64)
	Source *ImageSource `json:"source,omitempty"`

	// For image_url content (OpenAI style)
	ImageURL *ImageURL `json:"image_url,omitempty"`

	// For tool_use blocks (assistant calling a tool)
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`

	// For tool_result blocks (user providing tool output)
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`

	// For audio content
	InputAudio *InputAudio `json:"input_audio,omitempty"`

	// Cache control for Anthropic prompt caching
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ImageSource represents base64-encoded image data (Anthropic style).
type ImageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ImageURL represents a URL reference to an image (OpenAI style).
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// InputAudio represents audio input data.
type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"` // "wav", "mp3"
}

// CacheControl for Anthropic prompt caching.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// MessageContent can be a simple string or an array of ContentParts.
// This allows backward compatibility with simple text messages while
// supporting rich multimodal content.
type MessageContent struct {
	Text  string        // Simple text content
	Parts []ContentPart // Rich multimodal content
}

// IsSimpleText returns true if the content is just plain text.
func (mc *MessageContent) IsSimpleText() bool {
	return len(mc.Parts) == 0
}

// String returns the text content, concatenating all text parts if multimodal.
func (mc *MessageContent) String() string {
	if mc.IsSimpleText() {
		return mc.Text
	}
	var result string
	for _, part := range mc.Parts {
		if part.Type == ContentTypeText {
			result += part.Text
		}
	}
	return result
}

// MarshalJSON implements json.Marshaler.
func (mc MessageContent) MarshalJSON() ([]byte, error) {
	if mc.IsSimpleText() {
		return json.Marshal(mc.Text)
	}
	return json.Marshal(mc.Parts)
}

// UnmarshalJSON implements json.Unmarshaler.
func (mc *MessageContent) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		mc.Text = str
		mc.Parts = nil
		return nil
	}

	// Try array of content parts
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	mc.Parts = parts
	mc.Text = ""
	return nil
}

// NewTextContent creates a simple text content.
func NewTextContent(text string) MessageContent {
	return MessageContent{Text: text}
}

// NewMultipartContent creates multimodal content from parts.
func NewMultipartContent(parts ...ContentPart) MessageContent {
	return MessageContent{Parts: parts}
}

// TextPart creates a text content part.
func TextPart(text string) ContentPart {
	return ContentPart{Type: ContentTypeText, Text: text}
}

// ImagePart creates an image content part from base64 data.
func ImagePart(mediaType, base64Data string) ContentPart {
	return ContentPart{
		Type: ContentTypeImage,
		Source: &ImageSource{
			Type:      "base64",
			MediaType: mediaType,
			Data:      base64Data,
		},
	}
}

// ImageURLPart creates an image content part from a URL.
func ImageURLPart(url, detail string) ContentPart {
	return ContentPart{
		Type: ContentTypeImageURL,
		ImageURL: &ImageURL{
			URL:    url,
			Detail: detail,
		},
	}
}

// ToolUsePart creates a tool use content part.
func ToolUsePart(id, name string, input any) ContentPart {
	return ContentPart{
		Type:  ContentTypeToolUse,
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// ToolResultPart creates a tool result content part.
func ToolResultPart(toolUseID, content string, isError bool) ContentPart {
	return ContentPart{
		Type:      ContentTypeToolResult,
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}
}
