// Package openai provides a frontdoor handler for the OpenAI Chat Completions API format.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/middleware"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/conversation"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/codec"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/shadow"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// FrontdoorType is the frontdoor type identifier used in configuration.
const FrontdoorType = "openai"

// FrontdoorAPIType returns the canonical API type for this frontdoor.
func FrontdoorAPIType() domain.APIType {
	return domain.APITypeOpenAI
}

// FrontdoorRoute defines an HTTP route registration.
type FrontdoorRoute struct {
	Path    string
	Method  string
	Handler func(http.ResponseWriter, *http.Request)
}

// RegisterFrontdoor registers the OpenAI frontdoor factory.
func RegisterFrontdoor() {
	if frontdoor.IsRegistered(FrontdoorType) {
		return
	}

	frontdoor.RegisterFactory(frontdoor.FrontdoorFactory{
		Type:           FrontdoorType,
		APIType:        FrontdoorAPIType(),
		Description:    "OpenAI Chat Completions API format",
		CreateHandlers: createFrontdoorHandlers,
	})
}

// createFrontdoorHandlers creates handler registrations for OpenAI frontdoor.
func createFrontdoorHandlers(cfg frontdoor.HandlerConfig) []frontdoor.HandlerRegistration {
	handler := NewFrontdoorHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models, cfg.ShadowConfig)
	routes := CreateFrontdoorHandlerRegistrations(handler, cfg.BasePath)
	result := make([]frontdoor.HandlerRegistration, len(routes))
	for i, r := range routes {
		result[i] = frontdoor.HandlerRegistration{Path: r.Path, Method: r.Method, Handler: r.Handler}
	}
	return result
}

// CreateFrontdoorHandlerRegistrations creates the HTTP handler registrations for OpenAI frontdoor.
func CreateFrontdoorHandlerRegistrations(handler *FrontdoorHandler, basePath string) []FrontdoorRoute {
	return []FrontdoorRoute{
		{Path: basePath + "/v1/chat/completions", Method: http.MethodPost, Handler: handler.HandleChatCompletion},
		{Path: basePath + "/v1/models", Method: http.MethodGet, Handler: handler.HandleListModels},
	}
}

// FrontdoorHandler handles OpenAI Chat Completions API requests.
type FrontdoorHandler struct {
	provider     ports.Provider
	store        storage.ConversationStore
	appName      string
	models       []domain.Model
	codec        codec.Codec
	shadowConfig *config.ShadowConfig
}

// NewFrontdoorHandler creates a new OpenAI frontdoor handler.
func NewFrontdoorHandler(provider ports.Provider, store storage.ConversationStore, appName string, models []config.ModelListItem, shadowCfg *config.ShadowConfig) *FrontdoorHandler {
	exposedModels := make([]domain.Model, 0, len(models))
	for _, model := range models {
		exposedModels = append(exposedModels, domain.Model{
			ID:      model.ID,
			Object:  model.Object,
			OwnedBy: model.OwnedBy,
			Created: model.Created,
		})
	}

	return &FrontdoorHandler{
		provider:     provider,
		store:        store,
		appName:      appName,
		models:       exposedModels,
		codec:        NewCodec(),
		shadowConfig: shadowCfg,
	}
}

