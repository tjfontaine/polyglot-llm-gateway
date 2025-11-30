package domain

import (
	"encoding/json"
	"time"
)

// ShadowResult captures the full pipeline execution for a shadow provider.
// It mirrors the primary interaction structure to enable complete comparison.
type ShadowResult struct {
	// ID uniquely identifies this shadow result
	ID string `json:"id"`

	// InteractionID links this shadow to the primary interaction
	InteractionID string `json:"interaction_id"`

	// ProviderName is the shadow provider that handled this request
	ProviderName string `json:"provider_name"`

	// ProviderModel is the model used by the shadow provider (if overridden)
	ProviderModel string `json:"provider_model,omitempty"`

	// Request contains the shadow request transformation
	Request *ShadowRequest `json:"request"`

	// Response contains the shadow response transformation (nil if error)
	Response *ShadowResponse `json:"response,omitempty"`

	// Error contains any error that occurred during shadow execution
	Error *InteractionError `json:"error,omitempty"`

	// Duration is the total time taken for the shadow request
	Duration time.Duration `json:"duration_ns"`

	// TokensIn is the input token count from the shadow provider
	TokensIn int `json:"tokens_in,omitempty"`

	// TokensOut is the output token count from the shadow provider
	TokensOut int `json:"tokens_out,omitempty"`

	// Divergences contains structural differences from the primary response
	Divergences []Divergence `json:"divergences,omitempty"`

	// HasStructuralDivergence is a quick flag for filtering
	HasStructuralDivergence bool `json:"has_structural_divergence"`

	// CreatedAt is when this shadow execution started
	CreatedAt time.Time `json:"created_at"`
}

// ShadowRequest captures the shadow request transformation.
type ShadowRequest struct {
	// Canonical is the canonical request (same as primary, may have model overridden)
	Canonical json.RawMessage `json:"canonical,omitempty"`

	// ProviderRequest is the provider-specific request sent to shadow provider
	ProviderRequest json.RawMessage `json:"provider_request,omitempty"`
}

// ShadowResponse captures the shadow response transformation.
type ShadowResponse struct {
	// Raw is the raw response from shadow provider
	Raw json.RawMessage `json:"raw,omitempty"`

	// Canonical is the canonical translation of shadow response
	Canonical json.RawMessage `json:"canonical,omitempty"`

	// ClientResponse is what the client WOULD have received (re-encoded via frontdoor codec)
	ClientResponse json.RawMessage `json:"client_response,omitempty"`

	// FinishReason from the shadow response
	FinishReason string `json:"finish_reason,omitempty"`

	// Usage from the shadow response
	Usage *Usage `json:"usage,omitempty"`
}

// Divergence describes a structural difference between primary and shadow responses.
type Divergence struct {
	// Type categorizes the kind of divergence
	Type DivergenceType `json:"type"`

	// Path is the JSON path where the divergence occurred
	Path string `json:"path"`

	// Description provides a human-readable explanation
	Description string `json:"description"`

	// Primary is the value from the primary response (if applicable)
	Primary any `json:"primary,omitempty"`

	// Shadow is the value from the shadow response (if applicable)
	Shadow any `json:"shadow,omitempty"`
}

// DivergenceType categorizes the kind of structural divergence.
type DivergenceType string

const (
	// DivergenceMissingField indicates a field present in primary but missing in shadow
	DivergenceMissingField DivergenceType = "missing_field"

	// DivergenceExtraField indicates a field present in shadow but missing in primary
	DivergenceExtraField DivergenceType = "extra_field"

	// DivergenceTypeMismatch indicates different types for the same field
	DivergenceTypeMismatch DivergenceType = "type_mismatch"

	// DivergenceArrayLength indicates different array lengths
	DivergenceArrayLength DivergenceType = "array_length"

	// DivergenceNullMismatch indicates one is null and the other is not
	DivergenceNullMismatch DivergenceType = "null_mismatch"
)

// ShadowResultSummary provides a lightweight view for listing shadow results.
type ShadowResultSummary struct {
	ID                      string        `json:"id"`
	InteractionID           string        `json:"interaction_id"`
	ProviderName            string        `json:"provider_name"`
	ProviderModel           string        `json:"provider_model,omitempty"`
	Duration                time.Duration `json:"duration_ns"`
	TokensIn                int           `json:"tokens_in,omitempty"`
	TokensOut               int           `json:"tokens_out,omitempty"`
	Status                  string        `json:"status"` // "success" or "error"
	HasStructuralDivergence bool          `json:"has_structural_divergence"`
	DivergenceCount         int           `json:"divergence_count"`
	DivergenceTypes         []string      `json:"divergence_types,omitempty"`
	CreatedAt               time.Time     `json:"created_at"`
}

// ToSummary converts a ShadowResult to a ShadowResultSummary.
func (s *ShadowResult) ToSummary() *ShadowResultSummary {
	status := "success"
	if s.Error != nil {
		status = "error"
	}

	// Collect unique divergence types
	typeSet := make(map[DivergenceType]bool)
	for _, d := range s.Divergences {
		typeSet[d.Type] = true
	}
	types := make([]string, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, string(t))
	}

	return &ShadowResultSummary{
		ID:                      s.ID,
		InteractionID:           s.InteractionID,
		ProviderName:            s.ProviderName,
		ProviderModel:           s.ProviderModel,
		Duration:                s.Duration,
		TokensIn:                s.TokensIn,
		TokensOut:               s.TokensOut,
		Status:                  status,
		HasStructuralDivergence: s.HasStructuralDivergence,
		DivergenceCount:         len(s.Divergences),
		DivergenceTypes:         types,
		CreatedAt:               s.CreatedAt,
	}
}
