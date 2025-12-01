package domain

import (
	"encoding/json"
	"time"
)

// Interaction represents a single request/response exchange through the gateway.
// This provides full bidirectional visibility into what was received, how it was
// translated to canonical format, what was sent to the provider, and the response.
type Interaction struct {
	// ID uniquely identifies this interaction
	ID string `json:"id"`

	// TenantID identifies the tenant for multi-tenant deployments
	TenantID string `json:"tenant_id"`

	// Frontdoor identifies the API type that received the request (openai, anthropic, responses)
	Frontdoor APIType `json:"frontdoor"`

	// Provider identifies the backend provider that handled the request
	Provider string `json:"provider"`

	// AppName identifies the application configuration used
	AppName string `json:"app_name,omitempty"`

	// RequestedModel is the model requested by the client
	RequestedModel string `json:"requested_model"`

	// ServedModel is the model actually used by the provider
	ServedModel string `json:"served_model,omitempty"`

	// ProviderModel is the actual model name as known by the provider (before any mapping)
	ProviderModel string `json:"provider_model,omitempty"`

	// Streaming indicates if this was a streaming request
	Streaming bool `json:"streaming"`

	// Request contains the incoming request details
	Request *InteractionRequest `json:"request"`

	// Response contains the outgoing response details
	Response *InteractionResponse `json:"response,omitempty"`

	// Error contains any error that occurred during processing
	Error *InteractionError `json:"error,omitempty"`

	// Metadata contains additional key-value pairs for the interaction
	Metadata map[string]string `json:"metadata,omitempty"`

	// RequestHeaders contains relevant HTTP headers from the incoming request
	RequestHeaders map[string]string `json:"request_headers,omitempty"`

	// Status indicates the current state of the interaction
	Status InteractionStatus `json:"status"`

	// Duration is the total time taken for the interaction
	Duration time.Duration `json:"duration_ns"`

	// TransformationSteps captures the transformation flow for debugging
	TransformationSteps []TransformationStep `json:"transformation_steps,omitempty"`

	// PreviousInteractionID links to a previous interaction for continuations (Responses API)
	PreviousInteractionID string `json:"previous_interaction_id,omitempty"`

	// ThreadKey is an optional key for grouping related interactions
	ThreadKey string `json:"thread_key,omitempty"`

	// CreatedAt is when the interaction was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the interaction was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// InteractionStatus represents the status of an interaction
type InteractionStatus string

const (
	InteractionStatusPending    InteractionStatus = "pending"
	InteractionStatusInProgress InteractionStatus = "in_progress"
	InteractionStatusCompleted  InteractionStatus = "completed"
	InteractionStatusFailed     InteractionStatus = "failed"
	InteractionStatusCancelled  InteractionStatus = "cancelled"
)

// InteractionRequest contains details about the incoming request
type InteractionRequest struct {
	// Raw is the original raw request body
	Raw json.RawMessage `json:"raw,omitempty"`

	// Canonical is the request after translation to canonical format
	Canonical *CanonicalRequest `json:"canonical,omitempty"`

	// CanonicalJSON is a JSON representation of the canonical request (for storage)
	CanonicalJSON json.RawMessage `json:"canonical_json,omitempty"`

	// UnmappedFields contains field names from the raw request that were not mapped
	// to the canonical format (for debugging and visibility)
	UnmappedFields []string `json:"unmapped_fields,omitempty"`

	// ProviderRequest is the request sent to the provider (after any transformations)
	ProviderRequest json.RawMessage `json:"provider_request,omitempty"`
}

// InteractionResponse contains details about the response
type InteractionResponse struct {
	// Raw is the raw response from the provider
	Raw json.RawMessage `json:"raw,omitempty"`

	// Canonical is the response after translation from provider format
	Canonical *CanonicalResponse `json:"canonical,omitempty"`

	// CanonicalJSON is a JSON representation of the canonical response (for storage)
	CanonicalJSON json.RawMessage `json:"canonical_json,omitempty"`

	// UnmappedFields contains field names from the provider response that were not mapped
	UnmappedFields []string `json:"unmapped_fields,omitempty"`

	// ClientResponse is what was actually sent back to the client
	ClientResponse json.RawMessage `json:"client_response,omitempty"`

	// ProviderResponseID is the ID assigned by the provider (e.g., "chatcmpl-xxx", "msg_xxx")
	// This is stored for reference but NOT used as our primary key
	ProviderResponseID string `json:"provider_response_id,omitempty"`

	// FinishReason indicates why the model stopped generating
	FinishReason string `json:"finish_reason,omitempty"`

	// Usage contains token usage information
	Usage *Usage `json:"usage,omitempty"`
}

// InteractionError contains error details
type InteractionError struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// TransformationStep captures details about a single transformation in the request/response flow
type TransformationStep struct {
	// Stage identifies the transformation stage
	Stage string `json:"stage"` // "decode_request", "model_mapping", "encode_provider_request", etc.

	// Timestamp when this transformation started
	Timestamp time.Time `json:"timestamp"`

	// Duration of this transformation
	Duration time.Duration `json:"duration_ns"`

	// Codec used for this transformation (if applicable)
	Codec string `json:"codec,omitempty"` // "openai", "anthropic"

	// Description of what happened in human-readable form
	Description string `json:"description"`

	// Details provides structured information about the transformation
	Details map[string]interface{} `json:"details,omitempty"`

	// Warnings captures any issues during transformation (e.g., unmapped fields)
	Warnings []string `json:"warnings,omitempty"`
}

// ModelMappingMetadata contains details about model mapping and provider selection
type ModelMappingMetadata struct {
	OriginalModel  string `json:"original_model"`
	MappedModel    string `json:"mapped_model"`
	ProviderModel  string `json:"provider_model,omitempty"`
	Reason         string `json:"reason"` // "rewrite_rule", "prefix_match", "fallback", "default"
	RuleMatched    string `json:"rule_matched,omitempty"`
	ProviderChosen string `json:"provider_chosen"`
}

// InteractionSummary provides a lightweight view of an interaction for listing
type InteractionSummary struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Frontdoor      APIType           `json:"frontdoor"`
	Provider       string            `json:"provider"`
	AppName        string            `json:"app_name,omitempty"`
	RequestedModel string            `json:"requested_model"`
	ServedModel    string            `json:"served_model,omitempty"`
	Streaming      bool              `json:"streaming"`
	Status         InteractionStatus `json:"status"`
	Duration       time.Duration     `json:"duration_ns"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// ToSummary converts an Interaction to an InteractionSummary
func (i *Interaction) ToSummary() *InteractionSummary {
	return &InteractionSummary{
		ID:             i.ID,
		TenantID:       i.TenantID,
		Frontdoor:      i.Frontdoor,
		Provider:       i.Provider,
		AppName:        i.AppName,
		RequestedModel: i.RequestedModel,
		ServedModel:    i.ServedModel,
		Streaming:      i.Streaming,
		Status:         i.Status,
		Duration:       i.Duration,
		Metadata:       i.Metadata,
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
	}
}

// NewInteraction creates a new Interaction with default values
func NewInteraction(id, tenantID string) *Interaction {
	now := time.Now()
	return &Interaction{
		ID:        id,
		TenantID:  tenantID,
		Status:    InteractionStatusPending,
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
