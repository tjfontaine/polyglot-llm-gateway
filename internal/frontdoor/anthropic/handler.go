package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tjfontaine/poly-llm-gateway/internal/domain"
)

type Handler struct {
	provider domain.Provider
}

func NewHandler(provider domain.Provider) *Handler {
	return &Handler{provider: provider}
}

// Anthropic Messages API request format
type MessagesRequest struct {
	Model     string          `json:"model"`
	Messages  []Message       `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream,omitempty"`
	System    []SystemMessage `json:"system,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type SystemMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
	var req MessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to CanonicalRequest
	canonReq := &domain.CanonicalRequest{
		Model:     req.Model,
		Messages:  make([]domain.Message, 0, len(req.Messages)+len(req.System)),
		Stream:    req.Stream,
		MaxTokens: req.MaxTokens,
	}

	// Add system messages first
	for _, sys := range req.System {
		canonReq.Messages = append(canonReq.Messages, domain.Message{
			Role:    "system",
			Content: sys.Text,
		})
	}

	// Add conversation messages
	for _, msg := range req.Messages {
		canonReq.Messages = append(canonReq.Messages, domain.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if req.Stream {
		h.handleStream(w, r, canonReq)
		return
	}

	resp, err := h.provider.Complete(r.Context(), canonReq)
	if err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthropicResp)
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
			break
		}

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
}
