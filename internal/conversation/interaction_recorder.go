package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/middleware"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// RecordInteractionParams contains parameters for recording an interaction
type RecordInteractionParams struct {
	// Store is the interaction store to use
	Store storage.ConversationStore

	// Request information
	RawRequest      json.RawMessage
	CanonicalReq    *domain.CanonicalRequest
	UnmappedRequest []string

	// Response information (populated after provider call)
	RawResponse         json.RawMessage
	CanonicalResp       *domain.CanonicalResponse
	UnmappedResponse    []string
	ClientResponse      json.RawMessage
	ProviderRequestBody json.RawMessage

	// Headers from incoming request
	RequestHeaders http.Header

	// Frontdoor and provider info
	Frontdoor    domain.APIType
	Provider     string
	AppName      string
	Streaming    bool
	Error        error
	Duration     time.Duration
	FinishReason string

	// Responses API threading
	PreviousInteractionID string
	ThreadKey             string

	// ClientStorePreference captures the client's original store preference (for auditing)
	ClientStorePreference *bool
}

// RecordInteraction stores a full interaction record with bidirectional visibility.
// This captures the raw request, canonical mapping, what was sent to the provider,
// and the response with all mappings for full observability.
func RecordInteraction(ctx context.Context, params RecordInteractionParams) string {
	store, ok := params.Store.(storage.InteractionStore)
	if !ok || store == nil {
		// Fall back to legacy conversation recording if store doesn't support interactions
		return recordLegacy(ctx, params)
	}

	logger := slog.Default()

	// Decouple persistence from the request lifecycle
	persistCtx, cancel := buildPersistenceContext(ctx, 5*time.Second)
	defer cancel()

	// Always use gateway-owned ID as primary key
	// Check if frontdoor already assigned an ID via request metadata
	interactionID := ""
	if params.CanonicalReq != nil && params.CanonicalReq.Metadata != nil {
		if existingID := params.CanonicalReq.Metadata["interaction_id"]; existingID != "" {
			interactionID = existingID
		}
	}
	if interactionID == "" {
		interactionID = "int_" + uuid.New().String()
	}

	tenantID := tenantIDFromContext(persistCtx)

	// Build interaction record
	interaction := domain.NewInteraction(interactionID, tenantID)
	interaction.Frontdoor = params.Frontdoor
	interaction.Provider = params.Provider
	interaction.AppName = params.AppName
	interaction.Streaming = params.Streaming
	interaction.Duration = params.Duration
	interaction.PreviousInteractionID = params.PreviousInteractionID
	interaction.ThreadKey = params.ThreadKey

	if params.CanonicalReq != nil {
		interaction.RequestedModel = params.CanonicalReq.Model
	}

	if params.CanonicalResp != nil {
		interaction.ServedModel = params.CanonicalResp.Model
		interaction.ProviderModel = params.CanonicalResp.ProviderModel
		// Store provider's response ID as metadata (NOT as our primary key)
		if params.CanonicalResp.ID != "" {
			interaction.Metadata["provider_response_id"] = params.CanonicalResp.ID
		}
	}

	// Add request ID and other metadata
	if reqID, ok := persistCtx.Value(middleware.RequestIDKey).(string); ok && reqID != "" {
		interaction.Metadata["request_id"] = reqID
	}

	// Record client's store preference for auditing
	if params.ClientStorePreference != nil {
		if *params.ClientStorePreference {
			interaction.Metadata["client_store_preference"] = "true"
		} else {
			interaction.Metadata["client_store_preference"] = "false"
		}
	}

	// Capture relevant request headers
	if params.RequestHeaders != nil {
		interaction.RequestHeaders = extractRelevantHeaders(params.RequestHeaders)
	}

	// Build request record with full visibility
	interaction.Request = &domain.InteractionRequest{
		Raw:            params.RawRequest,
		UnmappedFields: params.UnmappedRequest,
	}

	if params.CanonicalReq != nil {
		canonicalJSON, err := json.Marshal(params.CanonicalReq)
		if err == nil {
			interaction.Request.CanonicalJSON = canonicalJSON
		}
	}

	if len(params.ProviderRequestBody) > 0 {
		interaction.Request.ProviderRequest = params.ProviderRequestBody
	}

	// Build response record
	if params.CanonicalResp != nil || len(params.RawResponse) > 0 || params.Error != nil {
		interaction.Response = &domain.InteractionResponse{
			Raw:            params.RawResponse,
			UnmappedFields: params.UnmappedResponse,
			ClientResponse: params.ClientResponse,
			FinishReason:   params.FinishReason,
		}

		if params.CanonicalResp != nil {
			canonicalJSON, err := json.Marshal(params.CanonicalResp)
			if err == nil {
				interaction.Response.CanonicalJSON = canonicalJSON
			}
			interaction.Response.Usage = &params.CanonicalResp.Usage

			if len(params.CanonicalResp.Choices) > 0 {
				interaction.Response.FinishReason = params.CanonicalResp.Choices[0].FinishReason
			}

			// Store provider's response ID in the response record
			if params.CanonicalResp.ID != "" {
				interaction.Response.ProviderResponseID = params.CanonicalResp.ID
			}
		}
	}

	// Build transformation steps for debugging
	var transformationSteps []domain.TransformationStep

	// Step 1: Decode request from frontdoor format to canonical
	if params.CanonicalReq != nil {
		warnings := buildUnmappedWarnings(params.UnmappedRequest)
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "decode_request",
			Timestamp:   interaction.CreatedAt,
			Codec:       string(params.Frontdoor),
			Description: fmt.Sprintf("Decoded %s request to canonical format", params.Frontdoor),
			Details: map[string]interface{}{
				"unmapped_fields_count": len(params.UnmappedRequest),
			},
			Warnings: warnings,
		})
	}

	// Step 2: Model mapping and provider selection (if different from requested)
	if interaction.ServedModel != "" && interaction.ServedModel != interaction.RequestedModel {
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "model_mapping",
			Timestamp:   interaction.CreatedAt,
			Description: "Mapped model and selected provider",
			Details: map[string]interface{}{
				"original_model":  interaction.RequestedModel,
				"mapped_model":    interaction.ServedModel,
				"provider_chosen": params.Provider,
			},
		})
	}

	// Step 3: Encode request for provider
	if len(params.ProviderRequestBody) > 0 {
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "encode_provider_request",
			Timestamp:   interaction.CreatedAt,
			Description: fmt.Sprintf("Encoded canonical request to %s format", params.Provider),
			Details: map[string]interface{}{
				"provider": params.Provider,
			},
		})
	}

	// Step 4: Decode provider response to canonical
	if params.CanonicalResp != nil {
		warnings := buildUnmappedWarnings(params.UnmappedResponse)
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "decode_provider_response",
			Timestamp:   interaction.CreatedAt,
			Description: fmt.Sprintf("Decoded %s response to canonical format", params.Provider),
			Details: map[string]interface{}{
				"unmapped_fields_count": len(params.UnmappedResponse),
			},
			Warnings: warnings,
		})
	}

	// Step 5: Encode response for client
	if len(params.ClientResponse) > 0 {
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "encode_client_response",
			Timestamp:   interaction.CreatedAt,
			Codec:       string(params.Frontdoor),
			Description: fmt.Sprintf("Encoded canonical response to %s format", params.Frontdoor),
		})
	}

	interaction.TransformationSteps = transformationSteps

	// Set status based on outcome
	if params.Error != nil {
		interaction.Status = domain.InteractionStatusFailed
		interaction.Error = &domain.InteractionError{
			Type:    "error",
			Message: params.Error.Error(),
		}
		// Try to extract more specific error info
		if apiErr, ok := params.Error.(*domain.APIError); ok {
			interaction.Error.Type = string(apiErr.Type)
			interaction.Error.Code = string(apiErr.Code)
		}
	} else {
		interaction.Status = domain.InteractionStatusCompleted
	}

	// Persist the interaction
	if err := store.SaveInteraction(persistCtx, interaction); err != nil {
		logger.Error("failed to save interaction",
			slog.String("interaction_id", interactionID),
			slog.String("tenant_id", tenantID),
			slog.String("error", err.Error()),
		)
	}

	return interactionID
}

