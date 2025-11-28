// Package tokens provides token counting capabilities across different LLM providers.
package tokens

import (
	"context"
	"fmt"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// Registry manages token counters for different providers/models.
// It supports:
// 1. Providers that implement domain.TokenCountProvider for native token counting
// 2. Registered domain.TokenCounter implementations (like tiktoken for OpenAI)
// 3. A fallback estimator for unknown models
type Registry struct {
	counters []domain.TokenCounter
	fallback domain.TokenCounter
	provider domain.Provider // Optional provider to check for TokenCountProvider interface
}

// NewRegistry creates a new token counter registry.
func NewRegistry() *Registry {
	return &Registry{
		fallback: NewEstimator(), // Default fallback estimator
	}
}

// Register adds a token counter to the registry.
func (r *Registry) Register(counter domain.TokenCounter) {
	r.counters = append(r.counters, counter)
}

// SetFallback sets the fallback counter for unsupported models.
func (r *Registry) SetFallback(counter domain.TokenCounter) {
	r.fallback = counter
}

// SetProvider sets a provider that may implement TokenCountProvider.
// If the provider implements the interface and supports the model,
// it will be used instead of the registered counters.
func (r *Registry) SetProvider(provider domain.Provider) {
	r.provider = provider
}

// CountTokens counts tokens using the appropriate counter for the model.
// Priority order:
// 1. If a provider is set and implements TokenCountProvider, use it if it supports the model
// 2. Use registered counters that support the model
// 3. Use the fallback estimator
func (r *Registry) CountTokens(ctx context.Context, req *domain.TokenCountRequest) (*domain.TokenCountResponse, error) {
	// Check if the provider implements TokenCountProvider
	if r.provider != nil {
		if tcp, ok := r.provider.(domain.TokenCountProvider); ok {
			if tcp.SupportsTokenCounting(req.Model) {
				return tcp.CountTokensCanonical(ctx, req)
			}
		}
	}

	// Find a registered counter that supports this model
	for _, counter := range r.counters {
		if counter.SupportsModel(req.Model) {
			return counter.CountTokens(ctx, req)
		}
	}

	// Use fallback
	if r.fallback != nil {
		return r.fallback.CountTokens(ctx, req)
	}

	return nil, fmt.Errorf("no token counter available for model: %s", req.Model)
}

// GetCounter returns the appropriate counter for a model.
func (r *Registry) GetCounter(model string) domain.TokenCounter {
	for _, counter := range r.counters {
		if counter.SupportsModel(model) {
			return counter
		}
	}
	return r.fallback
}

// Estimator provides token count estimation based on character/word analysis.
// This is a fallback for providers without native token counting.
type Estimator struct {
	// CharsPerToken is the average characters per token (default: 4)
	CharsPerToken float64
}

// NewEstimator creates a new token estimator.
func NewEstimator() *Estimator {
	return &Estimator{
		CharsPerToken: 4.0, // Reasonable default for most models
	}
}

// CountTokens estimates the token count.
func (e *Estimator) CountTokens(ctx context.Context, req *domain.TokenCountRequest) (*domain.TokenCountResponse, error) {
	totalChars := 0

	// Count system message
	if req.System != "" {
		totalChars += len(req.System)
	}

	// Count all messages
	for _, msg := range req.Messages {
		totalChars += len(msg.Role)
		totalChars += len(msg.Content)
		// Add overhead for message formatting (approximately)
		totalChars += 4 // role tokens + separators
	}

	// Count tools (rough estimate)
	for _, tool := range req.Tools {
		totalChars += len(tool.Name)
		totalChars += len(tool.Description)
		// Tool schema adds overhead
		totalChars += 50 // rough estimate for schema
	}

	tokens := int(float64(totalChars) / e.CharsPerToken)

	return &domain.TokenCountResponse{
		InputTokens: tokens,
		Model:       req.Model,
		Estimated:   true,
	}, nil
}

// SupportsModel returns true - estimator supports all models as a fallback.
func (e *Estimator) SupportsModel(model string) bool {
	return true
}

// ModelMatcher helps match model names to provider patterns.
type ModelMatcher struct {
	prefixes []string
	exact    []string
}

// NewModelMatcher creates a new model matcher.
func NewModelMatcher(prefixes, exact []string) *ModelMatcher {
	return &ModelMatcher{
		prefixes: prefixes,
		exact:    exact,
	}
}

// Matches returns true if the model matches any pattern.
func (m *ModelMatcher) Matches(model string) bool {
	// Check exact matches first
	for _, e := range m.exact {
		if model == e {
			return true
		}
	}

	// Check prefix matches
	for _, p := range m.prefixes {
		if strings.HasPrefix(model, p) {
			return true
		}
	}

	return false
}
