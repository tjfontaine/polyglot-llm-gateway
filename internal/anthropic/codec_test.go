package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

func TestCodec_Name(t *testing.T) {
	c := NewCodec()
	if got := c.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
}

func TestCodec_DecodeRequest(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantModel     string
		wantMaxTok    int
		wantMsgCount  int
		wantFirstRole string // Check first message role (e.g., "system" when system prompt present)
		wantErr       bool
	}{
		{
			name: "basic request with string content",
			input: `{
				"model": "claude-3-haiku-20240307",
				"max_tokens": 100,
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`,
			wantModel:     "claude-3-haiku-20240307",
			wantMaxTok:    100,
			wantMsgCount:  1,
			wantFirstRole: "user",
		},
		{
			name: "request with system prompt",
			input: `{
				"model": "claude-3-opus-20240229",
				"max_tokens": 200,
				"system": "You are helpful",
				"messages": [
					{"role": "user", "content": "Hi"}
				]
			}`,
			wantModel:     "claude-3-opus-20240229",
			wantMaxTok:    200,
			wantMsgCount:  2,        // system + user message
			wantFirstRole: "system", // system messages are prepended
		},
		{
			name: "request with temperature",
			input: `{
				"model": "claude-3-sonnet-20240229",
				"max_tokens": 50,
				"temperature": 0.7,
				"messages": [
					{"role": "user", "content": "Test"}
				]
			}`,
			wantModel:     "claude-3-sonnet-20240229",
			wantMaxTok:    50,
			wantMsgCount:  1,
			wantFirstRole: "user",
		},
		{
			name:    "invalid JSON",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	c := NewCodec()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.DecodeRequest([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", got.Model, tt.wantModel)
			}
			if got.MaxTokens != tt.wantMaxTok {
				t.Errorf("MaxTokens = %d, want %d", got.MaxTokens, tt.wantMaxTok)
			}
			if len(got.Messages) != tt.wantMsgCount {
				t.Errorf("len(Messages) = %d, want %d", len(got.Messages), tt.wantMsgCount)
			}
			if tt.wantFirstRole != "" && len(got.Messages) > 0 {
				if got.Messages[0].Role != tt.wantFirstRole {
					t.Errorf("Messages[0].Role = %q, want %q", got.Messages[0].Role, tt.wantFirstRole)
				}
			}
		})
	}
}

func TestCodec_EncodeRequest(t *testing.T) {
	c := NewCodec()

	req := &domain.CanonicalRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 100,
		Messages: []domain.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	data, err := c.EncodeRequest(req)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}

	// Decode back to verify
	var apiReq MessagesRequest
	if err := json.Unmarshal(data, &apiReq); err != nil {
		t.Fatalf("Failed to unmarshal encoded request: %v", err)
	}

	if apiReq.Model != req.Model {
		t.Errorf("Model = %q, want %q", apiReq.Model, req.Model)
	}
	if apiReq.MaxTokens != req.MaxTokens {
		t.Errorf("MaxTokens = %d, want %d", apiReq.MaxTokens, req.MaxTokens)
	}
}

func TestCodec_DecodeResponse(t *testing.T) {
	c := NewCodec()

	input := `{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"model": "claude-3-haiku-20240307",
		"content": [
			{"type": "text", "text": "Hello there!"}
		],
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 10,
			"output_tokens": 5
		}
	}`

	got, err := c.DecodeResponse([]byte(input))
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}

	if got.ID != "msg_123" {
		t.Errorf("ID = %q, want %q", got.ID, "msg_123")
	}
	if got.Model != "claude-3-haiku-20240307" {
		t.Errorf("Model = %q, want %q", got.Model, "claude-3-haiku-20240307")
	}
	if len(got.Choices) != 1 {
		t.Fatalf("len(Choices) = %d, want 1", len(got.Choices))
	}
	if got.Choices[0].Message.Content != "Hello there!" {
		t.Errorf("Content = %q, want %q", got.Choices[0].Message.Content, "Hello there!")
	}
	if got.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", got.Usage.PromptTokens)
	}
	if got.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", got.Usage.CompletionTokens)
	}
}

