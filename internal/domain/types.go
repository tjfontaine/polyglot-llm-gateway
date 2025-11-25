package domain

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ToolDefinition represents a tool that the model can call.
type ToolDefinition struct {
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
	Tools       []ToolDefinition  `json:"tools,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	// UserAgent is the User-Agent header from the incoming request.
	// Providers should forward this to upstream APIs for traceability.
	UserAgent string `json:"-"`
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
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// CanonicalEvent represents a streaming event.
type CanonicalEvent struct {
	Role         string         // e.g., "assistant"
	ContentDelta string         // The text fragment
	ToolCall     *ToolCallChunk // Partial tool execution data
	Usage        *Usage         // Final event often contains token counts
	Error        error          // In-stream errors
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
