package anthropic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	anthropicapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/conversation"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	anthropic_provider "github.com/tjfontaine/polyglot-llm-gateway/internal/provider/anthropic"
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

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)
	providerName := h.provider.Name()

	// Decode directly into Anthropic API request type
	var apiReq anthropicapi.MessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&apiReq); err != nil {
		logger.Error("failed to decode anthropic messages request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to canonical request using provider helper
	canonReq, err := anthropic_provider.ToCanonicalRequest(&apiReq)
	if err != nil {
		logger.Error("invalid anthropic messages request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Capture User-Agent from incoming request and pass it through
	canonReq.UserAgent = r.Header.Get("User-Agent")

	server.AddLogField(r.Context(), "requested_model", canonReq.Model)
	server.AddLogField(r.Context(), "frontdoor", "anthropic")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", providerName)

	if apiReq.Stream {
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

	// Convert to Anthropic response format using shared types
	anthropicResp := anthropicapi.MessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      resp.Model,
		StopReason: resp.Choices[0].FinishReason,
		Content: []anthropicapi.ResponseContent{
			{
				Type: "text",
				Text: resp.Choices[0].Message.Content,
			},
		},
		Usage: anthropicapi.MessagesUsage{
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

		// Use shared Anthropic streaming types
		chunk := anthropicapi.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: anthropicapi.BlockDelta{
				Type: "text_delta",
				Text: event.ContentDelta,
			},
		}

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
