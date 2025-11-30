package domain

import (
	"encoding/json"
	"time"
)

// InteractionEvent represents a single auditable step in the request/response pipeline.
// It is append-only and grouped by InteractionID to form a timeline.
type InteractionEvent struct {
	ID                 string          `json:"id"`
	InteractionID      string          `json:"interaction_id"`
	Stage              string          `json:"stage"`     // e.g., frontdoor_decode, provider_encode, provider_decode, thread_resolve, thread_update, frontdoor_encode, error
	Direction          string          `json:"direction"` // ingress | egress | internal
	APIType            APIType         `json:"api_type"`
	Frontdoor          string          `json:"frontdoor,omitempty"`
	Provider           string          `json:"provider,omitempty"`
	AppName            string          `json:"app_name,omitempty"`
	ModelRequested     string          `json:"model_requested,omitempty"`
	ModelServed        string          `json:"model_served,omitempty"`
	ProviderModel      string          `json:"provider_model,omitempty"`
	ThreadKey          string          `json:"thread_key,omitempty"`
	PreviousResponseID string          `json:"previous_response_id,omitempty"`
	Raw                json.RawMessage `json:"raw,omitempty"`       // payload as seen on the wire for this stage
	Canonical          json.RawMessage `json:"canonical,omitempty"` // canonical representation (if applicable)
	Headers            json.RawMessage `json:"headers,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}