// recordLegacy falls back to the legacy conversation recording
func recordLegacy(ctx context.Context, params RecordInteractionParams) string {
	if params.Store == nil {
		return ""
	}

	metadata := map[string]string{
		"frontdoor": string(params.Frontdoor),
		"app":       params.AppName,
		"provider":  params.Provider,
	}
	if params.Streaming {
		metadata["stream"] = "true"
	}

	return Record(ctx, params.Store, "", params.CanonicalReq, params.CanonicalResp, metadata)
}

// extractRelevantHeaders extracts headers that are useful for observability
// without exposing sensitive information like API keys
func extractRelevantHeaders(headers http.Header) map[string]string {
	relevant := make(map[string]string)

	// List of headers to capture (case-insensitive)
	keysToCapture := []string{
		"User-Agent",
		"Content-Type",
		"Accept",
		"X-Request-Id",
		"X-Correlation-Id",
		"X-Trace-Id",
		// Anthropic specific headers
		"anthropic-version",
		"anthropic-beta",
		// OpenAI specific headers
		"openai-organization",
		// General debugging headers
		"X-Forwarded-For",
		"X-Real-Ip",
	}

	for _, key := range keysToCapture {
		if val := headers.Get(key); val != "" {
			relevant[key] = val
		}
	}

	return relevant
}

