package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
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

// ResponseStore defines the interface for Responses API storage
type ResponseStore interface {
	ConversationStore

	// SaveResponse saves a response for later retrieval
	SaveResponse(ctx context.Context, resp *ResponseRecord) error

	// GetResponse retrieves a response by ID
	GetResponse(ctx context.Context, id string) (*ResponseRecord, error)

	// UpdateResponseStatus updates the status of a response
	UpdateResponseStatus(ctx context.Context, id, status string) error

	// GetResponsesByPreviousID retrieves responses that continue from a given response
	GetResponsesByPreviousID(ctx context.Context, previousID string) ([]*ResponseRecord, error)

	// ListResponses lists responses with pagination
	ListResponses(ctx context.Context, opts ListOptions) ([]*ResponseRecord, error)
}

// Interaction represents a unified view of either a conversation or a response
type Interaction struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"` // "conversation" or "response"
	TenantID     string            `json:"tenant_id"`
	Status       string            `json:"status,omitempty"` // For responses: in_progress, completed, failed, cancelled
	Model        string            `json:"model,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	MessageCount int               `json:"message_count,omitempty"` // For conversations
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
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

// ResponseRecord represents a stored Responses API response
type ResponseRecord struct {
	ID                 string            `json:"id"`
	TenantID           string            `json:"tenant_id"`
	Status             string            `json:"status"` // "in_progress", "completed", "failed", "cancelled"
	Model              string            `json:"model"`
	Request            json.RawMessage   `json:"request"`  // Serialized request
	Response           json.RawMessage   `json:"response"` // Serialized response
	Metadata           map[string]string `json:"metadata,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// InteractionStore defines the interface for unified interaction storage.
// This provides full visibility into request/response pairs with canonical mapping data.
type InteractionStore interface {
	ConversationStore

	// SaveInteraction saves an interaction record
	SaveInteraction(ctx context.Context, interaction *domain.Interaction) error

	// AppendInteractionEvent appends an auditable pipeline event for an interaction.
	AppendInteractionEvent(ctx context.Context, event *domain.InteractionEvent) error

	// ListInteractionEvents returns events for an interaction ordered by time.
	ListInteractionEvents(ctx context.Context, interactionID string, opts InteractionListOptions) ([]*domain.InteractionEvent, error)

	// GetInteraction retrieves an interaction by ID
	GetInteraction(ctx context.Context, id string) (*domain.Interaction, error)

	// UpdateInteraction updates an existing interaction
	UpdateInteraction(ctx context.Context, interaction *domain.Interaction) error

	// ListInteractions lists interactions with pagination and optional filtering
	ListInteractions(ctx context.Context, opts InteractionListOptions) ([]*domain.InteractionSummary, error)

	// SetThreadState stores the last response ID for a thread key
	SetThreadState(threadKey, responseID string) error

	// GetThreadState retrieves the last response ID for a thread key
	GetThreadState(threadKey string) (string, error)
}

// InteractionListOptions defines options for listing interactions
type InteractionListOptions struct {
	TenantID      string
	Frontdoor     domain.APIType // Filter by frontdoor type
	Provider      string         // Filter by provider
	Status        string         // Filter by status
	Limit         int
	Offset        int
	InteractionID string // Optional: restrict to a single interaction (for event listing)
}
