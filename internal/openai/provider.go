// Provider implements the domain.Provider interface for the OpenAI API.
package openai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/conversation"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// ProviderOption configures the provider.
type ProviderOption func(*Provider)

// WithProviderBaseURL sets a custom base URL for the API.
func WithProviderBaseURL(baseURL string) ProviderOption {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

// extractThreadKey resolves a dotted JSON path from the raw request and returns a salted key.
// It returns ("", false) if no raw request is present or the path cannot be resolved to a string.
func (p *Provider) extractThreadKey(req *domain.CanonicalRequest) (string, bool) {
	// If no explicit path is configured, try canonical metadata (e.g., Anthropic CLI user_id)
	if p.threadKeyPath == "" {
		if uid := req.Metadata["user_id"]; uid != "" {
			h := sha256.Sum256([]byte(p.apiKey + ":" + uid))
			return fmt.Sprintf("%x", h[:]), true
		}
		return "", false
	}

	if len(req.RawRequest) == 0 {
		return "", false
	}

	var payload any
	if err := json.Unmarshal(req.RawRequest, &payload); err != nil {
		return "", false
	}

	parts := strings.Split(p.threadKeyPath, ".")
	cur := payload
	for _, part := range parts {
		obj, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		val, ok := obj[part]
		if !ok {
			return "", false
		}
		cur = val
	}

	strVal, ok := cur.(string)
	if !ok || strVal == "" {
		return "", false
	}

	h := sha256.Sum256([]byte(p.apiKey + ":" + strVal))
	return fmt.Sprintf("%x", h[:]), true
}

func (p *Provider) getThreadState(key string) string {
	if key == "" {
		return ""
	}

	p.stateMu.RLock()
	val := p.threadState[key]
	usePersistent := p.persistThreads
	store := p.threadStore
	p.stateMu.RUnlock()

	if val != "" {
		return val
	}

	if !usePersistent || store == nil {
		return ""
	}

	// Lazy-load from persistent store so restarts can resume threads
	stored, err := store.GetThreadState(key)
	if err != nil || stored == "" {
		return ""
	}

	p.stateMu.Lock()
	if p.threadState[key] == "" {
		p.threadState[key] = stored
	}
	p.stateMu.Unlock()

	return stored
}

func (p *Provider) setThreadState(key, responseID string) {
	if key == "" || responseID == "" {
		return
	}

	p.stateMu.Lock()
	p.threadState[key] = responseID
	store := p.threadStore
	usePersistent := p.persistThreads
	p.stateMu.Unlock()

	if usePersistent && store != nil {
		// Best-effort; avoid failing the request on persistence errors
		_ = store.SetThreadState(key, responseID)
	}
}

func (p *Provider) logEvent(ctx context.Context, interactionID string, evt *domain.InteractionEvent) {
	if p.eventStore == nil || evt == nil {
		return
	}
	if evt.InteractionID == "" {
		evt.InteractionID = interactionID
	}
	if evt.ID == "" {
		evt.ID = "evt_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	conversation.LogEvent(ctx, p.eventStore, evt)
}

// WithProviderHTTPClient sets a custom HTTP client.
func WithProviderHTTPClient(httpClient *http.Client) ProviderOption {
	return func(p *Provider) {
		p.httpClient = httpClient
	}
}

// WithResponsesAPI enables using the Responses API instead of Chat Completions.
func WithResponsesAPI(enable bool) ProviderOption {
	return func(p *Provider) {
		p.useResponsesAPI = enable
	}
}

// WithResponsesThreadKeyPath enables optional thread reuse for Responses API requests.
// The key is extracted from the raw request via a dotted JSON path (e.g., "metadata.thread_id").
// Keys are salted with the provider API key before storage. If the path cannot be resolved,
// thread reuse is skipped.
func WithResponsesThreadKeyPath(path string) ProviderOption {
	return func(p *Provider) {
		p.threadKeyPath = path
	}
}

// WithResponsesThreadPersistence enables persistence of thread state when storage is available.
func WithResponsesThreadPersistence(enable bool) ProviderOption {
	return func(p *Provider) {
		p.persistThreads = enable
	}
}

// WithThreadStateStore attaches a persistent thread state store.
func WithThreadStateStore(store storage.ThreadStateStore) ProviderOption {
	return func(p *Provider) {
		p.threadStore = store
	}
}

// WithEventStore attaches an InteractionStore for audit logging.
func WithEventStore(store storage.InteractionStore) ProviderOption {
	return func(p *Provider) {
		p.eventStore = store
	}
}

// Provider implements the domain.Provider interface using our custom OpenAI client.
type Provider struct {
	client          *Client
	apiKey          string
	baseURL         string
	httpClient      *http.Client
	useResponsesAPI bool
	threadKeyPath   string
	threadState     map[string]string
	stateMu         sync.RWMutex
	persistThreads  bool
	threadStore     storage.ThreadStateStore
	eventStore      storage.InteractionStore
}

// NewProvider creates a new OpenAI provider.
func NewProvider(apiKey string, opts ...ProviderOption) *Provider {
	p := &Provider{
		apiKey:      apiKey,
		threadState: make(map[string]string),
	}

	for _, opt := range opts {
		opt(p)
	}

	// Build client options
	var clientOpts []ClientOption
	if p.baseURL != "" {
		clientOpts = append(clientOpts, WithBaseURL(p.baseURL))
	}
	if p.httpClient != nil {
		clientOpts = append(clientOpts, WithHTTPClient(p.httpClient))
	}

	p.client = NewClient(apiKey, clientOpts...)
	return p
}

// SetThreadStore attaches a persistent store for thread state.
func (p *Provider) SetThreadStore(store storage.ThreadStateStore) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	p.threadStore = store
}

// SetEventStore attaches an interaction event store for audit logging.
func (p *Provider) SetEventStore(store storage.InteractionStore) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	p.eventStore = store
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) APIType() domain.APIType {
	return domain.APITypeOpenAI
}

