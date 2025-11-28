package responses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// Handler handles OpenAI Responses API requests
type Handler struct {
	store    storage.ConversationStore
	provider domain.Provider
	logger   *slog.Logger
}

// NewHandler creates a new Responses API handler
func NewHandler(store storage.ConversationStore, provider domain.Provider) *Handler {
	return &Handler{
		store:    store,
		provider: provider,
		logger:   slog.Default(),
	}
}

// HandleCreateResponse handles POST /v1/responses
func (h *Handler) HandleCreateResponse(w http.ResponseWriter, r *http.Request) {
	var req domain.ResponsesAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	if req.Model == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "model is required")
		return
	}

	tenantID := getTenantID(r.Context())
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)

	h.logger.Info("responses API request",
		slog.String("request_id", requestID),
		slog.String("model", req.Model),
		slog.Bool("stream", req.Stream),
	)

	// Convert to canonical request
	canonReq := domain.FromResponsesAPIRequest(&req)
	canonReq.TenantID = tenantID
	canonReq.UserAgent = r.Header.Get("User-Agent")
	canonReq.SourceAPIType = domain.APITypeResponses

	// Handle previous response continuation
	if req.PreviousResponseID != "" {
		if err := h.loadPreviousContext(r.Context(), req.PreviousResponseID, canonReq); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
	}

	// Generate response ID
	responseID := "resp_" + uuid.New().String()

	if req.Stream {
		h.handleStreamingResponse(w, r, &req, canonReq, responseID, tenantID)
		return
	}

	h.handleNonStreamingResponse(w, r, &req, canonReq, responseID, tenantID)
}

