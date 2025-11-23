package openai

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/conversation"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

type Handler struct {
	provider domain.Provider
	store    storage.ConversationStore
	appName  string
}

func NewHandler(provider domain.Provider, store storage.ConversationStore, appName string) *Handler {
	return &Handler{
		provider: provider,
		store:    store,
		appName:  appName,
	}
}

func (h *Handler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	var req domain.CanonicalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	server.AddLogField(r.Context(), "frontdoor", "openai")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", h.provider.Name())
	server.AddLogField(r.Context(), "requested_model", req.Model)

	// For Phase 1, we assume the request is already in a format we can map directly
	// or we just use the CanonicalRequest struct as the target for decoding
	// since it's a superset.

	if req.Stream {
		h.handleStream(w, r, &req)
		return
	}

	resp, err := h.provider.Complete(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger := slog.Default()
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
	conversation.Record(r.Context(), h.store, resp.ID, &req, resp, metadata)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request, req *domain.CanonicalRequest) {
	events, err := h.provider.Stream(r.Context(), req)
	if err != nil {
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

	for event := range events {
		if event.Error != nil {
			// In a real stream, we might send a specific error event or just log it
			// and close the stream. For now, we'll just log (or ignore) and break.
			// Sending an error in the middle of a stream is tricky.
			break
		}

		builder.WriteString(event.ContentDelta)

		// Construct an OpenAI-compatible chunk
		chunk := struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			Model   string `json:"model"`
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}{
			ID:      "chatcmpl-123", // TODO: Generate ID
			Object:  "chat.completion.chunk",
			Created: 1234567890, // TODO: Use current time
			Model:   req.Model,
			Choices: []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Delta: struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					}{
						Role:    event.Role,
						Content: event.ContentDelta,
					},
					FinishReason: nil,
				},
			},
		}

		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	logger := slog.Default()

	metadata := map[string]string{
		"frontdoor": "openai",
		"app":       h.appName,
		"provider":  h.provider.Name(),
		"stream":    "true",
	}

	conversation.Record(r.Context(), h.store, "", req, &domain.CanonicalResponse{
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
	}, metadata)

	logger.Info("chat stream completed",
		slog.String("frontdoor", "openai"),
		slog.String("app", h.appName),
		slog.String("provider", h.provider.Name()),
		slog.String("requested_model", req.Model),
		slog.String("served_model", req.Model),
	)
}