func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	opts := &RequestOptions{
		UserAgent: req.UserAgent,
	}

	if p.useResponsesAPI {
		return p.completeWithResponses(ctx, req, opts)
	}

	// Use codec to convert canonical request to API request
	apiReq := CanonicalToAPIRequest(req)

	// Marshal the request body for debugging visibility
	reqBody, marshalErr := json.Marshal(apiReq)

	resp, err := p.client.CreateChatCompletion(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	// Use codec to convert API response to canonical response
	canonResp := APIResponseToCanonical(resp)

	// Attach the provider request body if marshaling succeeded
	if marshalErr == nil {
		canonResp.ProviderRequestBody = reqBody
	}

	return canonResp, nil
}

func (p *Provider) completeWithResponses(ctx context.Context, req *domain.CanonicalRequest, opts *RequestOptions) (*domain.CanonicalResponse, error) {
	interactionID := req.Metadata["interaction_id"]
	if interactionID == "" {
		interactionID = "int_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	apiReq := canonicalToResponsesRequest(req)
	var threadKey string
	var previousID string

	if p.threadKeyPath != "" {
		if key, ok := p.extractThreadKey(req); ok {
			threadKey = key
			if prev := p.getThreadState(key); prev != "" {
				apiReq.PreviousResponseID = prev
				previousID = prev
			}
		}
	}

	// Marshal the request body for debugging visibility
	reqBody, marshalErr := json.Marshal(apiReq)

	if threadKey != "" {
		p.logEvent(ctx, interactionID, &domain.InteractionEvent{
			Stage:              "thread_resolve",
			Direction:          "internal",
			APIType:            domain.APITypeOpenAI,
			Provider:           p.Name(),
			ModelRequested:     req.Model,
			ThreadKey:          threadKey,
			PreviousResponseID: previousID,
			Raw:                reqBody,
		})
	}

	resp, err := p.client.CreateResponse(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	canonResp := responsesResponseToCanonical(resp)

	// Attach the provider request body if marshaling succeeded
	if marshalErr == nil {
		canonResp.ProviderRequestBody = reqBody
	}

	if p.threadKeyPath != "" {
		if key, ok := p.extractThreadKey(req); ok && canonResp.ID != "" {
			p.setThreadState(key, canonResp.ID)
			p.logEvent(ctx, interactionID, &domain.InteractionEvent{
				Stage:          "thread_update",
				Direction:      "internal",
				APIType:        domain.APITypeOpenAI,
				Provider:       p.Name(),
				ModelRequested: req.Model,
				ThreadKey:      key,
				Raw:            reqBody,
				Metadata:       []byte(fmt.Sprintf(`{"new_response_id":"%s"}`, canonResp.ID)),
			})
		}
	}

	p.logEvent(ctx, interactionID, &domain.InteractionEvent{
		Stage:          "provider_encode",
		Direction:      "egress",
		APIType:        domain.APITypeOpenAI,
		Provider:       p.Name(),
		ModelRequested: req.Model,
		Raw:            reqBody,
	})
	if canonResp.RawResponse != nil {
		p.logEvent(ctx, interactionID, &domain.InteractionEvent{
			Stage:          "provider_decode",
			Direction:      "ingress",
			APIType:        domain.APITypeOpenAI,
			Provider:       p.Name(),
			ModelRequested: req.Model,
			ModelServed:    canonResp.Model,
			ProviderModel:  canonResp.ProviderModel,
			Raw:            canonResp.RawResponse,
		})
	}

	return canonResp, nil
}

func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	opts := &RequestOptions{
		UserAgent: req.UserAgent,
	}

	if p.useResponsesAPI {
		return p.streamWithResponses(ctx, req, opts)
	}

	// Use codec to convert canonical request to API request
	apiReq := CanonicalToAPIRequest(req)

	stream, err := p.client.StreamChatCompletion(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)
		for result := range stream {
			if result.Err != nil {
				out <- domain.CanonicalEvent{Error: result.Err}
				return
			}

			// Use codec to convert API chunk to canonical event
			event := APIChunkToCanonical(result.Chunk)
			out <- *event
		}
	}()

	return out, nil
}

