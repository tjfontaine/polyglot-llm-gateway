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

	"github.com/tjfontaine/polyglot-llm-gateway/internal/codec"
	anthropiccodec "github.com/tjfontaine/polyglot-llm-gateway/internal/codec/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/conversation"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/server"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tokens"
)

type Handler struct {
	provider     domain.Provider
	store        storage.ConversationStore
	appName      string
	models       []domain.Model
	codec        codec.Codec
	tokenCounter *tokens.Registry
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

	// Set up token counter registry with OpenAI tiktoken support
	tokenRegistry := tokens.NewRegistry()
	tokenRegistry.Register(tokens.NewOpenAICounter())

	return &Handler{
		provider:     provider,
		store:        store,
		appName:      appName,
		models:       exposedModels,
		codec:        anthropiccodec.New(),
		tokenCounter: tokenRegistry,
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
		codec.WriteError(w, err, domain.APITypeAnthropic)
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

	// Set rate limit info in context for middleware to write as headers
	if resp.RateLimits != nil {
		ctx := server.SetRateLimits(r.Context(), &server.RateLimitInfo{
			RequestsLimit:     resp.RateLimits.RequestsLimit,
			RequestsRemaining: resp.RateLimits.RequestsRemaining,
			RequestsReset:     resp.RateLimits.RequestsReset,
			TokensLimit:       resp.RateLimits.TokensLimit,
			TokensRemaining:   resp.RateLimits.TokensRemaining,
			TokensReset:       resp.RateLimits.TokensReset,
		})
		r = r.WithContext(ctx)
	}

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

	// Write rate limit headers directly (since middleware already executed)
	h.writeRateLimitHeaders(w, resp.RateLimits)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// writeRateLimitHeaders writes normalized rate limit headers from the provider response.
func (h *Handler) writeRateLimitHeaders(w http.ResponseWriter, rl *domain.RateLimitInfo) {
	if rl == nil {
		return
	}

	if rl.RequestsLimit > 0 {
		w.Header().Set("x-ratelimit-limit-requests", fmt.Sprintf("%d", rl.RequestsLimit))
	}
	if rl.RequestsRemaining > 0 {
		w.Header().Set("x-ratelimit-remaining-requests", fmt.Sprintf("%d", rl.RequestsRemaining))
	}
	if rl.RequestsReset != "" {
		w.Header().Set("x-ratelimit-reset-requests", rl.RequestsReset)
	}
	if rl.TokensLimit > 0 {
		w.Header().Set("x-ratelimit-limit-tokens", fmt.Sprintf("%d", rl.TokensLimit))
	}
	if rl.TokensRemaining > 0 {
		w.Header().Set("x-ratelimit-remaining-tokens", fmt.Sprintf("%d", rl.TokensRemaining))
	}
	if rl.TokensReset != "" {
		w.Header().Set("x-ratelimit-reset-tokens", rl.TokensReset)
	}
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
				codec.WriteError(w, err, domain.APITypeAnthropic)
				return
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write(respBody)
			return
		}
	}

	// Fallback: use token counter registry (with tiktoken for OpenAI models)
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

	// Convert to token count request
	tokenReq := h.canonicalToTokenRequest(canonReq)

	// Use token counter registry (will use tiktoken for OpenAI models, estimation for others)
	tokenResp, err := h.tokenCounter.CountTokens(r.Context(), tokenReq)
	if err != nil {
		logger.Error("token counting failed",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		server.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return Anthropic-format response
	respBody := fmt.Sprintf(`{"input_tokens": %d}`, tokenResp.InputTokens)

	logMethod := "count_tokens"
	if tokenResp.Estimated {
		logMethod = "count_tokens (estimated)"
	}
	logger.Info(logMethod,
		slog.String("request_id", requestID),
		slog.String("model", canonReq.Model),
		slog.Int("input_tokens", tokenResp.InputTokens),
		slog.Bool("estimated", tokenResp.Estimated),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(respBody))
}

// canonicalToTokenRequest converts a canonical request to a token count request.
func (h *Handler) canonicalToTokenRequest(req *domain.CanonicalRequest) *domain.TokenCountRequest {
	tokenReq := &domain.TokenCountRequest{
		Model:    req.Model,
		Messages: req.Messages,
	}

	// Extract system message if present
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			tokenReq.System = msg.Content
			break
		}
	}

	return tokenReq
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
		codec.WriteError(w, err, domain.APITypeAnthropic)
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
	var servedModel string

	for event := range events {
		if event.Error != nil {
			// Context cancellation is expected when client disconnects - log at info level
			if errors.Is(event.Error, context.Canceled) {
				logger.Info("stream canceled by client",
					slog.String("request_id", requestID),
				)
			} else {
				logger.Error("stream event error",
					slog.String("request_id", requestID),
					slog.String("error", event.Error.Error()),
				)
			}
			break
		}

		// Capture served model from streaming events
		if event.Model != "" {
			servedModel = event.Model
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

	// Use served model from provider if available, otherwise use requested model
	if servedModel == "" {
		servedModel = req.Model
	}

	metadata := map[string]string{
		"frontdoor": "anthropic",
		"app":       h.appName,
		"provider":  providerName,
		"stream":    "true",
	}

	recordResp := &domain.CanonicalResponse{
		Model: servedModel,
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

	server.AddLogField(r.Context(), "served_model", servedModel)

	conversation.Record(r.Context(), h.store, "", req, recordResp, metadata)

	logger.Info("messages stream completed",
		slog.String("request_id", requestID),
		slog.String("frontdoor", "anthropic"),
		slog.String("app", h.appName),
		slog.String("provider", providerName),
		slog.String("requested_model", req.Model),
		slog.String("served_model", servedModel),
	)
}