// buildUnmappedWarnings creates warning messages for unmapped fields
func buildUnmappedWarnings(unmappedFields []string) []string {
	if len(unmappedFields) == 0 {
		return nil
	}

	warnings := make([]string, len(unmappedFields))
	for i, field := range unmappedFields {
		warnings[i] = fmt.Sprintf("Field '%s' could not be mapped", field)
	}
	return warnings
}

// StartInteraction creates and persists an initial interaction record at the start of a request.
// This allows tracking in-progress requests and provides visibility even if the request fails.
func StartInteraction(ctx context.Context, store storage.ConversationStore, params RecordInteractionParams) (*domain.Interaction, error) {
	iStore, ok := store.(storage.InteractionStore)
	if !ok || iStore == nil {
		return nil, nil
	}

	// Always use gateway-owned ID as primary key
	// Check if frontdoor already assigned an ID via request metadata
	interactionID := ""
	if params.CanonicalReq != nil && params.CanonicalReq.Metadata != nil {
		if existingID := params.CanonicalReq.Metadata["interaction_id"]; existingID != "" {
			interactionID = existingID
		}
	}
	if interactionID == "" {
		interactionID = "int_" + uuid.New().String()
	}
	tenantID := tenantIDFromContext(ctx)

	interaction := domain.NewInteraction(interactionID, tenantID)
	interaction.Frontdoor = params.Frontdoor
	interaction.Provider = params.Provider
	interaction.AppName = params.AppName
	interaction.Streaming = params.Streaming
	interaction.Status = domain.InteractionStatusPending
	interaction.PreviousInteractionID = params.PreviousInteractionID
	interaction.ThreadKey = params.ThreadKey

	if params.CanonicalReq != nil {
		interaction.RequestedModel = params.CanonicalReq.Model
	}

	// Add request ID
	if reqID, ok := ctx.Value(middleware.RequestIDKey).(string); ok && reqID != "" {
		interaction.Metadata["request_id"] = reqID
	}

	// Capture request headers
	if params.RequestHeaders != nil {
		interaction.RequestHeaders = extractRelevantHeaders(params.RequestHeaders)
	}

	// Build request record
	interaction.Request = &domain.InteractionRequest{
		Raw:            params.RawRequest,
		UnmappedFields: params.UnmappedRequest,
	}

	if params.CanonicalReq != nil {
		canonicalJSON, err := json.Marshal(params.CanonicalReq)
		if err == nil {
			interaction.Request.CanonicalJSON = canonicalJSON
		}
	}

	// Decouple persistence from the request lifecycle
	persistCtx, cancel := buildPersistenceContext(ctx, 5*time.Second)
	defer cancel()

	if err := iStore.SaveInteraction(persistCtx, interaction); err != nil {
		return nil, err
	}

	return interaction, nil
}

