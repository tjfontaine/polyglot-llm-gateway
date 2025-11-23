package storage

import (
	"context"
	"time"
)

// ConversationStore defines the interface for conversation storage
type ConversationStore interface {
	// CreateConversation creates a new conversation
	CreateConversation(ctx context.Context, conv *Conversation) error

	// GetConversation retrieves a conversation by ID
	GetConversation(ctx context.Context, id string) (*Conversation, error)

	// AddMessage adds a message to a conversation
	AddMessage(ctx context.Context, convID string, msg *Message) error

	// ListConversations lists conversations with pagination
	ListConversations(ctx context.Context, opts ListOptions) ([]*Conversation, error)

	// DeleteConversation deletes a conversation
	DeleteConversation(ctx context.Context, id string) error

	// Close closes the storage connection
	Close() error
}

// Conversation represents a conversation thread
type Conversation struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Messages  []Message         `json:"messages"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Message represents a single message in a conversation
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ListOptions defines options for listing conversations
type ListOptions struct {
	TenantID string
	Limit    int
	Offset   int
}
