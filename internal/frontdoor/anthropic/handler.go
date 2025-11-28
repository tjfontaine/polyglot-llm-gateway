package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	anthropicapi "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/codec"
	anthropiccodec "github.com/tjfontaine/polyglot-llm-gateway/internal/codec/anthropic"
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
		codec:    anthropiccodec.New(),
	}
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)
	providerName := h.provider.Name()

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
	canonReq, err := h.codec.DecodeRequest(body)
	if err != nil {
		logger.Error("failed to decode anthropic messages request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Capture User-Agent from incoming request and pass it through
	canonReq.UserAgent = r.Header.Get("User-Agent")

	// Set source API type and raw request for pass-through optimization
	canonReq.SourceAPIType = domain.APITypeAnthropic
	canonReq.RawRequest = body

	server.AddLogField(r.Context(), "requested_model", canonReq.Model)
	server.AddLogField(r.Context(), "frontdoor", "anthropic")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", providerName)

	if canonReq.Stream {
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

	// Use raw response if available (pass-through mode), otherwise encode
	var respBody []byte
	if len(resp.RawResponse) > 0 && resp.SourceAPIType == domain.APITypeAnthropic {
		// Pass-through: use raw response directly
		respBody = resp.RawResponse
		logger.Debug("using pass-through response",
			slog.String("request_id", requestID),
		)
	} else {
		// Standard path: encode canonical response to Anthropic format
		respBody, err = h.codec.EncodeResponse(resp)
		if err != nil {
			logger.Error("failed to encode response",
				slog.String("request_id", requestID),
				slog.String("error", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
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

// HandleCountTokens handles the /v1/messages/count_tokens endpoint.
// This endpoint supports:
// 1. Native pass-through for providers that support CountTokens (e.g., Anthropic)
// 2. Estimation for other providers via the token counter registry
func (h *Handler) HandleCountTokens(w http.ResponseWriter, r *http.Request) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(server.RequestIDKey).(string)

	server.AddLogField(r.Context(), "frontdoor", "anthropic")
	server.AddLogField(r.Context(), "app", h.appName)
	server.AddLogField(r.Context(), "provider", h.provider.Name())

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

	// Check if provider supports CountTokens natively
	type countTokensProvider interface {
		CountTokens(ctx context.Context, body []byte) ([]byte, error)
	}

	if ctp, ok := h.provider.(countTokensProvider); ok {
		// Pass through to native provider
		respBody, err := ctp.CountTokens(r.Context(), body)
		if err != nil {
			// Check if this is a "not supported" error - if so, fall back to estimation
			if strings.Contains(err.Error(), "count_tokens not supported") {
				logger.Debug("count_tokens not supported by provider, falling back to estimation",
					slog.String("request_id", requestID),
				)
				// Fall through to estimation below
			} else {
				// Real error from provider
				logger.Error("count_tokens failed",
					slog.String("request_id", requestID),
					slog.String("error", err.Error()),
				)
				server.AddError(r.Context(), err)
				writeAPIError(w, err)
				return
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write(respBody)
			return
		}
	}

	// Fallback: use estimation via codec
	canonReq, err := h.codec.DecodeRequest(body)
	if err != nil {
		logger.Error("failed to decode count_tokens request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use simple estimation based on message content
	inputTokens := h.estimateTokens(canonReq)

	// Return Anthropic-format response
	respBody := fmt.Sprintf(`{"input_tokens": %d}`, inputTokens)

	logger.Info("count_tokens estimated",
		slog.String("request_id", requestID),
		slog.String("model", canonReq.Model),
		slog.Int("input_tokens", inputTokens),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(respBody))
}

// estimateTokens provides a rough token estimate when native counting isn't available.
func (h *Handler) estimateTokens(req *domain.CanonicalRequest) int {
	totalChars := 0

	// Count system messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			totalChars += len(msg.Content)
		}
	}

	// Count all messages
	for _, msg := range req.Messages {
		totalChars += len(msg.Role)
		totalChars += len(msg.Content)
		// Add overhead for message formatting
		totalChars += 4
	}

	// Rough estimate: ~4 characters per token
	return (totalChars + 3) / 4
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

		// Use codec to encode stream chunk
		data, err := h.codec.EncodeStreamChunk(&event, nil)
		if err != nil {
			logger.Error("failed to encode stream chunk",
				slog.String("request_id", requestID),
				slog.String("error", err.Error()),
			)
			break
		}

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

// writeAPIError writes an error response with the appropriate HTTP status code.
// It detects Anthropic API errors and maps their error types to HTTP status codes.
func writeAPIError(w http.ResponseWriter, err error) {
	var apiErr *anthropicapi.APIError
	if errors.As(err, &apiErr) {
		statusCode := mapErrorTypeToStatus(apiErr.Type)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		// Return the error in Anthropic format
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    apiErr.Type,
				"message": apiErr.Message,
			},
		})
		return
	}

	// For non-API errors, return a generic error response
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// mapErrorTypeToStatus maps Anthropic error types to HTTP status codes.
func mapErrorTypeToStatus(errType string) int {
	switch errType {
	case "invalid_request_error":
		return http.StatusBadRequest // 400
	case "authentication_error":
		return http.StatusUnauthorized // 401
	case "permission_error":
		return http.StatusForbidden // 403
	case "not_found_error":
		return http.StatusNotFound // 404
	case "rate_limit_error":
		return http.StatusTooManyRequests // 429
	case "overloaded_error":
		return http.StatusServiceUnavailable // 503
	case "api_error":
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError // 500
	}
}
