package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Store is an in-memory implementation of ConversationStore, ResponseStore, and InteractionStore
type Store struct {
	mu            sync.RWMutex
	conversations map[string]*storage.Conversation
	responses     map[string]*storage.ResponseRecord
	interactions  map[string]*domain.Interaction
	threads       map[string]string
	events        []*domain.InteractionEvent
}

// New creates a new in-memory store
func New() *Store {
	return &Store{
		conversations: make(map[string]*storage.Conversation),
		responses:     make(map[string]*storage.ResponseRecord),
		interactions:  make(map[string]*domain.Interaction),
		threads:       make(map[string]string),
		events:        []*domain.InteractionEvent{},
	}
}

// Ensure Store implements InteractionStore (which extends ConversationStore)
var _ storage.InteractionStore = (*Store)(nil)

func (s *Store) CreateConversation(ctx context.Context, conv *storage.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.conversations[conv.ID]; exists {
		return fmt.Errorf("conversation %s already exists", conv.ID)
	}

	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()
	conv.Messages = []storage.StoredMessage{}

	s.conversations[conv.ID] = conv
	return nil
}

func (s *Store) GetConversation(ctx context.Context, id string) (*storage.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.conversations[id]
	if !exists {
		return nil, fmt.Errorf("conversation %s not found", id)
	}

	return conv, nil
}

func (s *Store) AddMessage(ctx context.Context, convID string, msg *storage.StoredMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, exists := s.conversations[convID]
	if !exists {
		return fmt.Errorf("conversation %s not found", convID)
	}

	msg.CreatedAt = time.Now()
	conv.Messages = append(conv.Messages, *msg)
	conv.UpdatedAt = time.Now()

	return nil
}

func (s *Store) ListConversations(ctx context.Context, opts storage.ListOptions) ([]*storage.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*storage.Conversation
	for _, conv := range s.conversations {
		if opts.TenantID != "" && conv.TenantID != opts.TenantID {
			continue
		}
		result = append(result, conv)
	}

	// Simple pagination
	start := opts.Offset
	if start >= len(result) {
		return []*storage.Conversation{}, nil
	}

	end := start + opts.Limit
	if opts.Limit == 0 || end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

func (s *Store) DeleteConversation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.conversations[id]; !exists {
		return fmt.Errorf("conversation %s not found", id)
	}

	delete(s.conversations, id)
	return nil
}

func (s *Store) Close() error {
	return nil
}

// Thread state (in-memory only)
func (s *Store) SetThreadState(threadKey, responseID string) error {
	if threadKey == "" || responseID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[threadKey] = responseID
	return nil
}

func (s *Store) GetThreadState(threadKey string) (string, error) {
	if threadKey == "" {
		return "", nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.threads[threadKey], nil
}

// ResponseStore implementation

func (s *Store) SaveResponse(ctx context.Context, resp *storage.ResponseRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp.CreatedAt = time.Now()
	resp.UpdatedAt = time.Now()
	s.responses[resp.ID] = resp
	return nil
}

func (s *Store) GetResponse(ctx context.Context, id string) (*storage.ResponseRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp, exists := s.responses[id]
	if !exists {
		return nil, fmt.Errorf("response %s not found", id)
	}
	return resp, nil
}

func (s *Store) UpdateResponseStatus(ctx context.Context, id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp, exists := s.responses[id]
	if !exists {
		return fmt.Errorf("response %s not found", id)
	}

	resp.Status = status
	resp.UpdatedAt = time.Now()
	return nil
}

func (s *Store) GetResponsesByPreviousID(ctx context.Context, previousID string) ([]*storage.ResponseRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*storage.ResponseRecord
	for _, resp := range s.responses {
		if resp.PreviousResponseID == previousID {
			result = append(result, resp)
		}
	}
	return result, nil
}

func (s *Store) ListResponses(ctx context.Context, opts storage.ListOptions) ([]*storage.ResponseRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*storage.ResponseRecord
	for _, resp := range s.responses {
		if opts.TenantID != "" && resp.TenantID != opts.TenantID {
			continue
		}
		result = append(result, resp)
	}

	// Simple pagination
	start := opts.Offset
	if start >= len(result) {
		return []*storage.ResponseRecord{}, nil
	}

	end := start + opts.Limit
	if opts.Limit == 0 || end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// InteractionStore implementation

func (s *Store) SaveInteraction(ctx context.Context, interaction *domain.Interaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if interaction.CreatedAt.IsZero() {
		interaction.CreatedAt = now
	}
	interaction.UpdatedAt = now
	s.interactions[interaction.ID] = interaction
	return nil
}

func (s *Store) AppendInteractionEvent(ctx context.Context, event *domain.InteractionEvent) error {
	if event == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *Store) ListInteractionEvents(ctx context.Context, interactionID string, opts storage.InteractionListOptions) ([]*domain.InteractionEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*domain.InteractionEvent
	for _, evt := range s.events {
		if interactionID != "" && evt.InteractionID != interactionID {
			continue
		}
		result = append(result, evt)
	}

	start := opts.Offset
	if start > len(result) {
		start = len(result)
	}
	end := start + opts.Limit
	if opts.Limit == 0 || end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (s *Store) GetInteraction(ctx context.Context, id string) (*domain.Interaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	interaction, exists := s.interactions[id]
	if !exists {
		return nil, fmt.Errorf("interaction %s not found", id)
	}
	return interaction, nil
}

func (s *Store) UpdateInteraction(ctx context.Context, interaction *domain.Interaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.interactions[interaction.ID]; !exists {
		return fmt.Errorf("interaction %s not found", interaction.ID)
	}

	interaction.UpdatedAt = time.Now()
	s.interactions[interaction.ID] = interaction
	return nil
}

func (s *Store) ListInteractions(ctx context.Context, opts storage.InteractionListOptions) ([]*domain.InteractionSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*domain.InteractionSummary
	for _, interaction := range s.interactions {
		// Apply filters
		if opts.TenantID != "" && interaction.TenantID != opts.TenantID {
			continue
		}
		if opts.Frontdoor != "" && interaction.Frontdoor != opts.Frontdoor {
			continue
		}
		if opts.Provider != "" && interaction.Provider != opts.Provider {
			continue
		}
		if opts.Status != "" && string(interaction.Status) != opts.Status {
			continue
		}
		result = append(result, interaction.ToSummary())
	}

	// Sort by UpdatedAt descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	// Simple pagination
	start := opts.Offset
	if start >= len(result) {
		return []*domain.InteractionSummary{}, nil
	}

	end := start + opts.Limit
	if opts.Limit == 0 || end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}
