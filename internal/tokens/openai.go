package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/tiktoken-go/tokenizer"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// OpenAICounter provides accurate token counts for OpenAI models using tiktoken.
type OpenAICounter struct {
	matcher *ModelMatcher
	// codecCache caches tokenizer codecs by encoding name
	codecCache map[tokenizer.Encoding]tokenizer.Codec
	cacheMu    sync.RWMutex
}

// NewOpenAICounter creates a new OpenAI token counter.
func NewOpenAICounter() *OpenAICounter {
	return &OpenAICounter{
		matcher: NewModelMatcher(
			// Prefixes for OpenAI models (including future gpt-5.x, gpt-6, etc.)
			// Note: "o" prefix matches o1, o3, o4, o5, etc. reasoning models
			[]string{"gpt-", "o1", "o2", "o3", "o4", "o5", "o6", "text-embedding", "text-davinci"},
			// Exact matches for legacy models
			[]string{"davinci", "curie", "babbage", "ada"},
		),
		codecCache: make(map[tokenizer.Encoding]tokenizer.Codec),
	}
}

// getCodec returns the tokenizer codec for a model.
func (c *OpenAICounter) getCodec(model string) (tokenizer.Codec, error) {
	// Map model name to tokenizer.Model
	tmodel := mapModelName(model)

	// Try to get codec for the specific model
	codec, err := tokenizer.ForModel(tmodel)
	if err == nil {
		return codec, nil
	}

	// Fall back to encoding based on model prefix
	encoding := modelToEncoding(model)

	// Check cache
	c.cacheMu.RLock()
	if cached, ok := c.codecCache[encoding]; ok {
		c.cacheMu.RUnlock()
		return cached, nil
	}
	c.cacheMu.RUnlock()

	// Get encoding
	codec, err = tokenizer.Get(encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer encoding: %w", err)
	}

	// Cache it
	c.cacheMu.Lock()
	c.codecCache[encoding] = codec
	c.cacheMu.Unlock()

	return codec, nil
}

// mapModelName maps a model string to tokenizer.Model
func mapModelName(model string) tokenizer.Model {
	// Normalize model name
	model = strings.ToLower(model)

	// Direct mappings for known models
	switch {
	// GPT-5 family (exact matches for known variants)
	case model == "gpt-5":
		return tokenizer.GPT5
	case model == "gpt-5-mini" || strings.HasPrefix(model, "gpt-5-mini-"):
		return tokenizer.GPT5Mini
	case model == "gpt-5-nano" || strings.HasPrefix(model, "gpt-5-nano-"):
		return tokenizer.GPT5Nano
	// GPT-5.x and other gpt-5 variants (gpt-5-turbo, gpt-5.1, etc.) use GPT5 encoding
	case strings.HasPrefix(model, "gpt-5"):
		return tokenizer.GPT5

	// GPT-4.1 family
	case strings.HasPrefix(model, "gpt-4.1") || strings.HasPrefix(model, "gpt-41"):
		return tokenizer.GPT41

	// GPT-4o family
	case strings.HasPrefix(model, "gpt-4o"):
		return tokenizer.GPT4o

	// O-series reasoning models
	case model == "o1" || model == "o1-preview" || strings.HasPrefix(model, "o1-"):
		if strings.Contains(model, "mini") {
			return tokenizer.O1Mini
		}
		if strings.Contains(model, "preview") {
			return tokenizer.O1Preview
		}
		return tokenizer.O1
	case model == "o3" || strings.HasPrefix(model, "o3-"):
		if strings.Contains(model, "mini") {
			return tokenizer.O3Mini
		}
		return tokenizer.O3
	case model == "o4-mini" || strings.HasPrefix(model, "o4-mini"):
		return tokenizer.O4Mini
	// Future O-series models (o4, o5, o6, etc.) - use O4Mini as closest match
	case strings.HasPrefix(model, "o4"), strings.HasPrefix(model, "o5"), strings.HasPrefix(model, "o6"):
		return tokenizer.O4Mini

	// GPT-4 family
	case strings.HasPrefix(model, "gpt-4"):
		return tokenizer.GPT4

	// GPT-3.5 family
	case strings.HasPrefix(model, "gpt-3.5"):
		return tokenizer.GPT35Turbo

	// Future GPT models (gpt-6+) - use GPT5 encoding (o200k_base)
	case strings.HasPrefix(model, "gpt-6"), strings.HasPrefix(model, "gpt-7"):
		return tokenizer.GPT5

	// Text embedding
	case strings.HasPrefix(model, "text-embedding"):
		return tokenizer.TextEmbeddingAda002

	// Legacy models
	case strings.HasPrefix(model, "text-davinci-003"):
		return tokenizer.TextDavinci003
	case strings.HasPrefix(model, "text-davinci-002"):
		return tokenizer.TextDavinci002
	case strings.HasPrefix(model, "text-davinci"):
		return tokenizer.TextDavinci001
	case model == "davinci":
		return tokenizer.Davinci
	case model == "curie":
		return tokenizer.Curie
	case model == "babbage":
		return tokenizer.Babbage
	case model == "ada":
		return tokenizer.Ada

	default:
		// Return as Model type - tokenizer.ForModel will handle unknown models
		return tokenizer.Model(model)
	}
}

