package responses

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"errors"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Handler handles OpenAI Responses API requests
type Handler struct {
	store    storage.ConversationStore
	provider domain.Provider
}

// NewHandler creates a new Responses API handler
func NewHandler(store storage.ConversationStore, provider domain.Provider) *Handler {
	return &Handler{
		store:    store,
		provider: provider,
	}
}

// Thread request/response types
type CreateThreadRequest struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ThreadResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type CreateMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessageResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	ThreadID  string            `json:"thread_id"`
	Role      string            `json:"role"`
	Content   []ContentBlock    `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type ContentBlock struct {
	Type string      `json:"type"`
	Text TextContent `json:"text"`
}

type TextContent struct {
	Value string `json:"value"`
}

type CreateRunRequest struct {
	Model string `json:"model"`
}

type RunResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	CreatedAt int64  `json:"created_at"`
	ThreadID  string `json:"thread_id"`
	Status    string `json:"status"`
	Model     string `json:"model"`
}

// Responses API request/response types (one-shot)
type CreateResponseRequest struct {
	Model           string                 `json:"model"`
	Input           json.RawMessage        `json:"input"`
	Messages        []CreateMessageRequest `json:"messages"` // Optional compatibility field
	Stream          bool                   `json:"stream,omitempty"`
	MaxOutputTokens int                    `json:"max_output_tokens,omitempty"`
	Temperature     float32                `json:"temperature,omitempty"`
}

type ResponseOutput struct {
	Role    string                `json:"role"`
	Content []ResponseTextContent `json:"content"`
}

type ResponseTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CreateResponseResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Status  string           `json:"status"`
	Output  []ResponseOutput `json:"output"`
	Usage   *domain.Usage    `json:"usage,omitempty"`
}

// HandleCreateThread creates a new conversation thread
func (h *Handler) HandleCreateThread(w http.ResponseWriter, r *http.Request) {
	var req CreateThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get tenant from context (set by auth middleware)
	tenantID := getTenantID(r.Context())

	conv := &storage.Conversation{
		ID:       "thread_" + uuid.New().String(),
		TenantID: tenantID,
		Metadata: req.Metadata,
	}

	if err := h.store.CreateConversation(r.Context(), conv); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create thread: %v", err), http.StatusInternalServerError)
		return
	}

	resp := ThreadResponse{
		ID:        conv.ID,
		Object:    "thread",
		CreatedAt: conv.CreatedAt.Unix(),
		Metadata:  conv.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetThread retrieves a thread
func (h *Handler) HandleGetThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	// Verify tenant access
	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	resp := ThreadResponse{
		ID:        conv.ID,
		Object:    "thread",
		CreatedAt: conv.CreatedAt.Unix(),
		Metadata:  conv.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCreateMessage adds a message to a thread
func (h *Handler) HandleCreateMessage(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	var req CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify thread exists and tenant has access
	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	msg := &storage.Message{
		ID:      "msg_" + uuid.New().String(),
		Role:    req.Role,
		Content: req.Content,
	}

	if err := h.store.AddMessage(r.Context(), threadID, msg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add message: %v", err), http.StatusInternalServerError)
		return
	}

	resp := MessageResponse{
		ID:        msg.ID,
		Object:    "thread.message",
		CreatedAt: msg.CreatedAt.Unix(),
		ThreadID:  threadID,
		Role:      msg.Role,
		Content: []ContentBlock{
			{
				Type: "text",
				Text: TextContent{Value: msg.Content},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCreateResponse provides a lightweight Responses API entrypoint similar to OpenAI's /v1/responses.
func (h *Handler) HandleCreateResponse(w http.ResponseWriter, r *http.Request) {
	var req CreateResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	if req.Stream {
		http.Error(w, "streaming responses are not yet supported", http.StatusBadRequest)
		return
	}

	messages, err := parseInputMessages(req.Input, req.Messages)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid input: %v", err), http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r.Context())
	canonReq := &domain.CanonicalRequest{
		TenantID:    tenantID,
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxOutputTokens,
		Temperature: req.Temperature,
	}

	canonResp, err := h.provider.Complete(r.Context(), canonReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("provider error: %v", err), http.StatusInternalServerError)
		return
	}

	if len(canonResp.Choices) == 0 {
		http.Error(w, "provider returned no choices", http.StatusInternalServerError)
		return
	}

	choice := canonResp.Choices[0]
	resp := CreateResponseResponse{
		ID:      canonResp.ID,
		Object:  "response",
		Created: canonResp.Created,
		Model:   canonResp.Model,
		Status:  "completed",
		Output: []ResponseOutput{
			{
				Role: choice.Message.Role,
				Content: []ResponseTextContent{
					{Type: "output_text", Text: choice.Message.Content},
				},
			},
		},
	}

	// Only attach usage if provider supplied it.
	if canonResp.Usage != (domain.Usage{}) {
		resp.Usage = &canonResp.Usage
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCreateRun runs a conversation through the provider
func (h *Handler) HandleCreateRun(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get conversation
	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	// Verify tenant access
	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Convert to canonical request
	canonReq := &domain.CanonicalRequest{
		Model:    req.Model,
		Messages: make([]domain.Message, len(conv.Messages)),
	}

	for i, msg := range conv.Messages {
		canonReq.Messages[i] = domain.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Call provider
	canonResp, err := h.provider.Complete(r.Context(), canonReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusInternalServerError)
		return
	}

	// Store assistant response
	if len(canonResp.Choices) > 0 {
		assistantMsg := &storage.Message{
			ID:      "msg_" + uuid.New().String(),
			Role:    "assistant",
			Content: canonResp.Choices[0].Message.Content,
		}

		if err := h.store.AddMessage(r.Context(), threadID, assistantMsg); err != nil {
			http.Error(w, fmt.Sprintf("Failed to store response: %v", err), http.StatusInternalServerError)
			return
		}
	}

	resp := RunResponse{
		ID:        "run_" + uuid.New().String(),
		Object:    "thread.run",
		CreatedAt: time.Now().Unix(),
		ThreadID:  threadID,
		Status:    "completed",
		Model:     req.Model,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleListMessages lists messages in a thread
func (h *Handler) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	// Verify tenant access
	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	messages := make([]MessageResponse, len(conv.Messages))
	for i, msg := range conv.Messages {
		messages[i] = MessageResponse{
			ID:        msg.ID,
			Object:    "thread.message",
			CreatedAt: msg.CreatedAt.Unix(),
			ThreadID:  threadID,
			Role:      msg.Role,
			Content: []ContentBlock{
				{
					Type: "text",
					Text: TextContent{Value: msg.Content},
				},
			},
		}
	}

	resp := map[string]interface{}{
		"object": "list",
		"data":   messages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getTenantID extracts tenant ID from context
func getTenantID(ctx context.Context) string {
	// Try to get tenant from context
	if tenant := ctx.Value("tenant"); tenant != nil {
		// Type assert to get tenant ID
		// This is a simplified version - in production you'd have proper tenant type
		return "default"
	}
	return "default"
}

func parseInputMessages(rawInput json.RawMessage, fallbackMessages []CreateMessageRequest) ([]domain.Message, error) {
	// Accept either the new Responses API "input" field or a legacy "messages" field.
	if len(rawInput) == 0 && len(fallbackMessages) == 0 {
		return nil, errors.New("input is required")
	}

	// Try to parse "input" as a simple string
	var inputText string
	if len(rawInput) > 0 && json.Unmarshal(rawInput, &inputText) == nil {
		return []domain.Message{{Role: "user", Content: inputText}}, nil
	}

	// Try to parse "input" as an array of role/content pairs
	var inputMessages []CreateMessageRequest
	if len(rawInput) > 0 && json.Unmarshal(rawInput, &inputMessages) == nil && len(inputMessages) > 0 {
		msgs := make([]domain.Message, 0, len(inputMessages))
		for _, m := range inputMessages {
			if m.Content == "" {
				continue
			}
			role := m.Role
			if role == "" {
				role = "user"
			}
			msgs = append(msgs, domain.Message{
				Role:    role,
				Content: m.Content,
			})
		}
		if len(msgs) == 0 {
			return nil, errors.New("input messages missing content")
		}
		return msgs, nil
	}

	// Fallback to explicit messages field
	if len(fallbackMessages) > 0 {
		msgs := make([]domain.Message, 0, len(fallbackMessages))
		for _, m := range fallbackMessages {
			if m.Content == "" {
				continue
			}
			role := m.Role
			if role == "" {
				role = "user"
			}
			msgs = append(msgs, domain.Message{
				Role:    role,
				Content: m.Content,
			})
		}
		if len(msgs) == 0 {
			return nil, errors.New("messages missing content")
		}
		return msgs, nil
	}

	return nil, errors.New("unsupported input format")
}
