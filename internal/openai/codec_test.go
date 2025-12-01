package openai

import (
	"encoding/json"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

func TestCodec_Name(t *testing.T) {
	c := NewCodec()
	if got := c.Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

func TestCodec_DecodeRequest(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantModel    string
		wantMsgCount int
		wantStream   bool
		wantErr      bool
	}{
		{
			name: "basic chat request",
			input: `{
				"model": "gpt-4o",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`,
			wantModel:    "gpt-4o",
			wantMsgCount: 1,
		},
		{
			name: "request with system message",
			input: `{
				"model": "gpt-4o-mini",
				"messages": [
					{"role": "system", "content": "You are helpful"},
					{"role": "user", "content": "Hi"}
				]
			}`,
			wantModel:    "gpt-4o-mini",
			wantMsgCount: 2,
		},
		{
			name: "streaming request",
			input: `{
				"model": "gpt-4",
				"stream": true,
				"messages": [
					{"role": "user", "content": "Test"}
				]
			}`,
			wantModel:    "gpt-4",
			wantMsgCount: 1,
			wantStream:   true,
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
			if len(got.Messages) != tt.wantMsgCount {
				t.Errorf("len(Messages) = %d, want %d", len(got.Messages), tt.wantMsgCount)
			}
			if got.Stream != tt.wantStream {
				t.Errorf("Stream = %v, want %v", got.Stream, tt.wantStream)
			}
		})
	}
}

