// Package shadow provides shadow mode execution for comparing provider responses.
// Shadow mode allows sending requests to alternate providers in parallel with the
// primary provider, capturing and comparing responses to detect divergences.
package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/codec"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Executor manages shadow mode execution for a single request.
// It coordinates sending requests to shadow providers in parallel,
// collecting responses, and storing results.
type Executor struct {
	store          ports.ShadowStore
	providerLookup ProviderLookup
	codecLookup    CodecLookup
	logger         *slog.Logger
}

// ProviderLookup is a function that resolves a provider by name
type ProviderLookup func(name string) (ports.Provider, error)

// CodecLookup is a function that returns a codec for an API type
type CodecLookup func(apiType domain.APIType) (codec.Codec, error)

// ExecutorOption configures an Executor
type ExecutorOption func(*Executor)

// WithLogger sets the logger for the executor
func WithLogger(logger *slog.Logger) ExecutorOption {
	return func(e *Executor) {
		e.logger = logger
	}
}

// NewExecutor creates a new shadow executor
func NewExecutor(
	store ports.ShadowStore,
	providerLookup ProviderLookup,
	codecLookup CodecLookup,
	opts ...ExecutorOption,
) *Executor {
	e := &Executor{
		store:          store,
		providerLookup: providerLookup,
		codecLookup:    codecLookup,
		logger:         slog.Default(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ExecuteRequest contains the request context for shadow execution
type ExecuteRequest struct {
	// InteractionID links shadow results to the primary interaction
	InteractionID string

	// CanonicalRequest is the canonical form of the request
	CanonicalRequest *domain.CanonicalRequest

	// ShadowConfig contains the shadow providers configuration
	ShadowConfig *config.ShadowConfig

	// FrontdoorAPIType is the API type of the frontdoor (for encoding client responses)
	FrontdoorAPIType domain.APIType

	// PrimaryResponse is the canonical response from the primary provider (optional, for divergence detection)
	PrimaryResponse *domain.CanonicalResponse
}

// ExecuteResult contains the results of shadow execution
type ExecuteResult struct {
	// Results contains all shadow execution results
	Results []*domain.ShadowResult

	// Errors contains any errors that occurred during execution
	Errors []error
}

// Execute runs shadow mode execution for all configured shadow providers.
// It sends requests to all providers in parallel and collects results.
// Results are stored in the database and returned.
func (e *Executor) Execute(ctx context.Context, req *ExecuteRequest) *ExecuteResult {
	if req.ShadowConfig == nil || !req.ShadowConfig.Enabled || len(req.ShadowConfig.Providers) == 0 {
		return &ExecuteResult{}
	}

	// Create timeout context if configured
	execCtx := ctx
	if req.ShadowConfig.Timeout != "" {
		timeout, err := time.ParseDuration(req.ShadowConfig.Timeout)
		if err == nil && timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	// Execute all shadow providers in parallel
	var wg sync.WaitGroup
	results := make([]*domain.ShadowResult, len(req.ShadowConfig.Providers))
	errors := make([]error, len(req.ShadowConfig.Providers))

	for i, shadowProvider := range req.ShadowConfig.Providers {
		wg.Add(1)
		go func(idx int, sp config.ShadowProviderConfig) {
			defer wg.Done()
			result, err := e.executeShadowProvider(execCtx, req, sp)
			results[idx] = result
			errors[idx] = err
		}(i, shadowProvider)
	}

	wg.Wait()

	// Collect non-nil results and errors
	result := &ExecuteResult{}
	for i := range results {
		if results[i] != nil {
			// Run divergence detection if we have a primary response
			if req.PrimaryResponse != nil {
				divergences := DetectDivergences(req.PrimaryResponse, results[i])
				results[i].Divergences = divergences
				results[i].HasStructuralDivergence = len(divergences) > 0
			}

			// Store the result
			if err := e.store.SaveShadowResult(ctx, results[i]); err != nil {
				e.logger.Error("failed to save shadow result",
					"shadow_provider", results[i].ProviderName,
					"interaction_id", req.InteractionID,
					"error", err)
				result.Errors = append(result.Errors, fmt.Errorf("failed to save shadow result for %s: %w", results[i].ProviderName, err))
			}

			result.Results = append(result.Results, results[i])
		}
		if errors[i] != nil {
			result.Errors = append(result.Errors, errors[i])
		}
	}

	return result
}

// executeShadowProvider executes a single shadow provider
func (e *Executor) executeShadowProvider(
	ctx context.Context,
	req *ExecuteRequest,
	shadowCfg config.ShadowProviderConfig,
) (*domain.ShadowResult, error) {
	startTime := time.Now()

	// Look up the provider
	provider, err := e.providerLookup(shadowCfg.Name)
	if err != nil {
		return e.createErrorResult(req, shadowCfg, startTime, &domain.InteractionError{
			Type:    "provider_error",
			Code:    "provider_not_found",
			Message: fmt.Sprintf("shadow provider %s not found: %v", shadowCfg.Name, err),
		}), nil
	}

	// Clone the request for the shadow provider
	shadowReq := req.CanonicalRequest.Clone()

	// Apply model override if configured
	if shadowCfg.Model != "" {
		shadowReq.Model = shadowCfg.Model
	}

	// Apply max_tokens_multiplier if configured.
	// Default (nil or 0) = unlimited (important for reasoning models).
	// 1 = use original max_tokens.
	// >1 = multiply original max_tokens (useful for reasoning models that need more headroom).
	if shadowCfg.MaxTokensMultiplier != nil && *shadowCfg.MaxTokensMultiplier > 0 {
		shadowReq.MaxTokens = int(float64(shadowReq.MaxTokens) * *shadowCfg.MaxTokensMultiplier)
	} else {
		shadowReq.MaxTokens = 0
	}

	// Serialize the canonical request
	canonicalRequestJSON, _ := json.Marshal(shadowReq)

	// Get the codec for the provider's API type
	providerCodec, err := e.codecLookup(provider.APIType())
	if err != nil {
		return e.createErrorResult(req, shadowCfg, startTime, &domain.InteractionError{
			Type:    "codec_error",
			Code:    "codec_not_found",
			Message: fmt.Sprintf("codec for %s not found: %v", provider.APIType(), err),
		}), nil
	}

	// Encode the request to provider format
	providerRequestJSON, err := providerCodec.EncodeRequest(shadowReq)
	if err != nil {
		return e.createErrorResult(req, shadowCfg, startTime, &domain.InteractionError{
			Type:    "codec_error",
			Code:    "encode_request_failed",
			Message: fmt.Sprintf("failed to encode request: %v", err),
		}), nil
	}

	// Force non-streaming for shadow to simplify comparison
	shadowReq.Stream = false

	// Execute the request
	resp, err := provider.Complete(ctx, shadowReq)
	duration := time.Since(startTime)

	if err != nil {
		// Convert error to InteractionError
		var interactionErr *domain.InteractionError
		if apiErr, ok := err.(*domain.APIError); ok {
			interactionErr = &domain.InteractionError{
				Type:    string(apiErr.Type),
				Code:    string(apiErr.Code),
				Message: apiErr.Message,
			}
		} else {
			interactionErr = &domain.InteractionError{
				Type:    "provider_error",
				Code:    "request_failed",
				Message: err.Error(),
			}
		}
		return e.createErrorResult(req, shadowCfg, startTime, interactionErr), nil
	}

	// Use the actual raw response from the provider if available,
	// otherwise serialize the canonical response
	var responseRawJSON []byte
	if len(resp.RawResponse) > 0 {
		responseRawJSON = resp.RawResponse
	} else {
		responseRawJSON, _ = json.Marshal(resp)
	}

	// Serialize canonical response
	canonicalRespJSON, _ := json.Marshal(resp)

	// Encode response to client format (frontdoor API type)
	frontdoorCodec, err := e.codecLookup(req.FrontdoorAPIType)
	var clientResponseJSON []byte
	if err == nil {
		clientResponseJSON, _ = frontdoorCodec.EncodeResponse(resp)
	}

	// Build the result
	result := &domain.ShadowResult{
		ID:            uuid.New().String(),
		InteractionID: req.InteractionID,
		ProviderName:  shadowCfg.Name,
		ProviderModel: shadowCfg.Model,
		Request: &domain.ShadowRequest{
			Canonical:       canonicalRequestJSON,
			ProviderRequest: providerRequestJSON,
		},
		Response: &domain.ShadowResponse{
			Raw:            responseRawJSON,
			Canonical:      canonicalRespJSON,
			ClientResponse: clientResponseJSON,
			FinishReason:   getFinishReason(resp),
			Usage:          getUsage(resp),
		},
		Duration:  duration,
		TokensIn:  getTokensIn(resp),
		TokensOut: getTokensOut(resp),
		CreatedAt: time.Now(),
	}

	return result, nil
}

// createErrorResult creates a ShadowResult for an error case
func (e *Executor) createErrorResult(
	req *ExecuteRequest,
	shadowCfg config.ShadowProviderConfig,
	startTime time.Time,
	interactionErr *domain.InteractionError,
) *domain.ShadowResult {
	return &domain.ShadowResult{
		ID:            uuid.New().String(),
		InteractionID: req.InteractionID,
		ProviderName:  shadowCfg.Name,
		ProviderModel: shadowCfg.Model,
		Error:         interactionErr,
		Duration:      time.Since(startTime),
		CreatedAt:     time.Now(),
	}
}

// Helper functions to extract response details
func getFinishReason(resp *domain.CanonicalResponse) string {
	if resp != nil && len(resp.Choices) > 0 {
		return resp.Choices[0].FinishReason
	}
	return ""
}

func getUsage(resp *domain.CanonicalResponse) *domain.Usage {
	if resp != nil {
		return &resp.Usage
	}
	return nil
}

func getTokensIn(resp *domain.CanonicalResponse) int {
	if resp != nil {
		return resp.Usage.PromptTokens
	}
	return 0
}

func getTokensOut(resp *domain.CanonicalResponse) int {
	if resp != nil {
		return resp.Usage.CompletionTokens
	}
	return 0
}
