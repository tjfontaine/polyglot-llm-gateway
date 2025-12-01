package storage

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	corestorage "github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// Re-export storage interfaces and types from core/ports for backward compatibility.
type (
	ConversationStore      = corestorage.ConversationStore
	ResponseStore          = corestorage.ResponseStore
	InteractionStore       = corestorage.InteractionStore
	ShadowStore            = corestorage.ShadowStore
	Conversation           = corestorage.Conversation
	StoredMessage          = corestorage.StoredMessage
	ListOptions            = corestorage.ListOptions
	ResponseRecord         = corestorage.ResponseRecord
	InteractionSummary     = corestorage.InteractionSummary
	InteractionListOptions = corestorage.InteractionListOptions
	DivergenceListOptions  = corestorage.DivergenceListOptions
	InteractionEvent       = domain.InteractionEvent
	ShadowResult           = domain.ShadowResult
	ThreadStateStore       interface {
		SetThreadState(threadKey, responseID string) error
		GetThreadState(threadKey string) (string, error)
	}
)
