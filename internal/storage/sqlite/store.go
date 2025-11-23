package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/tjfontaine/poly-llm-gateway/internal/storage"
)

// Store is a SQLite implementation of ConversationStore
type Store struct {
	db *sql.DB
}

// New creates a new SQLite store
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		metadata TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON conversations(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) CreateConversation(ctx context.Context, conv *storage.Conversation) error {
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()

	metadata, err := json.Marshal(conv.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `INSERT INTO conversations (id, tenant_id, metadata, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, query,
		conv.ID, conv.TenantID, string(metadata), conv.CreatedAt, conv.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

func (s *Store) GetConversation(ctx context.Context, id string) (*storage.Conversation, error) {
	query := `SELECT id, tenant_id, metadata, created_at, updated_at
	          FROM conversations WHERE id = ?`

	var conv storage.Conversation
	var metadataJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&conv.ID, &conv.TenantID, &metadataJSON, &conv.CreatedAt, &conv.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if err := json.Unmarshal([]byte(metadataJSON), &conv.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Load messages
	messages, err := s.getMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	conv.Messages = messages

	return &conv, nil
}

func (s *Store) getMessages(ctx context.Context, convID string) ([]storage.Message, error) {
	query := `SELECT id, role, content, created_at
	          FROM messages WHERE conversation_id = ?
	          ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []storage.Message
	for rows.Next() {
		var msg storage.Message
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (s *Store) AddMessage(ctx context.Context, convID string, msg *storage.Message) error {
	msg.CreatedAt = time.Now()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert message
	query := `INSERT INTO messages (id, conversation_id, role, content, created_at)
	          VALUES (?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, query,
		msg.ID, convID, msg.Role, msg.Content, msg.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Update conversation updated_at
	updateQuery := `UPDATE conversations SET updated_at = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, updateQuery, time.Now(), convID)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

func (s *Store) ListConversations(ctx context.Context, opts storage.ListOptions) ([]*storage.Conversation, error) {
	query := `SELECT id, tenant_id, metadata, created_at, updated_at
	          FROM conversations WHERE tenant_id = ?
	          ORDER BY updated_at DESC
	          LIMIT ? OFFSET ?`

	limit := opts.Limit
	if limit == 0 {
		limit = 100 // default limit
	}

	rows, err := s.db.QueryContext(ctx, query, opts.TenantID, limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*storage.Conversation
	for rows.Next() {
		var conv storage.Conversation
		var metadataJSON string

		if err := rows.Scan(&conv.ID, &conv.TenantID, &metadataJSON,
			&conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &conv.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Load messages for each conversation
		messages, err := s.getMessages(ctx, conv.ID)
		if err != nil {
			return nil, err
		}
		conv.Messages = messages

		conversations = append(conversations, &conv)
	}

	return conversations, rows.Err()
}

func (s *Store) DeleteConversation(ctx context.Context, id string) error {
	query := `DELETE FROM conversations WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("conversation %s not found", id)
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
