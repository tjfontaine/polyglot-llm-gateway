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
			[]string{"gpt-4", "gpt-3.5", "o1", "o3", "text-embedding", "text-davinci"},
			[]string{"davinci", "curie", "babbage", "ada"},
		),
		encodingCache: make(map[string]*tiktoken.Tiktoken),
	}
}

// getEncoding returns the tiktoken encoding for a model, with caching.
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

	// Different models have different message overhead
	// GPT-4 and GPT-3.5-turbo use the chat format
	tokensPerMessage := 3 // <|start|>{role/name}\n{content}<|end|>\n
	tokensPerName := 1    // if there's a name, the role is omitted

	// Count system message
	if req.System != "" {
		totalTokens += tokensPerMessage
		totalTokens += len(enc.Encode(req.System, nil, nil))
		totalTokens += len(enc.Encode("system", nil, nil))
	}

	// Count all messages
	for _, msg := range req.Messages {
		totalTokens += tokensPerMessage

		// Count role
		totalTokens += len(enc.Encode(msg.Role, nil, nil))

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
			totalTokens += tokensPerName + 3 // overhead per tool call
		}
	}

	// Count tools/functions
	for _, tool := range req.Tools {
		// Tool definitions are serialized as JSON in the prompt
		totalTokens += len(enc.Encode(tool.Name, nil, nil))
		totalTokens += len(enc.Encode(tool.Description, nil, nil))
		if tool.Parameters != nil {
			paramBytes, _ := json.Marshal(tool.Parameters)
			totalTokens += len(enc.Encode(string(paramBytes), nil, nil))
		}
		totalTokens += 7 // overhead per tool definition
	}

	// Add priming tokens for assistant response
	totalTokens += 3 // <|start|>assistant<|message|>

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

// CountString counts tokens in a simple string for a given model.
// Useful for quick token counting without building a full request.
func (c *OpenAICounter) CountString(model, text string) (int, error) {
	enc, err := c.getEncoding(model)
	if err != nil {
		return 0, err
	}
	return len(enc.Encode(text, nil, nil)), nil
}

// GetModelEncoding returns the encoding name for a model.
func GetModelEncoding(model string) string {
	model = strings.ToLower(model)

	// GPT-4 and GPT-3.5-turbo models use cl100k_base
	if strings.HasPrefix(model, "gpt-4") ||
		strings.HasPrefix(model, "gpt-3.5") ||
		strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "text-embedding-3") ||
		strings.HasPrefix(model, "text-embedding-ada-002") {
		return "cl100k_base"
	}

	// Older models use p50k_base or r50k_base
	if strings.HasPrefix(model, "text-davinci") ||
		strings.HasPrefix(model, "code-davinci") {
		return "p50k_base"
	}

	if strings.HasPrefix(model, "davinci") ||
		strings.HasPrefix(model, "curie") ||
		strings.HasPrefix(model, "babbage") ||
		strings.HasPrefix(model, "ada") {
		return "r50k_base"
	}

	// Default to cl100k_base for unknown models
	return "cl100k_base"
}
