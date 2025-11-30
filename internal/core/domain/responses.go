package domain

import (
	"encoding/json"
	"time"
)

// ResponsesAPIRequest represents a request to the OpenAI Responses API.
// This supports the full spec including input types, tools, and conversation continuation.
type ResponsesAPIRequest struct {
	// Model to use for the response
	Model string `json:"model"`

	// Input can be a string, array of input items, or array of messages
	Input ResponsesInput `json:"input"`

	// Instructions to guide the model (replaces system prompt in thread context)
	Instructions string `json:"instructions,omitempty"`

	// Tools available for the model to use
	Tools []ResponsesTool `json:"tools,omitempty"`

	// ToolChoice controls how the model uses tools
	ToolChoice any `json:"tool_choice,omitempty"` // "auto", "none", "required", or {"type": "function", "name": "..."}

	// ParallelToolCalls allows multiple tool calls in a single turn
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`

	// MaxOutputTokens limits the response length
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`

	// Temperature controls randomness
	Temperature *float32 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	TopP *float32 `json:"top_p,omitempty"`

	// ResponseFormat specifies output format
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Stream enables streaming responses
	Stream bool `json:"stream,omitempty"`

	// StreamOptions configures streaming behavior
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`

	// PreviousResponseID to continue a conversation
	PreviousResponseID string `json:"previous_response_id,omitempty"`

	// Metadata for the response
	Metadata map[string]string `json:"metadata,omitempty"`

	// TruncationStrategy for handling long conversations
	TruncationStrategy *TruncationStrategy `json:"truncation_strategy,omitempty"`

	// Store whether to store the response for later retrieval
	Store *bool `json:"store,omitempty"`

	// User identifier for tracking
	User string `json:"user,omitempty"`
}

// ResponsesInput can be a string, array of input items, or conversation messages.
type ResponsesInput struct {
	Text  string               // Simple text input
	Items []ResponsesInputItem // Array of input items
}

// MarshalJSON implements json.Marshaler.
func (ri ResponsesInput) MarshalJSON() ([]byte, error) {
	if ri.Text != "" {
		return json.Marshal(ri.Text)
	}
	return json.Marshal(ri.Items)
}

// UnmarshalJSON implements json.Unmarshaler.
func (ri *ResponsesInput) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		ri.Text = str
		ri.Items = nil
		return nil
	}

	// Try array of input items
	var items []ResponsesInputItem
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	ri.Items = items
	ri.Text = ""
	return nil
}

// ResponsesInputItem represents a single input item.
type ResponsesInputItem struct {
	Type string `json:"type"` // "message", "item_reference"

	// For message type
	Role    string                 `json:"role,omitempty"`
	Content []ResponsesContentPart `json:"content,omitempty"`

	// For item_reference type
	ID string `json:"id,omitempty"`
}

// ResponsesContentPart represents a content part in input/output.
type ResponsesContentPart struct {
	Type string `json:"type"` // "input_text", "input_image", "input_audio", "output_text", "refusal"

	// For text content
	Text string `json:"text,omitempty"`

	// For image content
	ImageURL *ImageURL    `json:"image_url,omitempty"`
	Source   *ImageSource `json:"source,omitempty"` // base64 image

	// For audio content
	InputAudio *InputAudio `json:"input_audio,omitempty"`

	// For annotations (citations, file references)
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Annotation represents an annotation on content (citations, etc.)
type Annotation struct {
	Type       string `json:"type"` // "file_citation", "url_citation"
	Text       string `json:"text,omitempty"`
	StartIndex int    `json:"start_index,omitempty"`
	EndIndex   int    `json:"end_index,omitempty"`

	// For file citations
	FileCitation *FileCitation `json:"file_citation,omitempty"`

	// For URL citations
	URLCitation *URLCitation `json:"url_citation,omitempty"`
}

// FileCitation represents a file citation annotation.
type FileCitation struct {
	FileID string `json:"file_id"`
	Quote  string `json:"quote,omitempty"`
}

// URLCitation represents a URL citation annotation.
type URLCitation struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// ResponsesTool represents a tool definition for the Responses API.
type ResponsesTool struct {
	Type string `json:"type"` // "function", "code_interpreter", "file_search"

	// For function type
	Function *FunctionDef `json:"function,omitempty"`

	// For file_search type
	FileSearch *FileSearchConfig `json:"file_search,omitempty"`

	// For code_interpreter type
	CodeInterpreter *CodeInterpreterConfig `json:"code_interpreter,omitempty"`
}

// FileSearchConfig configures file search tool.
type FileSearchConfig struct {
	MaxNumResults int    `json:"max_num_results,omitempty"`
	VectorStoreID string `json:"vector_store_id,omitempty"`
	Filters       any    `json:"filters,omitempty"`
}

// CodeInterpreterConfig configures code interpreter tool.
type CodeInterpreterConfig struct {
	FileIDs   []string `json:"file_ids,omitempty"`
	Container string   `json:"container,omitempty"`
}

// StreamOptions configures streaming behavior.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// TruncationStrategy defines how to handle long conversations.
type TruncationStrategy struct {
	Type         string `json:"type"` // "auto", "last_messages"
	LastMessages int    `json:"last_messages,omitempty"`
}

// ResponsesAPIResponse represents a response from the Responses API.
type ResponsesAPIResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"` // "response"
	CreatedAt int64  `json:"created_at"`

	// Status of the response
	Status string `json:"status"` // "in_progress", "completed", "failed", "cancelled"

	// The model used
	Model string `json:"model"`

	// Output items produced by the model
	Output []ResponsesOutputItem `json:"output"`

	// Usage statistics
	Usage *ResponsesUsage `json:"usage,omitempty"`

	// Error information if status is "failed"
	Error *ResponsesError `json:"error,omitempty"`

	// Metadata from the request
	Metadata map[string]string `json:"metadata,omitempty"`

	// For continuing conversations
	PreviousResponseID string `json:"previous_response_id,omitempty"`
}

