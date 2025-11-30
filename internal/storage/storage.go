package storage

import corestorage "github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"

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
)
