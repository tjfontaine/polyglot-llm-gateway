package domain

import "context"

// TokenCountTool represents a tool for token counting purposes.
type TokenCountTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"` // JSON Schema
}

// TokenCountRequest represents a request to count tokens.
type TokenCountRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	System   string           `json:"system,omitempty"`
	Tools    []TokenCountTool `json:"tools,omitempty"`
}

// TokenCountResponse represents the response from counting tokens.
type TokenCountResponse struct {
	InputTokens int    `json:"input_tokens"`
	Model       string `json:"model,omitempty"`
	// Estimated indicates whether the count is an estimate (true) or exact (false)
	Estimated bool `json:"estimated,omitempty"`
}

// TokenCounter provides token counting capabilities.
type TokenCounter interface {
	// CountTokens counts the tokens in the given request.
	// Returns the count and whether it's an estimate.
	CountTokens(ctx context.Context, req *TokenCountRequest) (*TokenCountResponse, error)

	// SupportsModel returns true if this counter supports the given model.
	SupportsModel(model string) bool
}
