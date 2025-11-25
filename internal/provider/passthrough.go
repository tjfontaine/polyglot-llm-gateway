package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// PassthroughProvider wraps a provider and bypasses canonical conversion
// when the source API type matches the provider's API type.
type PassthroughProvider struct {
	inner      domain.Provider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// PassthroughOption configures the passthrough provider.
type PassthroughOption func(*PassthroughProvider)

// WithPassthroughAPIKey sets the API key for pass-through requests.
func WithPassthroughAPIKey(apiKey string) PassthroughOption {
	return func(p *PassthroughProvider) {
		p.apiKey = apiKey
	}
}

// WithPassthroughBaseURL sets the base URL for pass-through requests.
func WithPassthroughBaseURL(baseURL string) PassthroughOption {
	return func(p *PassthroughProvider) {
		p.baseURL = baseURL
	}
}

// WithPassthroughHTTPClient sets the HTTP client for pass-through requests.
func WithPassthroughHTTPClient(client *http.Client) PassthroughOption {
	return func(p *PassthroughProvider) {
		p.httpClient = client
	}
}

// NewPassthroughProvider creates a new pass-through provider wrapper.
func NewPassthroughProvider(inner domain.Provider, opts ...PassthroughOption) *PassthroughProvider {
	p := &PassthroughProvider{
		inner:      inner,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Set default base URLs based on provider type
	if p.baseURL == "" {
		switch inner.APIType() {
		case domain.APITypeOpenAI:
			p.baseURL = "https://api.openai.com/v1"
		case domain.APITypeAnthropic:
			p.baseURL = "https://api.anthropic.com/v1"
		}
	}

	return p
}

func (p *PassthroughProvider) Name() string {
	return p.inner.Name()
}

func (p *PassthroughProvider) APIType() domain.APIType {
	return p.inner.APIType()
}

// SupportsPassthrough returns true if the provider can handle raw requests
// from the given API type.
func (p *PassthroughProvider) SupportsPassthrough(sourceType domain.APIType) bool {
	return sourceType == p.inner.APIType() && len(p.apiKey) > 0
}

// Complete handles requests, using pass-through when possible.
func (p *PassthroughProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	// Check if we can use pass-through
	if p.SupportsPassthrough(req.SourceAPIType) && len(req.RawRequest) > 0 {
		rawResp, err := p.CompleteRaw(ctx, req)
		if err != nil {
			return nil, err
		}

		// Parse raw response into canonical format for recording
		resp, err := p.parseRawResponse(rawResp)
		if err != nil {
			return nil, err
		}
		resp.RawResponse = rawResp
		resp.SourceAPIType = p.inner.APIType()
		return resp, nil
	}

	// Fall back to canonical conversion
	return p.inner.Complete(ctx, req)
}

// CompleteRaw handles a raw request body and returns a raw response body.
func (p *PassthroughProvider) CompleteRaw(ctx context.Context, req *domain.CanonicalRequest) ([]byte, error) {
	var endpoint string
	var headers map[string]string

	switch p.inner.APIType() {
	case domain.APITypeOpenAI:
		endpoint = p.baseURL + "/chat/completions"
		headers = map[string]string{
			"Authorization": "Bearer " + p.apiKey,
			"Content-Type":  "application/json",
		}
	case domain.APITypeAnthropic:
		endpoint = p.baseURL + "/messages"
		headers = map[string]string{
			"x-api-key":         p.apiKey,
			"anthropic-version": "2023-06-01",
			"Content-Type":      "application/json",
		}
	default:
		return nil, fmt.Errorf("unsupported API type for passthrough: %s", p.inner.APIType())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(req.RawRequest))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Stream handles streaming requests, using pass-through when possible.
func (p *PassthroughProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	// Check if we can use pass-through
	if p.SupportsPassthrough(req.SourceAPIType) && len(req.RawRequest) > 0 {
		return p.StreamRaw(ctx, req)
	}

	// Fall back to canonical conversion
	return p.inner.Stream(ctx, req)
}

// StreamRaw handles a raw streaming request and returns canonical events with raw data attached.
func (p *PassthroughProvider) StreamRaw(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	var endpoint string
	var headers map[string]string

	switch p.inner.APIType() {
	case domain.APITypeOpenAI:
		endpoint = p.baseURL + "/chat/completions"
		headers = map[string]string{
			"Authorization": "Bearer " + p.apiKey,
			"Content-Type":  "application/json",
		}
	case domain.APITypeAnthropic:
		endpoint = p.baseURL + "/messages"
		headers = map[string]string{
			"x-api-key":         p.apiKey,
			"anthropic-version": "2023-06-01",
			"Content-Type":      "application/json",
		}
	default:
		return nil, fmt.Errorf("unsupported API type for passthrough: %s", p.inner.APIType())
	}

	// Modify raw request to ensure streaming is enabled
	var rawReq map[string]interface{}
	if err := json.Unmarshal(req.RawRequest, &rawReq); err != nil {
		return nil, fmt.Errorf("failed to parse raw request: %w", err)
	}
	rawReq["stream"] = true
	modifiedReq, err := json.Marshal(rawReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(modifiedReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Return a channel that wraps the raw SSE stream
	out := make(chan domain.CanonicalEvent)
	go p.streamRawReader(resp.Body, out, p.inner.APIType())
	return out, nil
}

func (p *PassthroughProvider) streamRawReader(body io.ReadCloser, out chan<- domain.CanonicalEvent, apiType domain.APIType) {
	defer close(out)
	defer body.Close()

	buf := make([]byte, 4096)
	var eventType string
	var data []byte

	for {
		n, err := body.Read(buf)
		if n > 0 {
			// Parse SSE format
			lines := bytes.Split(buf[:n], []byte("\n"))
			for _, line := range lines {
				line = bytes.TrimSpace(line)
				if len(line) == 0 {
					// Empty line means end of event
					if len(data) > 0 {
						event := p.parseStreamEvent(eventType, data, apiType)
						if event.Error != nil || event.ContentDelta != "" || event.Usage != nil {
							event.RawEvent = data
							out <- event
						}
						data = nil
						eventType = ""
					}
					continue
				}

				if bytes.HasPrefix(line, []byte("event:")) {
					eventType = string(bytes.TrimSpace(bytes.TrimPrefix(line, []byte("event:"))))
				} else if bytes.HasPrefix(line, []byte("data:")) {
					d := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
					if string(d) == "[DONE]" {
						return
					}
					data = d
				}
			}
		}

		if err != nil {
			if err != io.EOF {
				out <- domain.CanonicalEvent{Error: fmt.Errorf("stream read error: %w", err)}
			}
			return
		}
	}
}

func (p *PassthroughProvider) parseStreamEvent(eventType string, data []byte, apiType domain.APIType) domain.CanonicalEvent {
	event := domain.CanonicalEvent{}

	switch apiType {
	case domain.APITypeOpenAI:
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
					Role    string `json:"role"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			event.Error = fmt.Errorf("failed to parse chunk: %w", err)
			return event
		}
		if len(chunk.Choices) > 0 {
			event.ContentDelta = chunk.Choices[0].Delta.Content
			event.Role = chunk.Choices[0].Delta.Role
		}
		if chunk.Usage != nil {
			event.Usage = &domain.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

	case domain.APITypeAnthropic:
		switch eventType {
		case "content_block_delta":
			var delta struct {
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(data, &delta); err == nil {
				event.ContentDelta = delta.Delta.Text
			}
		case "message_start":
			var start struct {
				Message struct {
					Role  string `json:"role"`
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal(data, &start); err == nil {
				event.Role = start.Message.Role
				event.Usage = &domain.Usage{
					PromptTokens: start.Message.Usage.InputTokens,
				}
			}
		case "message_delta":
			var delta struct {
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(data, &delta); err == nil && delta.Usage.OutputTokens > 0 {
				event.Usage = &domain.Usage{
					CompletionTokens: delta.Usage.OutputTokens,
				}
			}
		}
	}

	return event
}

func (p *PassthroughProvider) parseRawResponse(rawResp []byte) (*domain.CanonicalResponse, error) {
	resp := &domain.CanonicalResponse{}

	switch p.inner.APIType() {
	case domain.APITypeOpenAI:
		var apiResp struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			Model   string `json:"model"`
			Choices []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(rawResp, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		resp.ID = apiResp.ID
		resp.Object = apiResp.Object
		resp.Created = apiResp.Created
		resp.Model = apiResp.Model
		resp.Usage = domain.Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		}

		for _, c := range apiResp.Choices {
			resp.Choices = append(resp.Choices, domain.Choice{
				Index: c.Index,
				Message: domain.Message{
					Role:    c.Message.Role,
					Content: c.Message.Content,
				},
				FinishReason: c.FinishReason,
			})
		}

	case domain.APITypeAnthropic:
		var apiResp struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Role    string `json:"role"`
			Model   string `json:"model"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			StopReason string `json:"stop_reason"`
			Usage      struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(rawResp, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		var content string
		for _, c := range apiResp.Content {
			if c.Type == "text" {
				content += c.Text
			}
		}

		resp.ID = apiResp.ID
		resp.Object = "chat.completion"
		resp.Model = apiResp.Model
		resp.Usage = domain.Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		}
		resp.Choices = []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    apiResp.Role,
					Content: content,
				},
				FinishReason: apiResp.StopReason,
			},
		}
	}

	return resp, nil
}

func (p *PassthroughProvider) ListModels(ctx context.Context) (*domain.ModelList, error) {
	return p.inner.ListModels(ctx)
}

// Ensure PassthroughProvider implements the extended interface
var _ domain.PassthroughProvider = (*PassthroughProvider)(nil)
