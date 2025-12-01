package responses

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

var errNotFound = errors.New("not found")

// mockProvider implements domain.Provider for testing
type mockProvider struct {
	completeFunc func(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error)
	streamFunc   func(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error)
}

func (m *mockProvider) Name() string              { return "mock" }
func (m *mockProvider) APIType() domain.APIType   { return domain.APITypeOpenAI }
func (m *mockProvider) SupportedModels() []string { return []string{"gpt-4"} }
func (m *mockProvider) SupportsStreaming() bool   { return true }
func (m *mockProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{}, nil
}
func (m *mockProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return nil, nil
}
func (m *mockProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}
	return nil, nil
}

// mockStore implements ports.InteractionStore for testing
type mockStore struct {
	interactions  map[string]*domain.Interaction
	conversations map[string]*ports.Conversation
}

func newMockStore() *mockStore {
	return &mockStore{
		interactions:  make(map[string]*domain.Interaction),
		conversations: make(map[string]*ports.Conversation),
	}
}

// ConversationStore methods
func (m *mockStore) CreateConversation(ctx context.Context, conv *ports.Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}
func (m *mockStore) GetConversation(ctx context.Context, id string) (*ports.Conversation, error) {
	if c, ok := m.conversations[id]; ok {
		return c, nil
	}
	return nil, errNotFound
}
func (m *mockStore) ListConversations(ctx context.Context, opts ports.ListOptions) ([]*ports.Conversation, error) {
	return nil, nil
}
func (m *mockStore) AddMessage(ctx context.Context, convID string, msg *ports.StoredMessage) error {
	if c, ok := m.conversations[convID]; ok {
		c.Messages = append(c.Messages, *msg)
	}
	return nil
}
func (m *mockStore) DeleteConversation(ctx context.Context, id string) error {
	delete(m.conversations, id)
	return nil
}
func (m *mockStore) Close() error {
	return nil
}

// InteractionStore methods
func (m *mockStore) SaveInteraction(ctx context.Context, interaction *domain.Interaction) error {
	m.interactions[interaction.ID] = interaction
	return nil
}
func (m *mockStore) GetInteraction(ctx context.Context, id string) (*domain.Interaction, error) {
	if i, ok := m.interactions[id]; ok {
		return i, nil
	}
	return nil, errNotFound
}
func (m *mockStore) GetInteractionByProviderResponseID(ctx context.Context, providerResponseID string) (*domain.Interaction, error) {
	for _, i := range m.interactions {
		if i.Response != nil && i.Response.ProviderResponseID == providerResponseID {
			return i, nil
		}
	}
	return nil, errNotFound
}
func (m *mockStore) UpdateInteraction(ctx context.Context, interaction *domain.Interaction) error {
	interaction.UpdatedAt = time.Now()
	m.interactions[interaction.ID] = interaction
	return nil
}
func (m *mockStore) ListInteractions(ctx context.Context, opts ports.InteractionListOptions) ([]*domain.InteractionSummary, error) {
	return nil, nil
}
func (m *mockStore) AppendInteractionEvent(ctx context.Context, event *domain.InteractionEvent) error {
	return nil
}
func (m *mockStore) ListInteractionEvents(ctx context.Context, interactionID string, opts ports.InteractionListOptions) ([]*domain.InteractionEvent, error) {
	return nil, nil
}
func (m *mockStore) SetThreadState(threadKey, responseID string) error {
	return nil
}
func (m *mockStore) GetThreadState(threadKey string) (string, error) {
	return "", errNotFound
}

