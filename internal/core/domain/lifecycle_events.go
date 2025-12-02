package domain

import (
	"time"
)

// LifecycleEvent represents a high-level lifecycle event for an interaction.
// These events are published to event buses for decoupled consumers (storage, analytics, etc.).
// This is distinct from InteractionEvent which tracks detailed pipeline audit trails.
type LifecycleEvent struct {
	Type          LifecycleEventType `json:"type"`
	InteractionID string             `json:"interaction_id"`
	TenantID      string             `json:"tenant_id"`
	Timestamp     time.Time          `json:"timestamp"`
	Data          interface{}        `json:"data"`
}

// LifecycleEventType identifies the type of lifecycle event.
type LifecycleEventType string

const (
	LifecycleEventStarted   LifecycleEventType = "interaction.started"
	LifecycleEventCompleted LifecycleEventType = "interaction.completed"
	LifecycleEventFailed    LifecycleEventType = "interaction.failed"
	LifecycleEventStreaming LifecycleEventType = "interaction.streaming"
)

// LifecycleStartedData contains data for interaction.started events.
type LifecycleStartedData struct {
	Frontdoor      APIType             `json:"frontdoor"`
	Provider       string              `json:"provider"`
	RequestedModel string              `json:"requested_model"`
	Request        *InteractionRequest `json:"request"`
}

// LifecycleCompletedData contains Data for interaction.completed events.
type LifecycleCompletedData struct {
	ServedModel  string               `json:"served_model"`
	Response     *InteractionResponse `json:"response"`
	Duration     time.Duration        `json:"duration_ns"`
	FinishReason string               `json:"finish_reason"`
}

// LifecycleFailedData contains data for interaction.failed events.
type LifecycleFailedData struct {
	Error *InteractionError `json:"error"`
}