func (p *Provider) streamWithResponses(ctx context.Context, req *domain.CanonicalRequest, opts *RequestOptions) (<-chan domain.CanonicalEvent, error) {
	interactionID := req.Metadata["interaction_id"]
	if interactionID == "" {
		interactionID = "int_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	apiReq := canonicalToResponsesRequest(req)

	var previousID string
	if p.threadKeyPath != "" {
		if key, ok := p.extractThreadKey(req); ok {
			if prev := p.getThreadState(key); prev != "" {
				apiReq.PreviousResponseID = prev
				p.logEvent(ctx, interactionID, &domain.InteractionEvent{
					Stage:              "thread_resolve",
					Direction:          "internal",
					APIType:            domain.APITypeOpenAI,
					Provider:           p.Name(),
					ModelRequested:     req.Model,
					ThreadKey:          key,
					PreviousResponseID: prev,
				})
				previousID = prev
			}
		}
	}
	apiReq.Stream = true

	if reqBytes, err := json.Marshal(apiReq); err == nil {
		p.logEvent(ctx, interactionID, &domain.InteractionEvent{
			Stage:          "provider_encode",
			Direction:      "egress",
			APIType:        domain.APITypeOpenAI,
			Provider:       p.Name(),
			ModelRequested: req.Model,
			Raw:            reqBytes,
		})
	}

	stream, err := p.client.StreamResponse(ctx, apiReq, opts)
	if err != nil {
		return nil, err
	}

	out := make(chan domain.CanonicalEvent)
	go func() {
		defer close(out)
		var currentModel string
		var currentResponseID string
		var threadKey string
		if p.threadKeyPath != "" {
			if key, ok := p.extractThreadKey(req); ok {
				threadKey = key
			}
		}

		for result := range stream {
			if result.Err != nil {
				out <- domain.CanonicalEvent{Error: result.Err}
				return
			}

			event := result.Event
			if event == nil {
				continue
			}

			if len(event.Data) > 0 {
				p.logEvent(ctx, interactionID, &domain.InteractionEvent{
					Stage:          "provider_decode",
					Direction:      "ingress",
					APIType:        domain.APITypeOpenAI,
					Provider:       p.Name(),
					ModelRequested: req.Model,
					ModelServed:    currentModel,
					ProviderModel:  currentModel,
					Raw:            event.Data,
				})
			}

			// Parse different event types (per OpenAI Responses API Spec v1.1)
			switch event.Type {
			case "response.created":
				// Per spec: {"id": "resp_123", "created": 1732000000, "model": "gpt-5"}
				var data struct {
					ID    string `json:"id"`
					Model string `json:"model"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					currentModel = data.Model
					currentResponseID = data.ID
				}

			case "response.output_item.delta":
				// Per spec: {"item_index": 0, "delta": {"content": "Hello"}} or {"delta": {"arguments": "..."}}
				var data struct {
					ItemIndex int `json:"item_index"`
					Delta     struct {
						Content   string `json:"content"`
						Arguments string `json:"arguments"`
					} `json:"delta"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					if data.Delta.Content != "" {
						out <- domain.CanonicalEvent{
							ContentDelta: data.Delta.Content,
							Model:        currentModel,
							ResponseID:   currentResponseID,
						}
					}
					// TODO: Handle arguments delta for tool calls if needed
				}

			case "response.content_part.added":
				// Per OpenAI spec: A new content part (e.g., text block) has started
				// Event payload: {"part": {...}, "item_id": "...", "content_index": 0}
				// We don't need to emit anything for this, just acknowledge it
				continue

			case "response.content_part.done":
				// Per OpenAI spec: A content part is finished
				// Event payload: {"part": {...}, "item_id": "...", "content_index": 0}
				// We don't need to emit anything for this, just acknowledge it
				continue

			case "response.output_item.added":
				// Per OpenAI spec: A new item (e.g., message) has started
				// Event payload: {"item": {...}, "output_index": 0}
				// We don't need to emit anything for this, just acknowledge it
				continue

			case "response.output_item.done":
				// Per OpenAI spec: An item is fully generated and finalized
				// Event payload: {"item": {...}, "output_index": 0}
				// We don't need to emit anything for this, the content was already streamed
				continue

			case "response.done":
				// DEPRECATED: This event is not part of the OpenAI Responses API spec.
				// It was used by a previous incorrect implementation of our frontdoor.
				// Keep for backwards compatibility but prefer response.completed.
				var data struct {
					Usage *struct {
						InputTokens  int `json:"input_tokens"`
						OutputTokens int `json:"output_tokens"`
						TotalTokens  int `json:"total_tokens"`
					} `json:"usage"`
					FinishReason string `json:"finish_reason"`
					ID           string `json:"id"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					ev := domain.CanonicalEvent{
						Model:        currentModel,
						ResponseID:   currentResponseID,
						FinishReason: data.FinishReason,
					}
					if data.Usage != nil {
						ev.Usage = &domain.Usage{
							PromptTokens:     data.Usage.InputTokens,
							CompletionTokens: data.Usage.OutputTokens,
							TotalTokens:      data.Usage.TotalTokens,
						}
					}
					out <- ev

					if len(event.Data) > 0 {
						p.logEvent(ctx, interactionID, &domain.InteractionEvent{
							Stage:          "provider_decode",
							Direction:      "ingress",
							APIType:        domain.APITypeOpenAI,
							Provider:       p.Name(),
							ModelRequested: req.Model,
							ModelServed:    currentModel,
							ProviderModel:  currentModel,
							Raw:            event.Data,
						})
					}

					if threadKey != "" {
						id := currentResponseID
						if id == "" && data.ID != "" {
							id = data.ID
						}
						if id != "" {
							p.setThreadState(threadKey, id)
							p.logEvent(ctx, interactionID, &domain.InteractionEvent{
								Stage:              "thread_update",
								Direction:          "internal",
								APIType:            domain.APITypeOpenAI,
								Provider:           p.Name(),
								ModelRequested:     req.Model,
								ThreadKey:          threadKey,
								PreviousResponseID: previousID,
								Metadata:           []byte(fmt.Sprintf(`{"new_response_id":"%s"}`, id)),
							})
						}
					}
				}

			case "response.in_progress":
				// Per spec: emitted when generation begins, contains full response object
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					currentModel = data.Response.Model
					currentResponseID = data.Response.ID
				}

			case "response.text.delta":
				// Per OpenAI Responses API spec: Text Stream with delta containing text chunk
				// Event payload: {"delta": "text", "item_id": "...", "content_index": 0}
				var data struct {
					Delta        string `json:"delta"`
					ItemID       string `json:"item_id"`
					ContentIndex int    `json:"content_index"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil && data.Delta != "" {
					out <- domain.CanonicalEvent{
						ContentDelta: data.Delta,
						Model:        currentModel,
						ResponseID:   currentResponseID,
					}
				}

			case "response.output_text.delta":
				// Legacy/alternative delta format
				var data struct {
					Delta string `json:"delta"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil && data.Delta != "" {
					out <- domain.CanonicalEvent{
						ContentDelta: data.Delta,
						Model:        currentModel,
						ResponseID:   currentResponseID,
					}
				}

			case "response.completed":
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					if data.Response.Usage != nil {
						out <- domain.CanonicalEvent{
							Model:      data.Response.Model,
							ResponseID: data.Response.ID,
							Usage: &domain.Usage{
								PromptTokens:     data.Response.Usage.InputTokens,
								CompletionTokens: data.Response.Usage.OutputTokens,
								TotalTokens:      data.Response.Usage.TotalTokens,
							},
						}
						if len(event.Data) > 0 {
							p.logEvent(ctx, interactionID, &domain.InteractionEvent{
								Stage:          "provider_decode",
								Direction:      "ingress",
								APIType:        domain.APITypeOpenAI,
								Provider:       p.Name(),
								ModelRequested: req.Model,
								ModelServed:    data.Response.Model,
								ProviderModel:  data.Response.Model,
								Raw:            event.Data,
							})
						}
						if threadKey != "" && data.Response.ID != "" {
							p.setThreadState(threadKey, data.Response.ID)
							p.logEvent(ctx, interactionID, &domain.InteractionEvent{
								Stage:              "thread_update",
								Direction:          "internal",
								APIType:            domain.APITypeOpenAI,
								Provider:           p.Name(),
								ModelRequested:     req.Model,
								ThreadKey:          threadKey,
								PreviousResponseID: previousID,
								Metadata:           []byte(fmt.Sprintf(`{"new_response_id":"%s"}`, data.Response.ID)),
							})
						}
					}
				}

			case "error":
				// Per OpenAI spec: {"type": "error", "error": {"type": "...", "code": "...", "message": "..."}}
				var data struct {
					Error struct {
						Type    string `json:"type"`
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"error"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					out <- domain.CanonicalEvent{
						Error: fmt.Errorf("%s: %s", data.Error.Code, data.Error.Message),
					}
					return
				}

			case "response.failed":
				// Per OpenAI spec: Emitted if an error occurs, contains the response with error details
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					errMsg := "response failed"
					if data.Response.Error != nil {
						errMsg = fmt.Sprintf("%s: %s", data.Response.Error.Code, data.Response.Error.Message)
					}
					out <- domain.CanonicalEvent{
						Error:      errors.New(errMsg),
						Model:      data.Response.Model,
						ResponseID: data.Response.ID,
					}
					return
				}

			case "response.incomplete":
				// Per OpenAI spec: Emitted if stopped early (e.g., max_tokens reached)
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					ev := domain.CanonicalEvent{
						Model:        data.Response.Model,
						ResponseID:   data.Response.ID,
						FinishReason: "length", // incomplete usually means max_tokens
					}
					if data.Response.Usage != nil {
						ev.Usage = &domain.Usage{
							PromptTokens:     data.Response.Usage.InputTokens,
							CompletionTokens: data.Response.Usage.OutputTokens,
							TotalTokens:      data.Response.Usage.TotalTokens,
						}
					}
					out <- ev
				}

			case "response.cancelled":
				// Per OpenAI spec: Emitted if the response was cancelled
				var data struct {
					Response ResponsesResponse `json:"response"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					out <- domain.CanonicalEvent{
						Model:        data.Response.Model,
						ResponseID:   data.Response.ID,
						FinishReason: "cancelled",
					}
				}

			case "response.function_call_arguments.delta":
				// Per OpenAI spec: Streaming JSON arguments for a function call
				var data struct {
					Delta  string `json:"delta"`
					ItemID string `json:"item_id"`
					CallID string `json:"call_id"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil && data.Delta != "" {
					tc := &domain.ToolCallChunk{ID: data.CallID}
					tc.Function.Arguments = data.Delta
					out <- domain.CanonicalEvent{
						Type:       domain.EventTypeContentBlockDelta,
						ToolCall:   tc,
						Model:      currentModel,
						ResponseID: currentResponseID,
					}
				}

			case "response.function_call_arguments.done":
				// Per OpenAI spec: Finalized arguments string for a function call
				var data struct {
					Arguments string `json:"arguments"`
					ItemID    string `json:"item_id"`
					CallID    string `json:"call_id"`
				}
				if err := json.Unmarshal(event.Data, &data); err == nil {
					tc := &domain.ToolCallChunk{ID: data.CallID}
					tc.Function.Arguments = data.Arguments
					out <- domain.CanonicalEvent{
						Type:       domain.EventTypeContentBlockStop,
						ToolCall:   tc,
						Model:      currentModel,
						ResponseID: currentResponseID,
					}
				}

			case "response.audio.delta":
				// Per OpenAI spec: Audio stream with base64 audio chunks
				// We don't currently support audio streaming, just acknowledge
				continue

			case "response.audio_transcript.delta":
				// Per OpenAI spec: Live transcript of generated audio
				// We don't currently support audio transcripts, just acknowledge
				continue

			case "response.audio.done":
				// Per OpenAI spec: Audio generation complete
				continue

			case "response.audio_transcript.done":
				// Per OpenAI spec: Audio transcript complete
				continue
			}
		}
	}()

	return out, nil
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	// ListModels doesn't need user agent passthrough as it's typically internal
	resp, err := p.client.ListModels(ctx, nil)
	if err != nil {
		return nil, err
	}

	models := make([]domain.Model, len(resp.Data))
	for i, m := range resp.Data {
		models[i] = domain.Model{
			ID:      m.ID,
			Object:  m.Object,
			OwnedBy: m.OwnedBy,
			Created: m.Created,
		}
	}

	return &domain.ModelList{
		Object: resp.Object,
		Data:   models,
	}, nil
}

// canonicalToResponsesRequest converts a canonical request to OpenAI Responses API format.
func canonicalToResponsesRequest(req *domain.CanonicalRequest) *ResponsesRequest {
	// Build input items from messages
	var input any

	// Build input as simple text when possible; otherwise use message items with input_text content parts
	if len(req.Messages) == 1 && req.Messages[0].Role == "user" && req.SystemPrompt == "" && req.Instructions == "" {
		input = req.Messages[0].Content
	} else {
		items := make([]ResponsesInputItem, 0, len(req.Messages))
		for _, msg := range req.Messages {
			contentType := "input_text"
			if msg.Role == "assistant" {
				contentType = "output_text"
			}
			item := ResponsesInputItem{
				Type:    "message",
				Role:    msg.Role,
				Content: []ResponsesContentPart{{Type: contentType, Text: msg.Content}},
			}
			items = append(items, item)
		}
		input = items
	}

	apiReq := &ResponsesRequest{
		Model: req.Model,
		Input: input,
	}

	// Set instructions from system prompt
	if req.Instructions != "" {
		apiReq.Instructions = req.Instructions
	} else if req.SystemPrompt != "" {
		apiReq.Instructions = req.SystemPrompt
	}

	if req.MaxTokens > 0 {
		apiReq.MaxOutputTokens = req.MaxTokens
	}

	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}

	if req.TopP > 0 {
		apiReq.TopP = &req.TopP
	}

	apiReq.ToolChoice = req.ToolChoice

	// Convert tools
	if len(req.Tools) > 0 {
		apiReq.Tools = make([]ResponsesTool, len(req.Tools))
		for i, t := range req.Tools {
			name := t.Name
			if name == "" {
				name = t.Function.Name
			}
			apiReq.Tools[i] = ResponsesTool{
				Name: name,
				Type: t.Type,
				Function: FunctionTool{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	return apiReq
}

// responsesResponseToCanonical converts an OpenAI Responses API response to canonical format.
func responsesResponseToCanonical(resp *ResponsesResponse) *domain.CanonicalResponse {
	choices := make([]domain.Choice, 0)

	for _, item := range resp.Output {
		if item.Type == "message" {
			var content string
			for _, part := range item.Content {
				if part.Type == "output_text" {
					content += part.Text
				}
			}

			choices = append(choices, domain.Choice{
				Index: len(choices),
				Message: domain.Message{
					Role:    item.Role,
					Content: content,
				},
				FinishReason: "stop",
			})
		} else if item.Type == "function_call" {
			// Handle function calls
			if len(choices) == 0 {
				choices = append(choices, domain.Choice{
					Index: 0,
					Message: domain.Message{
						Role: "assistant",
					},
					FinishReason: "tool_calls",
				})
			}

			choices[0].Message.ToolCalls = append(choices[0].Message.ToolCalls, domain.ToolCall{
				ID:   item.CallID,
				Type: "function",
				Function: domain.ToolCallFunction{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		}
	}

	// Ensure we have at least one choice
	if len(choices) == 0 {
		choices = append(choices, domain.Choice{
			Index:        0,
			Message:      domain.Message{Role: "assistant"},
			FinishReason: "stop",
		})
	}

	canonResp := &domain.CanonicalResponse{
		ID:            resp.ID,
		Object:        "chat.completion",
		Created:       resp.CreatedAt,
		Model:         resp.Model,
		Choices:       choices,
		SourceAPIType: domain.APITypeOpenAI,
		RawResponse:   resp.RawBody,
	}

	if resp.Usage != nil {
		canonResp.Usage = domain.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return canonResp
}
