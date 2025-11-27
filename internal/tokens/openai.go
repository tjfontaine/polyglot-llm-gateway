package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// OpenAICounter provides accurate token counts for OpenAI models using tiktoken.
type OpenAICounter struct {
	matcher *ModelMatcher
	// encodingCache caches tiktoken encodings by model
	encodingCache map[string]*tiktoken.Tiktoken
}

// NewOpenAICounter creates a new OpenAI token counter.
func NewOpenAICounter() *OpenAICounter {
	return &OpenAICounter{
		matcher: NewModelMatcher(
			// Prefixes for OpenAI models (including future gpt-5, gpt-6, etc.)
			[]string{"gpt-", "o1", "o3", "o4", "text-embedding", "text-davinci"},
			// Exact matches for legacy models
			[]string{"davinci", "curie", "babbage", "ada"},
		),
		encodingCache: make(map[string]*tiktoken.Tiktoken),
	}
}

// getEncoding returns the tiktoken encoding for a model.
func (c *OpenAICounter) getEncoding(model string) (*tiktoken.Tiktoken, error) {
	// Check cache first
	if enc, ok := c.encodingCache[model]; ok {
		return enc, nil
	}

	// Try to get encoding for the specific model
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		// Fall back to cl100k_base for newer models not yet in tiktoken
		enc, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, fmt.Errorf("failed to get tiktoken encoding: %w", err)
		}
	}

	c.encodingCache[model] = enc
	return enc, nil
}

// CountTokens counts tokens for OpenAI models using tiktoken.
func (c *OpenAICounter) CountTokens(ctx context.Context, req *domain.TokenCountRequest) (*domain.TokenCountResponse, error) {
	enc, err := c.getEncoding(req.Model)
	if err != nil {
		return nil, err
	}

	totalTokens := 0

	// Token overhead per message for chat models
	// Based on OpenAI's documentation:
	// - gpt-4, gpt-3.5-turbo: 3 tokens per message + 1 for role
	// - Plus 3 tokens for assistant priming at the end
	tokensPerMessage := 3
	tokensPerRole := 1

	// Count system message
	if req.System != "" {
		totalTokens += tokensPerMessage
		totalTokens += tokensPerRole
		totalTokens += len(enc.Encode(req.System, nil, nil))
	}

	// Count all messages
	for _, msg := range req.Messages {
		totalTokens += tokensPerMessage

		// Count role
		totalTokens += tokensPerRole

		// Count content
		if msg.RichContent != nil && len(msg.RichContent.Parts) > 0 {
			for _, part := range msg.RichContent.Parts {
				switch part.Type {
				case domain.ContentTypeText:
					totalTokens += len(enc.Encode(part.Text, nil, nil))
				case domain.ContentTypeToolUse:
					totalTokens += len(enc.Encode(part.Name, nil, nil))
					if part.Input != nil {
						argBytes, _ := json.Marshal(part.Input)
						totalTokens += len(enc.Encode(string(argBytes), nil, nil))
					}
					totalTokens += 3 // overhead for tool call structure
				case domain.ContentTypeToolResult:
					totalTokens += len(enc.Encode(part.Text, nil, nil))
					totalTokens += 2 // overhead for tool result
				}
			}
		} else {
			totalTokens += len(enc.Encode(msg.Content, nil, nil))
		}

		// Count tool calls if present
		for _, tc := range msg.ToolCalls {
			totalTokens += len(enc.Encode(tc.Function.Name, nil, nil))
			totalTokens += len(enc.Encode(tc.Function.Arguments, nil, nil))
			totalTokens += 3 // overhead per tool call
		}
	}

	// Count tools/functions
	for _, tool := range req.Tools {
		totalTokens += len(enc.Encode(tool.Name, nil, nil))
		totalTokens += len(enc.Encode(tool.Description, nil, nil))
		if tool.Parameters != nil {
			paramBytes, _ := json.Marshal(tool.Parameters)
			totalTokens += len(enc.Encode(string(paramBytes), nil, nil))
		}
		totalTokens += 7 // overhead per tool definition
	}

	// Add final assistant prompt tokens
	totalTokens += 3 // assistant priming

	return &domain.TokenCountResponse{
		InputTokens: totalTokens,
		Model:       req.Model,
		Estimated:   false, // tiktoken provides accurate counts
	}, nil
}

// SupportsModel returns true for OpenAI models.
func (c *OpenAICounter) SupportsModel(model string) bool {
	return c.matcher.Matches(model)
}

// CountText counts tokens for a plain text string.
func (c *OpenAICounter) CountText(model, text string) (int, error) {
	enc, err := c.getEncoding(model)
	if err != nil {
		return 0, err
	}
	return len(enc.Encode(text, nil, nil)), nil
}

// modelToEncoding maps model names to encoding names for tiktoken.
// This helps handle models that tiktoken doesn't recognize directly.
// 
// Encoding reference:
// - cl100k_base: GPT-4, GPT-3.5-turbo, text-embedding-ada-002, and newer models
// - p50k_base: text-davinci-003, text-davinci-002
// - r50k_base: davinci, curie, babbage, ada (legacy)
// - o200k_base: GPT-4o and potentially future models (if different from cl100k)
func modelToEncoding(model string) string {
	switch {
	// GPT-4o models might use o200k_base (newer encoding)
	case strings.HasPrefix(model, "gpt-4o"):
		return "cl100k_base" // tiktoken-go may not have o200k_base yet, fallback to cl100k
	// Future GPT models (gpt-5, gpt-6, etc.) - assume cl100k_base until tiktoken updates
	case strings.HasPrefix(model, "gpt-"):
		return "cl100k_base"
	// Reasoning models (o1, o3, o4, etc.)
	case strings.HasPrefix(model, "o1"), strings.HasPrefix(model, "o3"), strings.HasPrefix(model, "o4"):
		return "cl100k_base"
	// Embedding models
	case strings.HasPrefix(model, "text-embedding"):
		return "cl100k_base"
	// Legacy text-davinci models
	case strings.HasPrefix(model, "text-davinci-003"), strings.HasPrefix(model, "text-davinci-002"):
		return "p50k_base"
	case strings.HasPrefix(model, "text-davinci"):
		return "p50k_base"
	// Legacy completion models
	case model == "davinci" || model == "curie" || model == "babbage" || model == "ada":
		return "r50k_base"
	default:
		// Default to cl100k_base for unknown/future models
		return "cl100k_base"
	}
}
