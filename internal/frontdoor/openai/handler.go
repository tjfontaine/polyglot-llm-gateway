package openai

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

type Handler struct {
	provider domain.Provider
}

func NewHandler(provider domain.Provider) *Handler {
	return &Handler{provider: provider}
}

func (h *Handler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	var req domain.CanonicalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

	for event := range events {
		if event.Error != nil {
			// In a real stream, we might send a specific error event or just log it
			// and close the stream. For now, we'll just log (or ignore) and break.
			// Sending an error in the middle of a stream is tricky.
			break
		}

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
}