// ResponsesOutputItem represents an output item in the response.
type ResponsesOutputItem struct {
	Type string `json:"type"` // "message", "function_call", "function_call_output", "file"

	ID string `json:"id,omitempty"`

	// For message type
	Role    string                 `json:"role,omitempty"`
	Content []ResponsesContentPart `json:"content,omitempty"`

	// For function_call type
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// For function_call_output type
	Output string `json:"output,omitempty"`

	// Status for the item
	Status string `json:"status,omitempty"`
}

// ResponsesUsage represents token usage in Responses API.
type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	// Detailed breakdowns
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

// Responses API Streaming Events (per OpenAI Responses API Spec v1.1)
//
// Event flow:
// 1. response.created - sent immediately when request is accepted
// 2. response.output_item.added - when a new Item (message or function_call) begins
// 3. response.output_item.delta - for streaming content or arguments
// 4. response.output_item.done - when an Item is fully generated
// 5. response.done - final event with usage and finish_reason

// ResponsesStreamEvent represents a streaming event from the Responses API.
type ResponsesStreamEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ResponseCreatedEvent is sent when a response is created.
// Event: response.created
type ResponseCreatedEvent struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created"`
	Model     string `json:"model"`
}

// OutputItemAddedEvent is sent when an output item is added.
// Event: response.output_item.added
type OutputItemAddedEvent struct {
	ItemIndex int                 `json:"item_index"`
	Item      ResponsesOutputItem `json:"item"`
}

// OutputItemDeltaEvent is sent for streaming content or function arguments.
// Event: response.output_item.delta
type OutputItemDeltaEvent struct {
	ItemIndex int             `json:"item_index"`
	Delta     OutputItemDelta `json:"delta"`
}

// OutputItemDelta contains either content delta or arguments delta.
type OutputItemDelta struct {
	Content   string `json:"content,omitempty"`   // For text content
	Arguments string `json:"arguments,omitempty"` // For function arguments
}

// OutputItemDoneEvent is sent when an output item is complete.
// Event: response.output_item.done
type OutputItemDoneEvent struct {
	ItemIndex int                 `json:"item_index"`
	Item      ResponsesOutputItem `json:"item"`
}

// ResponseDoneEvent is sent when the entire response is complete.
// Event: response.done
type ResponseDoneEvent struct {
	Usage        *ResponsesUsage `json:"usage,omitempty"`
	FinishReason string          `json:"finish_reason"` // "stop", "tool_calls", "length", etc.
}

// ResponseFailedEvent is sent when a response fails.
// Event: response.failed
type ResponseFailedEvent struct {
	Response ResponsesAPIResponse `json:"response"`
}

// Legacy event types - kept for backwards compatibility with existing code
// These wrap the new types for internal use

// LegacyResponseCreatedEvent wraps ResponseCreatedEvent for internal handler use
type LegacyResponseCreatedEvent struct {
	Type     string               `json:"type"` // "response.created"
	Response ResponsesAPIResponse `json:"response"`
}