// modelToEncoding maps model names to encoding names for fallback.
//
// Encoding reference:
// - O200kBase: GPT-5, GPT-4.1, GPT-4o, O1, O3, O4-mini and newer models
// - Cl100kBase: GPT-4, GPT-3.5-turbo, text-embedding-ada-002
// - P50kBase: text-davinci-003, text-davinci-002
// - R50kBase: davinci, curie, babbage, ada (legacy)
func modelToEncoding(model string) tokenizer.Encoding {
	model = strings.ToLower(model)

	switch {
	// Newer models use O200k_base
	case strings.HasPrefix(model, "gpt-5"):
		return tokenizer.O200kBase
	case strings.HasPrefix(model, "gpt-4.1"), strings.HasPrefix(model, "gpt-41"):
		return tokenizer.O200kBase
	case strings.HasPrefix(model, "gpt-4o"):
		return tokenizer.O200kBase
	case strings.HasPrefix(model, "o1"), strings.HasPrefix(model, "o3"), strings.HasPrefix(model, "o4"):
		return tokenizer.O200kBase

	// GPT-4 and GPT-3.5 use cl100k_base
	case strings.HasPrefix(model, "gpt-4"):
		return tokenizer.Cl100kBase
	case strings.HasPrefix(model, "gpt-3.5"):
		return tokenizer.Cl100kBase

	// Embedding models
	case strings.HasPrefix(model, "text-embedding"):
		return tokenizer.Cl100kBase

	// Legacy text-davinci models
	case strings.HasPrefix(model, "text-davinci-003"), strings.HasPrefix(model, "text-davinci-002"):
		return tokenizer.P50kBase
	case strings.HasPrefix(model, "text-davinci"):
		return tokenizer.P50kBase

	// Legacy completion models
	case model == "davinci" || model == "curie" || model == "babbage" || model == "ada":
		return tokenizer.R50kBase

	default:
		// Default to O200k_base for unknown/future models (most likely encoding)
		return tokenizer.O200kBase
	}
}

// CountTokens counts tokens for OpenAI models using tiktoken.
func (c *OpenAICounter) CountTokens(ctx context.Context, req *domain.TokenCountRequest) (*domain.TokenCountResponse, error) {
	codec, err := c.getCodec(req.Model)
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
		ids, _, _ := codec.Encode(req.System)
		totalTokens += len(ids)
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
					ids, _, _ := codec.Encode(part.Text)
					totalTokens += len(ids)
				case domain.ContentTypeToolUse:
					ids, _, _ := codec.Encode(part.Name)
					totalTokens += len(ids)
					if part.Input != nil {
						argBytes, _ := json.Marshal(part.Input)
						ids, _, _ := codec.Encode(string(argBytes))
						totalTokens += len(ids)
					}
					totalTokens += 3 // overhead for tool call structure
				case domain.ContentTypeToolResult:
					ids, _, _ := codec.Encode(part.Text)
					totalTokens += len(ids)
					totalTokens += 2 // overhead for tool result
				}
			}
		} else {
			ids, _, _ := codec.Encode(msg.Content)
			totalTokens += len(ids)
		}

		// Count tool calls if present
		for _, tc := range msg.ToolCalls {
			ids, _, _ := codec.Encode(tc.Function.Name)
			totalTokens += len(ids)
			ids, _, _ = codec.Encode(tc.Function.Arguments)
			totalTokens += len(ids)
			totalTokens += 3 // overhead per tool call
		}
	}

	// Count tools/functions
	for _, tool := range req.Tools {
		ids, _, _ := codec.Encode(tool.Name)
		totalTokens += len(ids)
		ids, _, _ = codec.Encode(tool.Description)
		totalTokens += len(ids)
		if tool.Parameters != nil {
			paramBytes, _ := json.Marshal(tool.Parameters)
			ids, _, _ := codec.Encode(string(paramBytes))
			totalTokens += len(ids)
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
	codec, err := c.getCodec(model)
	if err != nil {
		return 0, err
	}
	ids, _, err := codec.Encode(text)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}