func TestCodec_EncodeRequest(t *testing.T) {
	c := NewCodec()

	req := &domain.CanonicalRequest{
		Model:     "gpt-4o",
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
	var apiReq ChatCompletionRequest
	if err := json.Unmarshal(data, &apiReq); err != nil {
		t.Fatalf("Failed to unmarshal encoded request: %v", err)
	}

	if apiReq.Model != req.Model {
		t.Errorf("Model = %q, want %q", apiReq.Model, req.Model)
	}
	if len(apiReq.Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1", len(apiReq.Messages))
	}
}

func TestCodec_DecodeResponse(t *testing.T) {
	c := NewCodec()

	input := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1677652288,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello there!"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 5,
			"total_tokens": 15
		}
	}`

	got, err := c.DecodeResponse([]byte(input))
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}

	if got.ID != "chatcmpl-123" {
		t.Errorf("ID = %q, want %q", got.ID, "chatcmpl-123")
	}
	if got.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", got.Model, "gpt-4o")
	}
	if len(got.Choices) != 1 {
		t.Fatalf("len(Choices) = %d, want 1", len(got.Choices))
	}
	if got.Choices[0].Message.Content != "Hello there!" {
		t.Errorf("Content = %q, want %q", got.Choices[0].Message.Content, "Hello there!")
	}
	if got.Choices[0].FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", got.Choices[0].FinishReason, "stop")
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
		Model: "gpt-4o-mini",
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

	var apiResp ChatCompletionResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		t.Fatalf("Failed to unmarshal encoded response: %v", err)
	}

	if apiResp.ID != resp.ID {
		t.Errorf("ID = %q, want %q", apiResp.ID, resp.ID)
	}
	if apiResp.Model != resp.Model {
		t.Errorf("Model = %q, want %q", apiResp.Model, resp.Model)
	}
	if len(apiResp.Choices) != 1 {
		t.Fatalf("len(Choices) = %d, want 1", len(apiResp.Choices))
	}
}

func TestCodec_DecodeStreamChunk(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDelta  string
		wantRole   string
		wantFinish string
		wantErr    bool
	}{
		{
			name: "content delta",
			input: `{
				"id": "chatcmpl-123",
				"object": "chat.completion.chunk",
				"choices": [{
					"index": 0,
					"delta": {"content": "Hello"}
				}]
			}`,
			wantDelta: "Hello",
		},
		{
			name: "role delta",
			input: `{
				"id": "chatcmpl-123",
				"object": "chat.completion.chunk",
				"choices": [{
					"index": 0,
					"delta": {"role": "assistant"}
				}]
			}`,
			wantRole: "assistant",
		},
		{
			name: "finish reason",
			input: `{
				"id": "chatcmpl-123",
				"object": "chat.completion.chunk",
				"choices": [{
					"index": 0,
					"delta": {},
					"finish_reason": "stop"
				}]
			}`,
			wantFinish: "stop",
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
			if got.FinishReason != tt.wantFinish {
				t.Errorf("FinishReason = %q, want %q", got.FinishReason, tt.wantFinish)
			}
		})
	}
}

func TestAPIRequestToCanonical(t *testing.T) {
	temp := float32(0.7)
	apiReq := &ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: "Be helpful"},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: &temp,
		Stream:      true,
	}

	got := APIRequestToCanonical(apiReq)

	if got.Model != apiReq.Model {
		t.Errorf("Model = %q, want %q", got.Model, apiReq.Model)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Role != "system" {
		t.Errorf("Messages[0].Role = %q, want %q", got.Messages[0].Role, "system")
	}
	if got.MaxTokens != 100 {
		t.Errorf("MaxTokens = %d, want 100", got.MaxTokens)
	}
	if got.Stream != true {
		t.Errorf("Stream = %v, want true", got.Stream)
	}
}

func TestCanonicalToAPIRequest(t *testing.T) {
	req := &domain.CanonicalRequest{
		Model:     "gpt-4o-mini",
		MaxTokens: 200,
		Messages: []domain.Message{
			{Role: "user", Content: "Question"},
			{Role: "assistant", Content: "Answer"},
		},
		Stream: true,
	}

	got := CanonicalToAPIRequest(req)

	if got.Model != req.Model {
		t.Errorf("Model = %q, want %q", got.Model, req.Model)
	}
	// CanonicalToAPIRequest uses MaxCompletionTokens (newer field) instead of MaxTokens
	if got.MaxCompletionTokens != req.MaxTokens {
		t.Errorf("MaxCompletionTokens = %d, want %d", got.MaxCompletionTokens, req.MaxTokens)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(got.Messages))
	}
	if got.Stream != req.Stream {
		t.Errorf("Stream = %v, want %v", got.Stream, req.Stream)
	}
}

func TestCanonicalToAPIResponse(t *testing.T) {
	resp := &domain.CanonicalResponse{
		ID:    "resp_test",
		Model: "gpt-4o",
		Choices: []domain.Choice{
			{
				Index:        0,
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
	if len(got.Choices) != 1 {
		t.Fatalf("len(Choices) = %d, want 1", len(got.Choices))
	}
	if got.Choices[0].Message.Content != "Response text" {
		t.Errorf("Content = %q, want %q", got.Choices[0].Message.Content, "Response text")
	}
	if got.Usage.PromptTokens != 50 {
		t.Errorf("PromptTokens = %d, want 50", got.Usage.PromptTokens)
	}
	if got.Usage.CompletionTokens != 25 {
		t.Errorf("CompletionTokens = %d, want 25", got.Usage.CompletionTokens)
	}
}

func TestAPIChunkToCanonical(t *testing.T) {
	chunk := &ChatCompletionChunk{
		ID:     "chatcmpl-123",
		Object: "chat.completion.chunk",
		Choices: []ChunkChoice{
			{
				Index: 0,
				Delta: ChunkDelta{
					Content: "Hello",
				},
			},
		},
	}

	got := APIChunkToCanonical(chunk)

	if got.ContentDelta != "Hello" {
		t.Errorf("ContentDelta = %q, want %q", got.ContentDelta, "Hello")
	}
}

func TestAPIChunkToCanonical_FinishReason(t *testing.T) {
	stopReason := "stop"
	chunk := &ChatCompletionChunk{
		ID:     "chatcmpl-123",
		Object: "chat.completion.chunk",
		Choices: []ChunkChoice{
			{
				Index:        0,
				Delta:        ChunkDelta{},
				FinishReason: &stopReason,
			},
		},
	}

	got := APIChunkToCanonical(chunk)

	if got.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", got.FinishReason, "stop")
	}
}