// CompleteInteraction updates an existing interaction with response data.
func CompleteInteraction(ctx context.Context, store storage.ConversationStore, interaction *domain.Interaction, params RecordInteractionParams) error {
	iStore, ok := store.(storage.InteractionStore)
	if !ok || iStore == nil || interaction == nil {
		return nil
	}

	logger := slog.Default()

	interaction.Duration = params.Duration

	if params.CanonicalResp != nil {
		interaction.ServedModel = params.CanonicalResp.Model
		interaction.ProviderModel = params.CanonicalResp.ProviderModel
		// Store provider's response ID as metadata (NOT as our primary key)
		if params.CanonicalResp.ID != "" {
			interaction.Metadata["provider_response_id"] = params.CanonicalResp.ID
		}
	}

	// CRITICAL: Validate provider contract for debug data capture
	// All providers MUST capture ProviderRequestBody and RawResponse for successful interactions
	// This data is required for the transformation flow visualization in the UI
	if params.CanonicalResp != nil && params.Error == nil && !params.Streaming {
		if len(params.ProviderRequestBody) == 0 {
			panic(fmt.Sprintf(
				"PROVIDER CONTRACT VIOLATION: Provider '%s' failed to capture ProviderRequestBody. "+
					"All providers MUST marshal and return the actual request sent to upstream APIs in CanonicalResponse.ProviderRequestBody. "+
					"See internal/backend/openai/provider.go or internal/backend/anthropic/provider.go for reference implementation.",
				params.Provider,
			))
		}

		if len(params.RawResponse) == 0 {
			panic(fmt.Sprintf(
				"PROVIDER CONTRACT VIOLATION: Provider '%s' failed to capture RawResponse. "+
					"All providers MUST store raw response bytes from upstream APIs in CanonicalResponse.RawResponse. "+
					"See internal/backend/openai/provider.go or internal/backend/anthropic/provider.go for reference implementation.",
				params.Provider,
			))
		}
	}

	// Build transformation steps for flow visualization
	var transformationSteps []domain.TransformationStep

	// Step 1: Decode client request to canonical
	if params.CanonicalReq != nil {
		warnings := buildUnmappedWarnings(params.UnmappedRequest)
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "decode_client_request",
			Timestamp:   interaction.CreatedAt,
			Description: fmt.Sprintf("Decoded %s request to canonical format", params.Frontdoor),
			Details: map[string]interface{}{
				"unmapped_fields_count": len(params.UnmappedRequest),
			},
			Warnings: warnings,
		})
	}

	// Step 2: Encode canonical request for provider
	if len(params.ProviderRequestBody) > 0 {
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "encode_provider_request",
			Timestamp:   interaction.CreatedAt,
			Description: fmt.Sprintf("Encoded canonical request to %s format", params.Provider),
			Details: map[string]interface{}{
				"provider": params.Provider,
			},
		})
	}

	// Step 4: Decode provider response to canonical
	if params.CanonicalResp != nil {
		warnings := buildUnmappedWarnings(params.UnmappedResponse)
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "decode_provider_response",
			Timestamp:   interaction.CreatedAt,
			Description: fmt.Sprintf("Decoded %s response to canonical format", params.Provider),
			Details: map[string]interface{}{
				"unmapped_fields_count": len(params.UnmappedResponse),
			},
			Warnings: warnings,
		})
	}

	// Step 5: Encode response for client
	if len(params.ClientResponse) > 0 {
		transformationSteps = append(transformationSteps, domain.TransformationStep{
			Stage:       "encode_client_response",
			Timestamp:   interaction.CreatedAt,
			Codec:       string(params.Frontdoor),
			Description: fmt.Sprintf("Encoded canonical response to %s format", params.Frontdoor),
		})
	}

	interaction.TransformationSteps = transformationSteps

	// Build response record
	interaction.Response = &domain.InteractionResponse{
		Raw:            params.RawResponse,
		UnmappedFields: params.UnmappedResponse,
		ClientResponse: params.ClientResponse,
		FinishReason:   params.FinishReason,
	}

	if params.CanonicalResp != nil {
		canonicalJSON, err := json.Marshal(params.CanonicalResp)
		if err == nil {
			interaction.Response.CanonicalJSON = canonicalJSON
		}
		interaction.Response.Usage = &params.CanonicalResp.Usage

		if len(params.CanonicalResp.Choices) > 0 {
			interaction.Response.FinishReason = params.CanonicalResp.Choices[0].FinishReason
		}

		// Store provider's response ID in the response record
		if params.CanonicalResp.ID != "" {
			interaction.Response.ProviderResponseID = params.CanonicalResp.ID
		}
	}

	if len(params.ProviderRequestBody) > 0 && interaction.Request != nil {
		interaction.Request.ProviderRequest = params.ProviderRequestBody
	}

	// Set status based on outcome
	if params.Error != nil {
		interaction.Status = domain.InteractionStatusFailed
		interaction.Error = &domain.InteractionError{
			Type:    "error",
			Message: params.Error.Error(),
		}
		if apiErr, ok := params.Error.(*domain.APIError); ok {
			interaction.Error.Type = string(apiErr.Type)
			interaction.Error.Code = string(apiErr.Code)
		}
	} else {
		interaction.Status = domain.InteractionStatusCompleted
	}

	// Decouple persistence from the request lifecycle
	persistCtx, cancel := buildPersistenceContext(ctx, 5*time.Second)
	defer cancel()

	if err := iStore.UpdateInteraction(persistCtx, interaction); err != nil {
		logger.Error("failed to update interaction",
			slog.String("interaction_id", interaction.ID),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}