func TestHandleCreateResponse_NonStreaming(t *testing.T) {
	provider := &mockProvider{
		completeFunc: func(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
			return &domain.CanonicalResponse{
				ID:    "cmpl_123",
				Model: "gpt-4",
				Choices: []domain.Choice{{
					Index: 0,
					Message: domain.Message{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				}},
				Usage: domain.Usage{
					PromptTokens:     10,
					CompletionTokens: 8,
					TotalTokens:      18,
				},
			}, nil
		},
	}

	handler := NewHandler(newMockStore(), provider)

	reqBody := `{"model": "gpt-4", "input": "Hello"}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleCreateResponse(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp domain.ResponsesAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", resp.Status)
	}
	if len(resp.Output) == 0 {
		t.Fatal("Expected at least one output item")
	}
}

func TestHandleCreateResponse_Streaming_TextContent(t *testing.T) {
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
			ch := make(chan domain.CanonicalEvent, 10)
			go func() {
				defer close(ch)
				ch <- domain.CanonicalEvent{ContentDelta: "Hello"}
				ch <- domain.CanonicalEvent{ContentDelta: " world"}
				ch <- domain.CanonicalEvent{
					FinishReason: "stop",
					Usage: &domain.Usage{
						PromptTokens:     5,
						CompletionTokens: 2,
						TotalTokens:      7,
					},
				}
			}()
			return ch, nil
		},
	}

	handler := NewHandler(newMockStore(), provider)

	reqBody := `{"model": "gpt-4", "input": "Hi", "stream": true}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleCreateResponse(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Parse SSE events
	events := parseSSEEvents(t, rec.Body.String())

	// Per spec: response.created, response.output_item.added, response.output_item.delta, response.output_item.done, response.done
	hasCreated := false
	hasDone := false
	hasDelta := false
	var doneEvent domain.ResponseDoneEvent

	for _, e := range events {
		if e.EventType == "response.created" {
			hasCreated = true
		}
		if e.EventType == "response.output_item.delta" {
			hasDelta = true
		}
		if e.EventType == "response.done" {
			hasDone = true
			if err := json.Unmarshal([]byte(e.Data), &doneEvent); err != nil {
				t.Errorf("Failed to parse response.done event: %v", err)
			}
		}
	}

	if !hasCreated {
		t.Error("Expected response.created event")
	}
	if !hasDelta {
		t.Error("Expected response.output_item.delta event")
	}
	if !hasDone {
		t.Error("Expected response.done event")
	}
	if doneEvent.FinishReason != "stop" {
		t.Errorf("Expected finish_reason 'stop', got %s", doneEvent.FinishReason)
	}
}

func TestHandleCreateResponse_Streaming_ToolCalls(t *testing.T) {
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
			ch := make(chan domain.CanonicalEvent, 10)
			go func() {
				defer close(ch)
				// Tool call start
				tc1 := domain.ToolCallChunk{Index: 0, ID: "call_123"}
				tc1.Function.Name = "get_weather"
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockStart,
					ToolCall: &tc1,
				}
				// Tool call arguments delta
				tc2 := domain.ToolCallChunk{Index: 0}
				tc2.Function.Arguments = `{"location":`
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockDelta,
					ToolCall: &tc2,
				}
				tc3 := domain.ToolCallChunk{Index: 0}
				tc3.Function.Arguments = `"SF"}`
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockDelta,
					ToolCall: &tc3,
				}
				// Tool call stop
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockStop,
					ToolCall: &domain.ToolCallChunk{Index: 0},
				}
				// Final event
				ch <- domain.CanonicalEvent{
					FinishReason: "tool_calls",
					Usage: &domain.Usage{
						PromptTokens:     20,
						CompletionTokens: 15,
						TotalTokens:      35,
					},
				}
			}()
			return ch, nil
		},
	}

	handler := NewHandler(newMockStore(), provider)

	reqBody := `{"model": "gpt-4", "input": "What's the weather?", "stream": true}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleCreateResponse(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	events := parseSSEEvents(t, rec.Body.String())

	// Check for spec-compliant events
	hasOutputItemAdded := false
	hasArgumentsDelta := false
	hasOutputItemDone := false
	var doneEvent domain.ResponseDoneEvent

	for _, e := range events {
		if e.EventType == "response.output_item.added" {
			var event domain.OutputItemAddedEvent
			if err := json.Unmarshal([]byte(e.Data), &event); err == nil {
				if event.Item.Type == "function_call" {
					hasOutputItemAdded = true
					if event.Item.Name != "get_weather" {
						t.Errorf("Expected function name 'get_weather', got %s", event.Item.Name)
					}
					if event.Item.CallID != "call_123" {
						t.Errorf("Expected call_id 'call_123', got %s", event.Item.CallID)
					}
				}
			}
		}
		if e.EventType == "response.output_item.delta" {
			var event domain.OutputItemDeltaEvent
			if err := json.Unmarshal([]byte(e.Data), &event); err == nil {
				if event.Delta.Arguments != "" {
					hasArgumentsDelta = true
				}
			}
		}
		if e.EventType == "response.output_item.done" {
			var event domain.OutputItemDoneEvent
			if err := json.Unmarshal([]byte(e.Data), &event); err == nil {
				if event.Item.Type == "function_call" {
					hasOutputItemDone = true
					if event.Item.Arguments != `{"location":"SF"}` {
						t.Errorf("Expected arguments '{\"location\":\"SF\"}', got %s", event.Item.Arguments)
					}
				}
			}
		}
		if e.EventType == "response.done" {
			if err := json.Unmarshal([]byte(e.Data), &doneEvent); err != nil {
				t.Errorf("Failed to parse response.done event: %v", err)
			}
		}
	}

	if !hasOutputItemAdded {
		t.Error("Expected response.output_item.added event for function_call")
	}
	if !hasArgumentsDelta {
		t.Error("Expected response.output_item.delta event with arguments")
	}
	if !hasOutputItemDone {
		t.Error("Expected response.output_item.done event for function_call")
	}

	// finish_reason should be "tool_calls" per spec
	if doneEvent.FinishReason != "tool_calls" {
		t.Errorf("Expected finish_reason 'tool_calls', got %s", doneEvent.FinishReason)
	}
}

func TestHandleCreateResponse_Streaming_MixedContent(t *testing.T) {
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
			ch := make(chan domain.CanonicalEvent, 20)
			go func() {
				defer close(ch)
				// First: text content
				ch <- domain.CanonicalEvent{ContentDelta: "Let me check "}
				ch <- domain.CanonicalEvent{ContentDelta: "the weather."}

				// Then: tool call
				tc1 := domain.ToolCallChunk{Index: 0, ID: "call_456"}
				tc1.Function.Name = "get_weather"
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockStart,
					ToolCall: &tc1,
				}
				tc2 := domain.ToolCallChunk{Index: 0}
				tc2.Function.Arguments = `{"city":"NYC"}`
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockDelta,
					ToolCall: &tc2,
				}
				ch <- domain.CanonicalEvent{
					Type:     domain.EventTypeContentBlockStop,
					ToolCall: &domain.ToolCallChunk{Index: 0},
				}
				ch <- domain.CanonicalEvent{
					FinishReason: "tool_calls",
				}
			}()
			return ch, nil
		},
	}

	handler := NewHandler(newMockStore(), provider)

	reqBody := `{"model": "gpt-4", "input": "What's the weather in NYC?", "stream": true}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleCreateResponse(rec, req)

	events := parseSSEEvents(t, rec.Body.String())

	// Should have delta events with content
	contentDeltaCount := 0
	for _, e := range events {
		if e.EventType == "response.output_item.delta" {
			var event domain.OutputItemDeltaEvent
			if err := json.Unmarshal([]byte(e.Data), &event); err == nil {
				if event.Delta.Content != "" {
					contentDeltaCount++
				}
			}
		}
	}

	if contentDeltaCount < 2 {
		t.Errorf("Expected at least 2 content delta events, got %d", contentDeltaCount)
	}

	// Check response.done has correct finish_reason
	var doneEvent domain.ResponseDoneEvent
	for _, e := range events {
		if e.EventType == "response.done" {
			json.Unmarshal([]byte(e.Data), &doneEvent)
		}
	}

	if doneEvent.FinishReason != "tool_calls" {
		t.Errorf("Expected finish_reason 'tool_calls', got %s", doneEvent.FinishReason)
	}
}

