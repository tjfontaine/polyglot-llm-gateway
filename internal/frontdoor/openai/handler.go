package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/codec"
	openaicodec "github.com/tjfontaine/polyglot-llm-gateway/internal/codec/openai"
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
	codec    codec.Codec
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
		codec:    openaicodec.New(),
	}
}

func (h *Handler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("failed to read request body",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use codec to decode request to canonical format
	req, err := h.codec.DecodeRequest(body)
	if err != nil {
		logger.Error("failed to decode openai chat completion request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Capture User-Agent from incoming request and pass it through
	req.UserAgent = r.Header.Get("User-Agent")

	server.AddLogField(r.Context(), "frontdoor", "openai")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", h.provider.Name())
	server.AddLogField(r.Context(), "requested_model", req.Model)

	if req.Stream {
		h.handleStream(w, r, req)
		return
	}

	resp, err := h.provider.Complete(r.Context(), req)
	if err != nil {
		logger.Error("chat completion failed",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
			slog.String("requested_model", req.Model),
			slog.String("provider", h.provider.Name()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requestedModel := req.Model
	servedModel := resp.Model
	logger.Info("chat completion",
		slog.String("frontdoor", "openai"),
		slog.String("app", h.appName),
		slog.String("provider", h.provider.Name()),
		slog.String("requested_model", requestedModel),
		slog.String("served_model", servedModel),
	)
	server.AddLogField(r.Context(), "served_model", servedModel)

	metadata := map[string]string{
		"frontdoor": "openai",
		"app":       h.appName,
		"provider":  h.provider.Name(),
	}
	conversation.Record(r.Context(), h.store, resp.ID, req, resp, metadata)

	// Use codec to encode response
	respBody, err := h.codec.EncodeResponse(resp)
	if err != nil {
		logger.Error("failed to encode response",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	server.AddLogField(r.Context(), "frontdoor", "openai")
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

	events, err := h.provider.Stream(r.Context(), req)
	if err != nil {
		logger.Error("failed to start chat stream",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
			slog.String("requested_model", req.Model),
			slog.String("provider", h.provider.Name()),
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
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	var builder strings.Builder
	streamID := "chatcmpl-" + uuid.New().String()
	created := time.Now().Unix()

	// Stream metadata for encoding chunks
	metadata := &codec.StreamMetadata{
		ID:      streamID,
		Model:   req.Model,
		Created: created,
	}

	for event := range events {
		if event.Error != nil {
			logger.Error("stream event error",
				slog.String("request_id", requestID),
				slog.String("error", event.Error.Error()),
			)
			server.AddError(r.Context(), event.Error)
			break
		}

		builder.WriteString(event.ContentDelta)

		// Use codec to encode stream chunk
		data, err := h.codec.EncodeStreamChunk(&event, metadata)
		if err != nil {
			logger.Error("failed to encode stream chunk",
				slog.String("request_id", requestID),
				slog.String("error", err.Error()),
			)
			break
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	recordMetadata := map[string]string{
		"frontdoor": "openai",
		"app":       h.appName,
		"provider":  h.provider.Name(),
		"stream":    "true",
	}

	conversation.Record(r.Context(), h.store, streamID, req, &domain.CanonicalResponse{
		ID:    streamID,
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
	}, recordMetadata)

	logger.Info("chat stream completed",
		slog.String("frontdoor", "openai"),
		slog.String("app", h.appName),
		slog.String("provider", h.provider.Name()),
		slog.String("requested_model", req.Model),
		slog.String("served_model", req.Model),
	)
}