func TestCodec_EncodeResponse(t *testing.T) {
	c := NewCodec()

	resp := &domain.CanonicalResponse{
		ID:    "resp_456",
		Model: "claude-3-opus-20240229",
		Choices: []domain.Choice{
			{
				Index:        0,
				Message:      domain.Message{Role: "assistant", Content: "Test response"},
				FinishReason: "stop",
			},
		},
		Usage: domain.Usage{
			PromptTokens:     20,
			CompletionTokens: 10,
			TotalTokens:      30,
		},
	}

	data, err := c.EncodeResponse(resp)
	if err != nil {
		t.Fatalf("EncodeResponse() error = %v", err)
	}

	var apiResp MessagesResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		t.Fatalf("Failed to unmarshal encoded response: %v", err)
	}

	if apiResp.ID != resp.ID {
		t.Errorf("ID = %q, want %q", apiResp.ID, resp.ID)
	}
	if apiResp.Model != resp.Model {
		t.Errorf("Model = %q, want %q", apiResp.Model, resp.Model)
	}
}

func TestCodec_DecodeStreamChunk(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDelta    string
		wantRole     string
		wantHasUsage bool
		wantErr      bool
	}{
		{
			name:      "content_block_delta",
			input:     `{"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}`,
			wantDelta: "Hello",
		},
		{
			name:         "message_start",
			input:        `{"type": "message_start", "message": {"id": "msg_123", "model": "claude-3-haiku-20240307", "role": "assistant", "usage": {"input_tokens": 10}}}`,
			wantRole:     "assistant",
			wantHasUsage: true,
		},
		{
			name:         "message_delta with usage",
			input:        `{"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"output_tokens": 10}}`,
			wantHasUsage: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	c := NewCodec()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.DecodeStreamChunk([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeStreamChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ContentDelta != tt.wantDelta {
				t.Errorf("ContentDelta = %q, want %q", got.ContentDelta, tt.wantDelta)
			}
			if got.Role != tt.wantRole {
				t.Errorf("Role = %q, want %q", got.Role, tt.wantRole)
			}
			if tt.wantHasUsage && got.Usage == nil {
				t.Error("Usage = nil, want non-nil")
			}
		})
	}
}

func TestAPIRequestToCanonical_ContentBlocks(t *testing.T) {
	// Test that content blocks are properly merged via JSON decode path
	input := `{
		"model": "claude-3-haiku-20240307",
		"max_tokens": 100,
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "Hello "},
					{"type": "text", "text": "world"}
				]
			}
		]
	}`

	c := NewCodec()
	got, err := c.DecodeRequest([]byte(input))
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(got.Messages))
	}
	if got.Messages[0].Content != "Hello world" {
		t.Errorf("Content = %q, want %q", got.Messages[0].Content, "Hello world")
	}
}

func TestCanonicalToAPIRequest(t *testing.T) {
	req := &domain.CanonicalRequest{
		Model:        "claude-3-opus-20240229",
		MaxTokens:    200,
		SystemPrompt: "Be helpful",
		Messages: []domain.Message{
			{Role: "user", Content: "Question"},
			{Role: "assistant", Content: "Answer"},
		},
	}

	got := CanonicalToAPIRequest(req)

	if got.Model != req.Model {
		t.Errorf("Model = %q, want %q", got.Model, req.Model)
	}
	if got.MaxTokens != req.MaxTokens {
		t.Errorf("MaxTokens = %d, want %d", got.MaxTokens, req.MaxTokens)
	}
	// System is a SystemMessages type, check the string representation
	if len(got.System) == 0 || got.System[0].Text != req.SystemPrompt {
		t.Errorf("System = %v, want text %q", got.System, req.SystemPrompt)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(got.Messages))
	}
}

func TestCanonicalToAPIResponse(t *testing.T) {
	resp := &domain.CanonicalResponse{
		ID:    "resp_test",
		Model: "claude-3-sonnet-20240229",
		Choices: []domain.Choice{
			{
				Message:      domain.Message{Role: "assistant", Content: "Response text"},
				FinishReason: "stop",
			},
		},
		Usage: domain.Usage{
			PromptTokens:     50,
			CompletionTokens: 25,
			TotalTokens:      75,
		},
	}

	got := CanonicalToAPIResponse(resp)

	if got.ID != resp.ID {
		t.Errorf("ID = %q, want %q", got.ID, resp.ID)
	}
	if got.Model != resp.Model {
		t.Errorf("Model = %q, want %q", got.Model, resp.Model)
	}
	if got.Role != "assistant" {
		t.Errorf("Role = %q, want %q", got.Role, "assistant")
	}
	if len(got.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(got.Content))
	}
	if got.Content[0].Text != "Response text" {
		t.Errorf("Content[0].Text = %q, want %q", got.Content[0].Text, "Response text")
	}
	if got.Usage.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", got.Usage.InputTokens)
	}
	if got.Usage.OutputTokens != 25 {
		t.Errorf("OutputTokens = %d, want 25", got.Usage.OutputTokens)
	}
}
