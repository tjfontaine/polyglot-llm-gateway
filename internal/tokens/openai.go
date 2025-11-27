package tokens

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// OpenAICounter estimates token counts for OpenAI models.
// Uses tiktoken-style cl100k_base encoding estimation.
type OpenAICounter struct {
	matcher *ModelMatcher
}

// NewOpenAICounter creates a new OpenAI token counter.
func NewOpenAICounter() *OpenAICounter {
	return &OpenAICounter{
		matcher: NewModelMatcher(
			[]string{"gpt-4", "gpt-3.5", "o1", "o3", "text-embedding", "text-davinci"},
			[]string{"davinci", "curie", "babbage", "ada"},
		),
	}
}

// CountTokens estimates tokens for OpenAI models using cl100k_base-style tokenization.
func (c *OpenAICounter) CountTokens(ctx context.Context, req *domain.TokenCountRequest) (*domain.TokenCountResponse, error) {
	totalTokens := 0

	// Add base overhead for chat format
	// Each message has ~4 tokens of overhead: <|start|>{role}<|end|>
	messageOverhead := 4

	// Count system message
	if req.System != "" {
		totalTokens += messageOverhead
		totalTokens += c.estimateTokens(req.System)
	}

	// Count all messages
	for _, msg := range req.Messages {
		totalTokens += messageOverhead

		// Count role (usually 1 token)
		totalTokens += 1

		// Count content
		if msg.RichContent != nil && len(msg.RichContent.Parts) > 0 {
			for _, part := range msg.RichContent.Parts {
				switch part.Type {
				case domain.ContentTypeText:
					totalTokens += c.estimateTokens(part.Text)
				case domain.ContentTypeToolUse:
					// Tool calls add structured overhead
					totalTokens += c.estimateTokens(part.Name)
					if part.Input != nil {
						argBytes, _ := json.Marshal(part.Input)
						totalTokens += c.estimateTokens(string(argBytes))
					}
					totalTokens += 3 // overhead for tool call structure
				case domain.ContentTypeToolResult:
					totalTokens += c.estimateTokens(part.Text)
					totalTokens += 2 // overhead for tool result
				}
			}
		} else {
			totalTokens += c.estimateTokens(msg.Content)
		}

		// Count tool calls if present
		for _, tc := range msg.ToolCalls {
			totalTokens += c.estimateTokens(tc.Function.Name)
			totalTokens += c.estimateTokens(tc.Function.Arguments)
			totalTokens += 3 // overhead per tool call
		}
	}

	// Count tools/functions (using TokenCountTool)
	for _, tool := range req.Tools {
		totalTokens += c.estimateTokens(tool.Name)
		totalTokens += c.estimateTokens(tool.Description)
		if tool.Parameters != nil {
			paramBytes, _ := json.Marshal(tool.Parameters)
			totalTokens += c.estimateTokens(string(paramBytes))
		}
		totalTokens += 7 // overhead per tool definition
	}

	// Add final assistant prompt tokens
	totalTokens += 3 // <|start|>assistant<|message|>

	return &domain.TokenCountResponse{
		InputTokens: totalTokens,
		Model:       req.Model,
		Estimated:   true, // We're estimating, not using tiktoken directly
	}, nil
}

// estimateTokens estimates the token count for a string using cl100k_base-style rules.
// This approximates GPT-4/GPT-3.5-turbo tokenization.
func (c *OpenAICounter) estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	tokens := 0

	// Split by whitespace and common patterns
	// cl100k_base tends to:
	// - Keep common words as single tokens
	// - Split uncommon words into subwords
	// - Treat punctuation separately in many cases
	// - Handle numbers digit-by-digit in some cases

	// Basic word splitting with punctuation awareness
	wordPattern := regexp.MustCompile(`[\w']+|[^\w\s]|\s+`)
	parts := wordPattern.FindAllString(text, -1)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Whitespace is often merged with the following token
		if strings.TrimSpace(part) == "" {
			continue
		}

		// Single punctuation is usually 1 token
		if len(part) == 1 && !isAlphanumeric(part[0]) {
			tokens++
			continue
		}

		// Estimate based on word characteristics
		tokens += c.estimateWordTokens(part)
	}

	return tokens
}

// estimateWordTokens estimates tokens for a single word/segment.
func (c *OpenAICounter) estimateWordTokens(word string) int {
	if word == "" {
		return 0
	}

	// Very short words are usually 1 token
	runeCount := utf8.RuneCountInString(word)
	if runeCount <= 3 {
		return 1
	}

	// Common English words are often 1 token
	if c.isCommonWord(strings.ToLower(word)) {
		return 1
	}

	// Numbers: roughly 1 token per 2-3 digits
	if isNumeric(word) {
		return (len(word) + 2) / 3
	}

	// For other words, estimate based on character count
	// cl100k_base averages about 3.5-4 chars per token for English
	// but varies with word complexity

	// Check for camelCase or mixed case (often split)
	if hasMixedCase(word) {
		// Split on case boundaries roughly
		return (runeCount + 3) / 4
	}

	// Standard estimation
	// Short-medium words: 1 token
	// Longer words: split into subwords
	if runeCount <= 6 {
		return 1
	} else if runeCount <= 10 {
		return 2
	} else if runeCount <= 15 {
		return 3
	}

	// Very long words: roughly 4 chars per token
	return (runeCount + 3) / 4
}

// isCommonWord checks if a word is in a list of common English words
// that are typically single tokens in cl100k_base.
func (c *OpenAICounter) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "been": true, "be": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"that": true, "which": true, "who": true, "whom": true, "this": true, "these": true,
		"those": true, "it": true, "its": true, "they": true, "them": true, "their": true,
		"he": true, "him": true, "his": true, "she": true, "her": true, "hers": true,
		"we": true, "us": true, "our": true, "you": true, "your": true, "yours": true,
		"i": true, "me": true, "my": true, "mine": true, "not": true, "no": true, "yes": true,
		"can": true, "cannot": true, "if": true, "then": true, "else": true, "when": true,
		"where": true, "why": true, "how": true, "what": true, "all": true, "each": true,
		"every": true, "both": true, "few": true, "more": true, "most": true, "other": true,
		"some": true, "such": true, "only": true, "own": true, "same": true, "so": true,
		"than": true, "too": true, "very": true, "just": true, "also": true, "now": true,
		"here": true, "there": true, "about": true, "after": true, "before": true, "between": true,
		"into": true, "through": true, "during": true, "under": true, "again": true, "further": true,
		"once": true, "user": true, "assistant": true, "system": true, "function": true,
		"message": true, "content": true, "role": true, "name": true, "type": true,
	}
	return commonWords[word]
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func hasMixedCase(s string) bool {
	hasUpper := false
	hasLower := false
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
		if hasUpper && hasLower {
			return true
		}
	}
	return false
}

// SupportsModel returns true for OpenAI models.
func (c *OpenAICounter) SupportsModel(model string) bool {
	return c.matcher.Matches(model)
}