// LegacyResponseFailedEvent wraps error response for internal handler use
type LegacyResponseFailedEvent struct {
	Type     string               `json:"type"` // "response.failed"
	Response ResponsesAPIResponse `json:"response"`
}

// Helper functions for converting between canonical and Responses API formats

// ToResponsesAPIResponse converts a CanonicalResponse to ResponsesAPIResponse.
func ToResponsesAPIResponse(resp *CanonicalResponse) *ResponsesAPIResponse {
	output := make([]ResponsesOutputItem, 0, len(resp.Choices))
	hasToolCalls := false

	for _, choice := range resp.Choices {
		content := make([]ResponsesContentPart, 0, 1)

		// Handle text content
		if choice.Message.Content != "" {
			content = append(content, ResponsesContentPart{
				Type: "output_text",
				Text: choice.Message.Content,
			})
		}

		// Handle tool calls
		for _, tc := range choice.Message.ToolCalls {
			hasToolCalls = true
			output = append(output, ResponsesOutputItem{
				Type:      "function_call",
				ID:        tc.ID,
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
				Status:    "completed",
			})
		}

		// Add message if we have content
		if len(content) > 0 {
			output = append(output, ResponsesOutputItem{
				Type:    "message",
				Role:    choice.Message.Role,
				Content: content,
				Status:  "completed",
			})
		}
	}

	// Determine response status based on finish reason
	// Per OpenAI Responses API spec:
	// - "completed": The response finished normally
	// - "incomplete": The response requires client-side action (tool calls)
	// - "failed": The response encountered an error
	// - "cancelled": The response was cancelled
	status := "completed"
	if len(resp.Choices) > 0 {
		finishReason := resp.Choices[0].FinishReason
		// Map Anthropic "tool_use" and OpenAI "tool_calls" to "incomplete" status
		// This signals to the client that they need to execute tools and continue
		if finishReason == "tool_calls" || finishReason == "tool_use" || hasToolCalls {
			status = "incomplete"
		}
	}

	return &ResponsesAPIResponse{
		ID:        resp.ID,
		Object:    "response",
		CreatedAt: resp.Created,
		Status:    status,
		Model:     resp.Model,
		Output:    output,
		Usage: &ResponsesUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
}

// FromResponsesAPIRequest converts a ResponsesAPIRequest to CanonicalRequest.
func FromResponsesAPIRequest(req *ResponsesAPIRequest) *CanonicalRequest {
	canonical := &CanonicalRequest{
		Model:              req.Model,
		Stream:             req.Stream,
		Instructions:       req.Instructions,
		PreviousResponseID: req.PreviousResponseID,
		Metadata:           req.Metadata,
	}

	if req.MaxOutputTokens > 0 {
		canonical.MaxTokens = req.MaxOutputTokens
	}

	if req.Temperature != nil {
		canonical.Temperature = *req.Temperature
	}

	if req.TopP != nil {
		canonical.TopP = *req.TopP
	}

	canonical.ToolChoice = req.ToolChoice
	canonical.ResponseFormat = req.ResponseFormat

	// Convert input to messages
	if req.Input.Text != "" {
		canonical.Messages = []Message{{Role: "user", Content: req.Input.Text}}
	} else {
		for _, item := range req.Input.Items {
			if item.Type == "message" {
				msg := Message{Role: item.Role}
				// Extract text content
				for _, part := range item.Content {
					if part.Type == "input_text" || part.Type == "output_text" {
						msg.Content += part.Text
					}
				}
				canonical.Messages = append(canonical.Messages, msg)
			}
		}
	}

	// Convert tools
	for _, tool := range req.Tools {
		if tool.Type == "function" && tool.Function != nil {
			canonical.Tools = append(canonical.Tools, ToolDefinition{
				Type:     "function",
				Function: *tool.Function,
			})
		}
	}

	// Add instructions as system message if provided
	if req.Instructions != "" {
		canonical.SystemPrompt = req.Instructions
	}

	return canonical
}

// ResponsesRecord represents a stored response for retrieval.
type ResponsesRecord struct {
	ID                 string               `json:"id"`
	TenantID           string               `json:"tenant_id"`
	Response           ResponsesAPIResponse `json:"response"`
	Request            ResponsesAPIRequest  `json:"request"`
	CreatedAt          time.Time            `json:"created_at"`
	PreviousResponseID string               `json:"previous_response_id,omitempty"`
}
