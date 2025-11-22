package openai

import (
	"encoding/json"
	"net/http"

	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
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

	resp, err := h.provider.Complete(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