func TestHandleCreateResponse_NonStreaming_ToolCalls(t *testing.T) {
	provider := &mockProvider{
		completeFunc: func(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
			return &domain.CanonicalResponse{
				ID:    "cmpl_456",
				Model: "gpt-4",
				Choices: []domain.Choice{{
					Index: 0,
					Message: domain.Message{
						Role:    "assistant",
						Content: "I'll check the weather.",
						ToolCalls: []domain.ToolCall{{
							ID:   "call_789",
							Type: "function",
							Function: domain.ToolCallFunction{
								Name:      "get_weather",
								Arguments: `{"location":"Paris"}`,
							},
						}},
					},
					FinishReason: "tool_calls",
				}},
				Usage: domain.Usage{
					PromptTokens:     15,
					CompletionTokens: 20,
					TotalTokens:      35,
				},
			}, nil
		},
	}

	handler := NewHandler(newMockStore(), provider)

	reqBody := `{"model": "gpt-4", "input": "What's the weather in Paris?"}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleCreateResponse(rec, req)

	var resp domain.ResponsesAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Status should be "incomplete" for tool_calls
	if resp.Status != "incomplete" {
		t.Errorf("Expected status 'incomplete', got %s", resp.Status)
	}

	// Should have function_call in output
	hasFunctionCall := false
	for _, item := range resp.Output {
		if item.Type == "function_call" {
			hasFunctionCall = true
			if item.Name != "get_weather" {
				t.Errorf("Expected function name 'get_weather', got %s", item.Name)
			}
			if item.Arguments != `{"location":"Paris"}` {
				t.Errorf("Unexpected arguments: %s", item.Arguments)
			}
		}
	}

	if !hasFunctionCall {
		t.Error("Expected function_call in output")
	}
}

// sseEvent represents a parsed SSE event
type sseEvent struct {
	EventType string
	Data      string
}

// parseSSEEvents parses SSE events from a response body
func parseSSEEvents(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))

	var currentEvent sseEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent.EventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && currentEvent.EventType != "" {
			events = append(events, currentEvent)
			currentEvent = sseEvent{}
		}
	}

	// Don't forget the last event if there's no trailing newline
	if currentEvent.EventType != "" {
		events = append(events, currentEvent)
	}

	return events
}
