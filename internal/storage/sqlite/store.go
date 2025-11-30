package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Store is a SQLite implementation of ConversationStore, ResponseStore, and InteractionStore
type Store struct {
	db *sql.DB
}

// Ensure Store implements InteractionStore (which extends ConversationStore)
var _ storage.InteractionStore = (*Store)(nil)

// New creates a new SQLite store
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
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
	statements := []string{
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			metadata TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS responses (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			status TEXT NOT NULL,
			model TEXT NOT NULL,
			request TEXT NOT NULL,
			response TEXT,
			metadata TEXT,
			previous_response_id TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS interactions (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			frontdoor TEXT NOT NULL,
			provider TEXT NOT NULL,
			app_name TEXT,
			requested_model TEXT NOT NULL,
			served_model TEXT,
			provider_model TEXT,
			streaming INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			duration_ns INTEGER,
			request_raw TEXT,
			request_canonical TEXT,
			request_unmapped_fields TEXT,
			request_provider TEXT,
			response_raw TEXT,
			response_canonical TEXT,
			response_unmapped_fields TEXT,
			response_client TEXT,
			response_finish_reason TEXT,
			response_usage TEXT,
			error_type TEXT,
			error_code TEXT,
			error_message TEXT,
			metadata TEXT,
			request_headers TEXT,
			transformation_steps TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS thread_state (
			thread_key TEXT PRIMARY KEY,
			response_id TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON conversations(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_responses_tenant ON responses(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_responses_previous ON responses(previous_response_id)`,
		`CREATE INDEX IF NOT EXISTS idx_interactions_tenant ON interactions(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_interactions_frontdoor ON interactions(frontdoor)`,
		`CREATE INDEX IF NOT EXISTS idx_interactions_provider ON interactions(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_interactions_status ON interactions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_thread_state_updated ON thread_state(updated_at)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w", err)
		}
	}

	return nil
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

// SetThreadState stores the last response ID for a given thread key.
func (s *Store) SetThreadState(threadKey, responseID string) error {
	if threadKey == "" || responseID == "" {
		return nil
	}
	_, err := s.db.Exec(`
	INSERT INTO thread_state (thread_key, response_id, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(thread_key) DO UPDATE SET response_id=excluded.response_id, updated_at=CURRENT_TIMESTAMP;
	`, threadKey, responseID)
	return err
}

// GetThreadState retrieves the last response ID for a given thread key.
func (s *Store) GetThreadState(threadKey string) (string, error) {
	if threadKey == "" {
		return "", nil
	}
	var respID string
	err := s.db.QueryRow(`SELECT response_id FROM thread_state WHERE thread_key = ?`, threadKey).Scan(&respID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return respID, err
}

// ResponseStore implementation

func (s *Store) SaveResponse(ctx context.Context, resp *storage.ResponseRecord) error {
	resp.CreatedAt = time.Now()
	resp.UpdatedAt = time.Now()

	metadata, err := json.Marshal(resp.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `INSERT INTO responses (id, tenant_id, status, model, request, response, metadata, previous_response_id, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, query,
		resp.ID, resp.TenantID, resp.Status, resp.Model,
		string(resp.Request), string(resp.Response), string(metadata),
		resp.PreviousResponseID, resp.CreatedAt, resp.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save response: %w", err)
	}

	return nil
}

func (s *Store) GetResponse(ctx context.Context, id string) (*storage.ResponseRecord, error) {
	query := `SELECT id, tenant_id, status, model, request, response, metadata, previous_response_id, created_at, updated_at
	          FROM responses WHERE id = ?`

	var resp storage.ResponseRecord
	var requestStr, responseStr, metadataStr, previousID sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&resp.ID, &resp.TenantID, &resp.Status, &resp.Model,
		&requestStr, &responseStr, &metadataStr, &previousID,
		&resp.CreatedAt, &resp.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("response %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	if requestStr.Valid {
		resp.Request = json.RawMessage(requestStr.String)
	}
	if responseStr.Valid {
		resp.Response = json.RawMessage(responseStr.String)
	}
	if metadataStr.Valid && metadataStr.String != "" {
		if err := json.Unmarshal([]byte(metadataStr.String), &resp.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if previousID.Valid {
		resp.PreviousResponseID = previousID.String
	}

	return &resp, nil
}

func (s *Store) UpdateResponseStatus(ctx context.Context, id, status string) error {
	query := `UPDATE responses SET status = ?, updated_at = ? WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update response status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("response %s not found", id)
	}

	return nil
}

func (s *Store) GetResponsesByPreviousID(ctx context.Context, previousID string) ([]*storage.ResponseRecord, error) {
	query := `SELECT id, tenant_id, status, model, request, response, metadata, previous_response_id, created_at, updated_at
	          FROM responses WHERE previous_response_id = ?
	          ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, previousID)
	if err != nil {
		return nil, fmt.Errorf("failed to query responses: %w", err)
	}
	defer rows.Close()

	var responses []*storage.ResponseRecord
	for rows.Next() {
		var resp storage.ResponseRecord
		var requestStr, responseStr, metadataStr, prevID sql.NullString

		if err := rows.Scan(&resp.ID, &resp.TenantID, &resp.Status, &resp.Model,
			&requestStr, &responseStr, &metadataStr, &prevID,
			&resp.CreatedAt, &resp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan response: %w", err)
		}

		if requestStr.Valid {
			resp.Request = json.RawMessage(requestStr.String)
		}
		if responseStr.Valid {
			resp.Response = json.RawMessage(responseStr.String)
		}
		if metadataStr.Valid && metadataStr.String != "" {
			if err := json.Unmarshal([]byte(metadataStr.String), &resp.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		if prevID.Valid {
			resp.PreviousResponseID = prevID.String
		}

		responses = append(responses, &resp)
	}

	return responses, rows.Err()
}

func (s *Store) ListResponses(ctx context.Context, opts storage.ListOptions) ([]*storage.ResponseRecord, error) {
	query := `SELECT id, tenant_id, status, model, request, response, metadata, previous_response_id, created_at, updated_at
	          FROM responses WHERE tenant_id = ?
	          ORDER BY updated_at DESC
	          LIMIT ? OFFSET ?`

	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, query, opts.TenantID, limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query responses: %w", err)
	}
	defer rows.Close()

	var responses []*storage.ResponseRecord
	for rows.Next() {
		var resp storage.ResponseRecord
		var requestStr, responseStr, metadataStr, prevID sql.NullString

		if err := rows.Scan(&resp.ID, &resp.TenantID, &resp.Status, &resp.Model,
			&requestStr, &responseStr, &metadataStr, &prevID,
			&resp.CreatedAt, &resp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan response: %w", err)
		}

		if requestStr.Valid {
			resp.Request = json.RawMessage(requestStr.String)
		}
		if responseStr.Valid {
			resp.Response = json.RawMessage(responseStr.String)
		}
		if metadataStr.Valid && metadataStr.String != "" {
			if err := json.Unmarshal([]byte(metadataStr.String), &resp.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		if prevID.Valid {
			resp.PreviousResponseID = prevID.String
		}

		responses = append(responses, &resp)
	}

	return responses, rows.Err()
}
