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
	Conversation           = corestorage.Conversation
	Message                = corestorage.Message
	ListOptions            = corestorage.ListOptions
	ResponseRecord         = corestorage.ResponseRecord
	Interaction            = corestorage.Interaction
	InteractionListOptions = corestorage.InteractionListOptions
	InteractionEvent       = domain.InteractionEvent
	ThreadStateStore       interface {
		SetThreadState(threadKey, responseID string) error
		GetThreadState(threadKey string) (string, error)
	}
)
