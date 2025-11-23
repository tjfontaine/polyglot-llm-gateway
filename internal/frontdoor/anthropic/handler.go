package anthropic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/conversation"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

type Handler struct {
	provider domain.Provider
	store    storage.ConversationStore
	appName  string
	models   []domain.Model
}

func NewHandler(provider domain.Provider, store storage.ConversationStore, appName string, models []config.ModelListItem) *Handler {
	exposedModels := make([]domain.Model, 0, len(models))
	for _, model := range models {
		exposedModels = append(exposedModels, domain.Model{
			ID:      model.ID,
			Object:  model.Object,
			OwnedBy: model.OwnedBy,
			Created: model.Created,
		})
	}

	return &Handler{
		provider: provider,
		store:    store,
		appName:  appName,
		models:   exposedModels,
	}
}

// Anthropic Messages API request format
type MessagesRequest struct {
	Model       string         `json:"model"`
	Messages    []Message      `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	System      SystemMessages `json:"system,omitempty"`
	Temperature float32        `json:"temperature,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Message struct {
	Role    string             `json:"role"`
	Content MessageContentList `json:"content"`
	Name    string             `json:"name,omitempty"`
}

type SystemMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SystemMessages []SystemMessage

func (s *SystemMessages) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Accept a single string
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = SystemMessages{{Type: "text", Text: single}}
		return nil
	}

	// Accept an array of text blocks
	var blocks []SystemMessage
	if err := json.Unmarshal(data, &blocks); err == nil {
		*s = blocks
		return nil
	}

	return fmt.Errorf("system must be a string or array of text blocks")
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MessageContentList supports both the Claude string shortcut and the full array format.
type MessageContentList []MessageContent

func (m *MessageContentList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Allow the simple string form
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*m = MessageContentList{{Type: "text", Text: single}}
		return nil
	}

	// Allow the array-of-blocks form
	var blocks []MessageContent
	if err := json.Unmarshal(data, &blocks); err == nil {
		// Default missing types to text for compatibility
		for i := range blocks {
			if blocks[i].Type == "" {
				blocks[i].Type = "text"
			}
		}
		*m = blocks
		return nil
	}

	return fmt.Errorf("content must be a string or array of content blocks")
}

