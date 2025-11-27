package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Store is an in-memory implementation of ConversationStore and ResponseStore
type Store struct {
	mu            sync.RWMutex
	conversations map[string]*storage.Conversation
	responses     map[string]*storage.ResponseRecord
}

// New creates a new in-memory store
func New() *Store {
	return &Store{
		conversations: make(map[string]*storage.Conversation),
		responses:     make(map[string]*storage.ResponseRecord),
	}
}

// Ensure Store implements ResponseStore
var _ storage.ResponseStore = (*Store)(nil)

func (s *Store) CreateConversation(ctx context.Context, conv *storage.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.conversations[conv.ID]; exists {
		return fmt.Errorf("conversation %s already exists", conv.ID)
	}

	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()
	conv.Messages = []storage.Message{}

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

func (s *Store) AddMessage(ctx context.Context, convID string, msg *storage.Message) error {
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