func (h *Handler) handleNonStreamingResponse(w http.ResponseWriter, r *http.Request, req *domain.ResponsesAPIRequest, canonReq *domain.CanonicalRequest, responseID, tenantID string) {
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)

	// Call provider
	canonResp, err := h.provider.Complete(r.Context(), canonReq)
	if err != nil {
		h.logger.Error("provider error",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		h.writeError(w, http.StatusInternalServerError, "provider_error", err.Error())
		return
	}

	// Convert to Responses API format
	resp := h.canonicalToResponsesAPI(canonResp, responseID, req)

	// Save response if store supports it
	if respStore, ok := h.store.(storage.ResponseStore); ok && (req.Store == nil || *req.Store) {
		reqJSON, _ := json.Marshal(req)
		respJSON, _ := json.Marshal(resp)
		record := &storage.ResponseRecord{
			ID:                 responseID,
			TenantID:           tenantID,
			Status:             "completed",
			Model:              req.Model,
			Request:            reqJSON,
			Response:           respJSON,
			Metadata:           req.Metadata,
			PreviousResponseID: req.PreviousResponseID,
		}
		if err := respStore.SaveResponse(r.Context(), record); err != nil {
			h.logger.Warn("failed to save response",
				slog.String("request_id", requestID),
				slog.String("error", err.Error()),
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// streamingToolCall tracks a tool call being streamed
type streamingToolCall struct {
	id        string
	name      string
	arguments strings.Builder
	itemID    string
	index     int
}

func (h *Handler) handleStreamingResponse(w http.ResponseWriter, r *http.Request, req *domain.ResponsesAPIRequest, canonReq *domain.CanonicalRequest, responseID, tenantID string) {
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)

	// Start streaming from provider
	events, err := h.provider.Stream(r.Context(), canonReq)
	if err != nil {
		h.logger.Error("provider stream error",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		h.writeError(w, http.StatusInternalServerError, "provider_error", err.Error())
		return
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "streaming_error", "Streaming not supported")
		return
	}

	createdAt := time.Now().Unix()

	// Send response.created event (per spec: {"id": "...", "created": ..., "model": "..."})
	h.sendSSEEvent(w, flusher, "response.created", domain.ResponseCreatedEvent{
		ID:        responseID,
		CreatedAt: createdAt,
		Model:     req.Model,
	})

	// Tracking state
	var fullText strings.Builder
	var finalUsage *domain.Usage
	var finishReason string
	var outputItems []domain.ResponsesOutputItem
	activeToolCalls := make(map[int]*streamingToolCall)

	// Message item state
	itemIndex := 0
	messageItemID := "item_" + uuid.New().String()
	messageStarted := false

	for event := range events {
		if event.Error != nil {
			// Context cancellation is expected when client disconnects - log at info level
			if errors.Is(event.Error, context.Canceled) {
				h.logger.Info("stream canceled by client",
					slog.String("request_id", requestID),
				)
				// Don't send error event to client since they disconnected
				return
			}
			h.logger.Error("stream event error",
				slog.String("request_id", requestID),
				slog.String("error", event.Error.Error()),
			)
			// Send error response
			h.sendSSEEvent(w, flusher, "response.failed", domain.ResponseFailedEvent{
				Response: domain.ResponsesAPIResponse{
					ID:        responseID,
					Object:    "response",
					CreatedAt: createdAt,
					Status:    "failed",
					Model:     req.Model,
					Error: &domain.ResponsesError{
						Type:    "server_error",
						Code:    "stream_error",
						Message: event.Error.Error(),
					},
				},
			})
			return
		}

		// Handle text content delta
		if event.ContentDelta != "" {
			// Start message output item if not already started
			if !messageStarted {
				// Send response.output_item.added (per spec)
				h.sendSSEEvent(w, flusher, "response.output_item.added", domain.OutputItemAddedEvent{
					ItemIndex: itemIndex,
					Item: domain.ResponsesOutputItem{
						Type:    "message",
						ID:      messageItemID,
						Role:    "assistant",
						Content: []domain.ResponsesContentPart{},
					},
				})
				messageStarted = true
			}

			fullText.WriteString(event.ContentDelta)
			// Send response.output_item.delta with delta.content (per spec)
			h.sendSSEEvent(w, flusher, "response.output_item.delta", domain.OutputItemDeltaEvent{
				ItemIndex: itemIndex,
				Delta: domain.OutputItemDelta{
					Content: event.ContentDelta,
				},
			})
		}

		// Handle tool call start
		if event.Type == domain.EventTypeContentBlockStart && event.ToolCall != nil {
			tc := event.ToolCall
			toolItemID := "item_" + uuid.New().String()

			// Finalize message output item if we have text content
			if messageStarted && fullText.Len() > 0 {
				// Send response.output_item.done for message
				h.sendSSEEvent(w, flusher, "response.output_item.done", domain.OutputItemDoneEvent{
					ItemIndex: itemIndex,
					Item: domain.ResponsesOutputItem{
						Type: "message",
						ID:   messageItemID,
						Role: "assistant",
						Content: []domain.ResponsesContentPart{{
							Type: "output_text",
							Text: fullText.String(),
						}},
					},
				})
				outputItems = append(outputItems, domain.ResponsesOutputItem{
					Type:   "message",
					ID:     messageItemID,
					Role:   "assistant",
					Status: "completed",
					Content: []domain.ResponsesContentPart{{
						Type: "output_text",
						Text: fullText.String(),
					}},
				})
				itemIndex++
				messageStarted = false
			}

			// Track this tool call
			activeToolCalls[tc.Index] = &streamingToolCall{
				id:     tc.ID,
				name:   tc.Function.Name,
				itemID: toolItemID,
				index:  itemIndex,
			}

			// Send response.output_item.added for function_call
			h.sendSSEEvent(w, flusher, "response.output_item.added", domain.OutputItemAddedEvent{
				ItemIndex: itemIndex,
				Item: domain.ResponsesOutputItem{
					Type:   "function_call",
					ID:     toolItemID,
					CallID: tc.ID,
					Name:   tc.Function.Name,
				},
			})

			itemIndex++
		}

		// Handle tool call argument delta
		if event.Type == domain.EventTypeContentBlockDelta && event.ToolCall != nil {
			tc := event.ToolCall
			if stc, ok := activeToolCalls[tc.Index]; ok {
				stc.arguments.WriteString(tc.Function.Arguments)

				// Send response.output_item.delta with delta.arguments (per spec)
				h.sendSSEEvent(w, flusher, "response.output_item.delta", domain.OutputItemDeltaEvent{
					ItemIndex: stc.index,
					Delta: domain.OutputItemDelta{
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}

		// Handle tool call complete
		if event.Type == domain.EventTypeContentBlockStop && event.ToolCall != nil {
			tc := event.ToolCall
			if stc, ok := activeToolCalls[tc.Index]; ok {
				// Send response.output_item.done for function_call
				h.sendSSEEvent(w, flusher, "response.output_item.done", domain.OutputItemDoneEvent{
					ItemIndex: stc.index,
					Item: domain.ResponsesOutputItem{
						Type:      "function_call",
						ID:        stc.itemID,
						CallID:    stc.id,
						Name:      stc.name,
						Arguments: stc.arguments.String(),
					},
				})

				outputItems = append(outputItems, domain.ResponsesOutputItem{
					Type:      "function_call",
					ID:        stc.itemID,
					CallID:    stc.id,
					Name:      stc.name,
					Arguments: stc.arguments.String(),
					Status:    "completed",
				})

				delete(activeToolCalls, tc.Index)
			}
		}

		// Capture usage if provided
		if event.Usage != nil {
			finalUsage = event.Usage
		}

		// Capture finish reason for status determination
		if event.FinishReason != "" {
			finishReason = event.FinishReason
		}
	}

	// Finalize any remaining text content
	if messageStarted {
		// Send response.output_item.done for message
		h.sendSSEEvent(w, flusher, "response.output_item.done", domain.OutputItemDoneEvent{
			ItemIndex: 0,
			Item: domain.ResponsesOutputItem{
				Type: "message",
				ID:   messageItemID,
				Role: "assistant",
				Content: []domain.ResponsesContentPart{{
					Type: "output_text",
					Text: fullText.String(),
				}},
			},
		})
		outputItems = append([]domain.ResponsesOutputItem{{
			Type:   "message",
			ID:     messageItemID,
			Role:   "assistant",
			Status: "completed",
			Content: []domain.ResponsesContentPart{{
				Type: "output_text",
				Text: fullText.String(),
			}},
		}}, outputItems...)
	}

	// Determine finish_reason (per spec: "stop", "tool_calls", "length")
	if finishReason == "" {
		finishReason = "stop"
	}
	// Map Anthropic "tool_use" to OpenAI "tool_calls"
	if finishReason == "tool_use" {
		finishReason = "tool_calls"
	}

	// Send response.done event (per spec: {"usage": {...}, "finish_reason": "stop"})
	var usagePtr *domain.ResponsesUsage
	if finalUsage != nil {
		usagePtr = &domain.ResponsesUsage{
			InputTokens:  finalUsage.PromptTokens,
			OutputTokens: finalUsage.CompletionTokens,
			TotalTokens:  finalUsage.TotalTokens,
		}
	}
	h.sendSSEEvent(w, flusher, "response.done", domain.ResponseDoneEvent{
		Usage:        usagePtr,
		FinishReason: finishReason,
	})

	// Determine response status based on finish reason for storage
	responseStatus := "completed"
	if finishReason == "tool_calls" {
		responseStatus = "incomplete"
	}

	// Save response if store supports it
	if respStore, ok := h.store.(storage.ResponseStore); ok && (req.Store == nil || *req.Store) {
		finalResp := &domain.ResponsesAPIResponse{
			ID:                 responseID,
			Object:             "response",
			CreatedAt:          createdAt,
			Status:             responseStatus,
			Model:              req.Model,
			Output:             outputItems,
			Usage:              usagePtr,
			PreviousResponseID: req.PreviousResponseID,
			Metadata:           req.Metadata,
		}
		reqJSON, _ := json.Marshal(req)
		respJSON, _ := json.Marshal(finalResp)
		record := &storage.ResponseRecord{
			ID:                 responseID,
			TenantID:           tenantID,
			Status:             responseStatus,
			Model:              req.Model,
			Request:            reqJSON,
			Response:           respJSON,
			Metadata:           req.Metadata,
			PreviousResponseID: req.PreviousResponseID,
		}
		if err := respStore.SaveResponse(r.Context(), record); err != nil {
			h.logger.Warn("failed to save streaming response",
				slog.String("request_id", requestID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func (h *Handler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("failed to marshal SSE event", slog.String("error", err.Error()))
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonData)
	flusher.Flush()
}

func (h *Handler) canonicalToResponsesAPI(canonResp *domain.CanonicalResponse, responseID string, req *domain.ResponsesAPIRequest) *domain.ResponsesAPIResponse {
	resp := domain.ToResponsesAPIResponse(canonResp)
	resp.ID = responseID
	resp.PreviousResponseID = req.PreviousResponseID
	resp.Metadata = req.Metadata
	return resp
}

func (h *Handler) loadPreviousContext(ctx context.Context, previousID string, canonReq *domain.CanonicalRequest) error {
	respStore, ok := h.store.(storage.ResponseStore)
	if !ok {
		return fmt.Errorf("response storage not available")
	}

	record, err := respStore.GetResponse(ctx, previousID)
	if err != nil {
		return fmt.Errorf("previous response not found: %w", err)
	}

	var prevResp domain.ResponsesAPIResponse
	if err := json.Unmarshal(record.Response, &prevResp); err != nil {
		return fmt.Errorf("failed to parse previous response: %w", err)
	}

	// Build messages from previous response
	for _, item := range prevResp.Output {
		if item.Type == "message" {
			var content string
			for _, part := range item.Content {
				if part.Type == "output_text" {
					content += part.Text
				}
			}
			// Prepend previous messages
			msg := domain.Message{
				Role:    item.Role,
				Content: content,
			}
			canonReq.Messages = append([]domain.Message{msg}, canonReq.Messages...)
		}
	}

	// Recursively load earlier context
	if record.PreviousResponseID != "" {
		return h.loadPreviousContext(ctx, record.PreviousResponseID, canonReq)
	}

	return nil
}

// HandleGetResponse handles GET /v1/responses/{response_id}
func (h *Handler) HandleGetResponse(w http.ResponseWriter, r *http.Request) {
	responseID := chi.URLParam(r, "response_id")

	respStore, ok := h.store.(storage.ResponseStore)
	if !ok {
		h.writeError(w, http.StatusNotImplemented, "not_implemented", "Response storage not available")
		return
	}

	record, err := respStore.GetResponse(r.Context(), responseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Response not found")
		return
	}

	// Verify tenant access
	tenantID := getTenantID(r.Context())
	if record.TenantID != tenantID {
		h.writeError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	var resp domain.ResponsesAPIResponse
	if err := json.Unmarshal(record.Response, &resp); err != nil {
		h.writeError(w, http.StatusInternalServerError, "parse_error", "Failed to parse response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCancelResponse handles POST /v1/responses/{response_id}/cancel
func (h *Handler) HandleCancelResponse(w http.ResponseWriter, r *http.Request) {
	responseID := chi.URLParam(r, "response_id")

	respStore, ok := h.store.(storage.ResponseStore)
	if !ok {
		h.writeError(w, http.StatusNotImplemented, "not_implemented", "Response storage not available")
		return
	}

	record, err := respStore.GetResponse(r.Context(), responseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Response not found")
		return
	}

	// Verify tenant access
	tenantID := getTenantID(r.Context())
	if record.TenantID != tenantID {
		h.writeError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	// Only in_progress responses can be cancelled
	if record.Status != "in_progress" {
		h.writeError(w, http.StatusBadRequest, "invalid_state", "Response is not in progress")
		return
	}

	if err := respStore.UpdateResponseStatus(r.Context(), responseID, "cancelled"); err != nil {
		h.writeError(w, http.StatusInternalServerError, "update_error", "Failed to cancel response")
		return
	}

	var resp domain.ResponsesAPIResponse
	if err := json.Unmarshal(record.Response, &resp); err != nil {
		h.writeError(w, http.StatusInternalServerError, "parse_error", "Failed to parse response")
		return
	}
	resp.Status = "cancelled"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Thread-based API (legacy compatibility)

// HandleCreateThread creates a new conversation thread
func (h *Handler) HandleCreateThread(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metadata map[string]string `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	tenantID := getTenantID(r.Context())

	conv := &storage.Conversation{
		ID:       "thread_" + uuid.New().String(),
		TenantID: tenantID,
		Metadata: req.Metadata,
	}

	if err := h.store.CreateConversation(r.Context(), conv); err != nil {
		h.writeError(w, http.StatusInternalServerError, "storage_error", "Failed to create thread")
		return
	}

	resp := map[string]interface{}{
		"id":         conv.ID,
		"object":     "thread",
		"created_at": conv.CreatedAt.Unix(),
		"metadata":   conv.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetThread retrieves a thread
func (h *Handler) HandleGetThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Thread not found")
		return
	}

	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		h.writeError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	resp := map[string]interface{}{
		"id":         conv.ID,
		"object":     "thread",
		"created_at": conv.CreatedAt.Unix(),
		"metadata":   conv.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCreateMessage adds a message to a thread
func (h *Handler) HandleCreateMessage(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	var req struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Thread not found")
		return
	}

	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		h.writeError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	msg := &storage.Message{
		ID:      "msg_" + uuid.New().String(),
		Role:    req.Role,
		Content: req.Content,
	}

	if err := h.store.AddMessage(r.Context(), threadID, msg); err != nil {
		h.writeError(w, http.StatusInternalServerError, "storage_error", "Failed to add message")
		return
	}

	resp := map[string]interface{}{
		"id":         msg.ID,
		"object":     "thread.message",
		"created_at": msg.CreatedAt.Unix(),
		"thread_id":  threadID,
		"role":       msg.Role,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": map[string]string{"value": msg.Content},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleListMessages lists messages in a thread
func (h *Handler) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Thread not found")
		return
	}

	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		h.writeError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	messages := make([]map[string]interface{}, len(conv.Messages))
	for i, msg := range conv.Messages {
		messages[i] = map[string]interface{}{
			"id":         msg.ID,
			"object":     "thread.message",
			"created_at": msg.CreatedAt.Unix(),
			"thread_id":  threadID,
			"role":       msg.Role,
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]string{"value": msg.Content},
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

// HandleCreateRun runs a conversation through the provider
func (h *Handler) HandleCreateRun(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread_id")

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	conv, err := h.store.GetConversation(r.Context(), threadID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Thread not found")
		return
	}

	tenantID := getTenantID(r.Context())
	if conv.TenantID != tenantID {
		h.writeError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	// Convert to canonical request
	canonReq := &domain.CanonicalRequest{
		Model:     req.Model,
		Messages:  make([]domain.Message, len(conv.Messages)),
		UserAgent: r.Header.Get("User-Agent"),
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
		h.writeError(w, http.StatusInternalServerError, "provider_error", err.Error())
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
			h.writeError(w, http.StatusInternalServerError, "storage_error", "Failed to store response")
			return
		}
	}

	resp := map[string]interface{}{
		"id":         "run_" + uuid.New().String(),
		"object":     "thread.run",
		"created_at": time.Now().Unix(),
		"thread_id":  threadID,
		"status":     "completed",
		"model":      req.Model,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}

func getTenantID(ctx context.Context) string {
	if tenant := ctx.Value("tenant"); tenant != nil {
		return "default"
	}
	return "default"
}