// Anthropic Messages API response format
type MessagesResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)
	providerName := h.provider.Name()

	var req MessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to decode anthropic messages request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	canonReq, err := toCanonicalRequest(req)
	if err != nil {
		logger.Error("invalid anthropic messages request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	server.AddLogField(r.Context(), "requested_model", canonReq.Model)
	server.AddLogField(r.Context(), "frontdoor", "anthropic")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", providerName)

	if req.Stream {
		h.handleStream(w, r, canonReq)
		return
	}

	resp, err := h.provider.Complete(r.Context(), canonReq)
	if err != nil {
		logger.Error("messages completion failed",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
			slog.String("requested_model", canonReq.Model),
			slog.String("provider", providerName),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to Anthropic format
	anthropicResp := MessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      resp.Model,
		StopReason: resp.Choices[0].FinishReason,
		Content: []ContentBlock{
			{
				Type: "text",
				Text: resp.Choices[0].Message.Content,
			},
		},
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	logger.Info("messages completion",
		slog.String("request_id", requestID),
		slog.String("frontdoor", "anthropic"),
		slog.String("app", h.appName),
		slog.String("provider", providerName),
		slog.String("requested_model", canonReq.Model),
		slog.String("served_model", resp.Model),
		slog.String("finish_reason", resp.Choices[0].FinishReason),
	)

	server.AddLogField(r.Context(), "served_model", resp.Model)

	metadata := map[string]string{
		"frontdoor": "anthropic",
		"app":       h.appName,
		"provider":  providerName,
	}
	conversation.Record(r.Context(), h.store, resp.ID, canonReq, resp, metadata)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthropicResp)
}

func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	server.AddLogField(r.Context(), "frontdoor", "anthropic")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", h.provider.Name())

	w.Header().Set("Content-Type", "application/json")

	if len(h.models) > 0 {
		json.NewEncoder(w).Encode(domain.ModelList{Object: "list", Data: h.models})
		return
	}

	list, err := h.provider.ListModels(r.Context())
	if err != nil {
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if list.Object == "" {
		list.Object = "list"
	}

	json.NewEncoder(w).Encode(list)
}

func toCanonicalRequest(req MessagesRequest) (*domain.CanonicalRequest, error) {
	canonReq := &domain.CanonicalRequest{
		Model:       req.Model,
		Messages:    make([]domain.Message, 0, len(req.Messages)+len(req.System)),
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	if len(req.Metadata) > 0 {
		canonReq.Metadata = make(map[string]string, len(req.Metadata))
		for k, v := range req.Metadata {
			strVal, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("metadata values must be strings (got %T for %s)", v, k)
			}
			canonReq.Metadata[k] = strVal
		}
	}

	// Add system messages first
	for _, sys := range req.System {
		if sys.Type != "" && sys.Type != "text" {
			return nil, fmt.Errorf("unsupported system block type: %s", sys.Type)
		}
		canonReq.Messages = append(canonReq.Messages, domain.Message{
			Role:    "system",
			Content: sys.Text,
		})
	}

	// Add conversation messages
	for idx, msg := range req.Messages {
		content, err := collapseContent(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("message %d: %w", idx, err)
		}

		canonReq.Messages = append(canonReq.Messages, domain.Message{
			Role:    msg.Role,
			Content: content,
			Name:    msg.Name,
		})
	}

	return canonReq, nil
}

func collapseContent(blocks MessageContentList) (string, error) {
	if len(blocks) == 0 {
		return "", fmt.Errorf("content is required")
	}

	var b strings.Builder
	for _, block := range blocks {
		blockType := block.Type
		if blockType == "" {
			blockType = "text"
		}
		if blockType != "text" {
			return "", fmt.Errorf("unsupported content block type: %s", blockType)
		}
		b.WriteString(block.Text)
	}

	return b.String(), nil
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request, req *domain.CanonicalRequest) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)
	providerName := h.provider.Name()

	events, err := h.provider.Stream(r.Context(), req)
	if err != nil {
		logger.Error("failed to start stream",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
			slog.String("requested_model", req.Model),
			slog.String("provider", providerName),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		server.AddError(r.Context(), fmt.Errorf("streaming not supported"))
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	var builder strings.Builder

	for event := range events {
		if event.Error != nil {
			logger.Error("stream event error",
				slog.String("request_id", requestID),
				slog.String("error", event.Error.Error()),
			)
			break
		}

		builder.WriteString(event.ContentDelta)

		// Anthropic streaming format
		chunk := struct {
			Type  string `json:"type"`
			Index int    `json:"index,omitempty"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta,omitempty"`
		}{
			Type:  "content_block_delta",
			Index: 0,
		}
		chunk.Delta.Type = "text_delta"
		chunk.Delta.Text = event.ContentDelta

		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", data)
		flusher.Flush()
	}

	// Send message_stop event
	fmt.Fprintf(w, "event: message_stop\ndata: {}\n\n")
	flusher.Flush()

	metadata := map[string]string{
		"frontdoor": "anthropic",
		"app":       h.appName,
		"provider":  providerName,
		"stream":    "true",
	}

	recordResp := &domain.CanonicalResponse{
		Model: req.Model,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    "assistant",
					Content: builder.String(),
				},
			},
		},
	}

	server.AddLogField(r.Context(), "served_model", recordResp.Model)

	conversation.Record(r.Context(), h.store, "", req, recordResp, metadata)

	logger.Info("messages stream completed",
		slog.String("request_id", requestID),
		slog.String("frontdoor", "anthropic"),
		slog.String("app", h.appName),
		slog.String("provider", providerName),
		slog.String("requested_model", req.Model),
		slog.String("served_model", recordResp.Model),
	)
}