func (h *FrontdoorHandler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	logger := slog.Default()
	requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)
	providerName := h.provider.Name()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("failed to read request body",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
		)
		middleware.AddError(r.Context(), err)
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
		middleware.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Capture User-Agent from incoming request and pass it through
	req.UserAgent = r.Header.Get("User-Agent")

	// Preserve request metadata from raw payload if present
	if req.Metadata == nil {
		var rawMeta struct {
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(body, &rawMeta); err == nil && len(rawMeta.Metadata) > 0 {
			req.Metadata = rawMeta.Metadata
		}
	}
	if req.Metadata == nil {
		req.Metadata = map[string]string{}
	}

	interactionID := req.Metadata["interaction_id"]
	if interactionID == "" {
		if requestID != "" {
			interactionID = "int_" + strings.ReplaceAll(requestID, "-", "")
		} else {
			interactionID = "int_" + strings.ReplaceAll(uuid.New().String(), "-", "")
		}
		req.Metadata["interaction_id"] = interactionID
	}

	// Set source API type and raw request for pass-through optimization
	req.SourceAPIType = domain.APITypeOpenAI
	req.RawRequest = body

	middleware.AddLogField(r.Context(), "frontdoor", "openai")
	middleware.AddLogField(r.Context(), "app", h.appName)
	middleware.AddLogField(r.Context(), "provider", providerName)
	middleware.AddLogField(r.Context(), "requested_model", req.Model)

	if req.Stream {
		h.handleStream(w, r, req, body, startTime)
		return
	}

	resp, err := h.provider.Complete(r.Context(), req)
	if err != nil {
		logger.Error("chat completion failed",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
			slog.String("requested_model", req.Model),
			slog.String("provider", providerName),
		)
		middleware.AddError(r.Context(), err)

		// Record failed interaction
		conversation.RecordInteraction(r.Context(), conversation.RecordInteractionParams{
			Store:          h.store,
			RawRequest:     body,
			CanonicalReq:   req,
			RequestHeaders: r.Header,
			Frontdoor:      domain.APITypeOpenAI,
			Provider:       providerName,
			AppName:        h.appName,
			Error:          err,
			Duration:       time.Since(startTime),
		})

		codec.WriteError(w, err, domain.APITypeOpenAI)
		return
	}

	requestedModel := req.Model
	servedModel := resp.Model

	// Build log fields
	logFields := []any{
		slog.String("frontdoor", "openai"),
		slog.String("app", h.appName),
		slog.String("provider", providerName),
		slog.String("requested_model", requestedModel),
		slog.String("served_model", servedModel),
	}
	if resp.ProviderModel != "" && resp.ProviderModel != servedModel {
		logFields = append(logFields, slog.String("provider_model", resp.ProviderModel))
	}
	logger.Info("chat completion", logFields...)

	middleware.AddLogField(r.Context(), "served_model", servedModel)
	if resp.ProviderModel != "" {
		middleware.AddLogField(r.Context(), "provider_model", resp.ProviderModel)
	}

	// Use raw response if available (pass-through mode), otherwise encode
	var respBody []byte
	if len(resp.RawResponse) > 0 && resp.SourceAPIType == domain.APITypeOpenAI {
		// Pass-through: use raw response directly
		respBody = resp.RawResponse
		logger.Debug("using pass-through response",
			slog.String("request_id", requestID),
		)
	} else {
		// Standard path: encode canonical response to OpenAI format
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

	// Record successful interaction with full bidirectional visibility
	// Capture the actual interaction ID (which may be the provider's response ID)
	actualInteractionID := conversation.RecordInteraction(r.Context(), conversation.RecordInteractionParams{
		Store:               h.store,
		RawRequest:          body,
		CanonicalReq:        req,
		ProviderRequestBody: resp.ProviderRequestBody,
		RawResponse:         resp.RawResponse,
		CanonicalResp:       resp,
		ClientResponse:      respBody,
		RequestHeaders:      r.Header,
		Frontdoor:           domain.APITypeOpenAI,
		Provider:            providerName,
		AppName:             h.appName,
		Duration:            time.Since(startTime),
	})

	// Trigger shadow execution asynchronously (non-blocking)
	// Use the actual interaction ID returned from recording (not the early-generated one)
	shadow.TriggerGlobalShadow(
		r.Context(),
		h.shadowConfig,
		actualInteractionID,
		req,
		resp,
		domain.APITypeOpenAI,
	)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func (h *FrontdoorHandler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	middleware.AddLogField(r.Context(), "frontdoor", "openai")
	middleware.AddLogField(r.Context(), "app", h.appName)
	middleware.AddLogField(r.Context(), "provider", h.provider.Name())

	w.Header().Set("Content-Type", "application/json")

	if len(h.models) > 0 {
		json.NewEncoder(w).Encode(domain.ModelList{Object: "list", Data: h.models})
		return
	}

	list, err := h.provider.ListModels(r.Context())
	if err != nil {
		middleware.AddError(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if list.Object == "" {
		list.Object = "list"
	}

	json.NewEncoder(w).Encode(list)
}

func (h *FrontdoorHandler) handleStream(w http.ResponseWriter, r *http.Request, req *domain.CanonicalRequest, rawRequest json.RawMessage, startTime time.Time) {
	logger := slog.Default()
	requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)
	providerName := h.provider.Name()

	events, err := h.provider.Stream(r.Context(), req)
	if err != nil {
		logger.Error("failed to start chat stream",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()),
			slog.String("requested_model", req.Model),
			slog.String("provider", providerName),
		)
		middleware.AddError(r.Context(), err)

		// Record failed streaming interaction
		conversation.RecordInteraction(r.Context(), conversation.RecordInteractionParams{
			Store:          h.store,
			RawRequest:     rawRequest,
			CanonicalReq:   req,
			RequestHeaders: r.Header,
			Frontdoor:      domain.APITypeOpenAI,
			Provider:       providerName,
			AppName:        h.appName,
			Streaming:      true,
			Error:          err,
			Duration:       time.Since(startTime),
		})

		codec.WriteError(w, err, domain.APITypeOpenAI)
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
	var servedModel string
	var providerModel string
	var finishReason string
	var usage *domain.Usage
	var streamErr error
	streamID := "chatcmpl-" + uuid.New().String()
	created := time.Now().Unix()

	// Stream metadata for encoding chunks
	streamMeta := &codec.StreamMetadata{
		ID:      streamID,
		Model:   req.Model,
		Created: created,
	}

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
				middleware.AddError(r.Context(), event.Error)
				streamErr = event.Error
			}
			break
		}

		// Capture served model from streaming events
		if event.Model != "" {
			servedModel = event.Model
		}
		// Capture provider model (the actual model used, before any rewriting)
		if event.ProviderModel != "" {
			providerModel = event.ProviderModel
		}
		// Capture finish reason and usage
		if event.FinishReason != "" {
			finishReason = event.FinishReason
		}
		if event.Usage != nil {
			usage = event.Usage
		}

		builder.WriteString(event.ContentDelta)

		// Use codec to encode stream chunk
		data, err := h.codec.EncodeStreamChunk(&event, streamMeta)
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

	// Use served model from provider if available, otherwise use requested model
	if servedModel == "" {
		servedModel = req.Model
	}

	// Build the canonical response for recording
	recordResp := &domain.CanonicalResponse{
		ID:            streamID,
		Model:         servedModel,
		ProviderModel: providerModel,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    "assistant",
					Content: builder.String(),
				},
				FinishReason: finishReason,
			},
		},
	}
	if usage != nil {
		recordResp.Usage = *usage
	}

	middleware.AddLogField(r.Context(), "served_model", servedModel)
	if providerModel != "" {
		middleware.AddLogField(r.Context(), "provider_model", providerModel)
	}

	// Record streaming interaction
	conversation.RecordInteraction(r.Context(), conversation.RecordInteractionParams{
		Store:          h.store,
		RawRequest:     rawRequest,
		CanonicalReq:   req,
		CanonicalResp:  recordResp,
		RequestHeaders: r.Header,
		Frontdoor:      domain.APITypeOpenAI,
		Provider:       providerName,
		AppName:        h.appName,
		Streaming:      true,
		Error:          streamErr,
		Duration:       time.Since(startTime),
		FinishReason:   finishReason,
	})

	// Build log fields
	logFields := []any{
		slog.String("frontdoor", "openai"),
		slog.String("app", h.appName),
		slog.String("provider", providerName),
		slog.String("requested_model", req.Model),
		slog.String("served_model", servedModel),
	}
	if providerModel != "" && providerModel != servedModel {
		logFields = append(logFields, slog.String("provider_model", providerModel))
	}

	logger.Info("chat stream completed", logFields...)
}
