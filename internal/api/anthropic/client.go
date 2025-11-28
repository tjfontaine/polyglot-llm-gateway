package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL = "https://api.anthropic.com"
	defaultVersion = "2023-06-01" // Base version
	latestVersion  = "2024-10-22" // Latest stable version
)

// ClientOption configures the client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithVersion sets the API version.
func WithVersion(version string) ClientOption {
	return func(c *Client) {
		c.version = version
	}
}

// Client is a custom HTTP client for the Anthropic API.
type Client struct {
	apiKey     string
	baseURL    string
	version    string
	httpClient *http.Client
}

// NewClient creates a new Anthropic API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		version:    defaultVersion,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RequestOptions contains per-request options.
type RequestOptions struct {
	// UserAgent is the User-Agent header to send with the request.
	// If set, it will be forwarded as-is to the upstream API.
	UserAgent string

	// BetaFeatures specifies which beta features to enable.
	// Example: "extended-thinking-2025-05-14,computer-use-2024-10-22"
	BetaFeatures string
}

// CreateMessage sends a messages request.
func (c *Client) CreateMessage(ctx context.Context, req *MessagesRequest, opts *RequestOptions) (*MessagesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq, opts)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if apiErr, err := ParseErrorResponse(respBody); err == nil && apiErr != nil {
			return nil, apiErr.ToCanonical()
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result MessagesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// StreamMessage sends a streaming messages request and returns a channel of events.
func (c *Client) StreamMessage(ctx context.Context, req *MessagesRequest, opts *RequestOptions) (<-chan StreamEventResult, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq, opts)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if apiErr, err := ParseErrorResponse(respBody); err == nil && apiErr != nil {
			return nil, apiErr.ToCanonical()
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	out := make(chan StreamEventResult)
	go c.streamReader(resp.Body, out)
	return out, nil
}

// StreamEventResult wraps a streaming event or error.
type StreamEventResult struct {
	EventType string
	Data      json.RawMessage
	Err       error
}

// ParseMessageStart parses a message_start event.
func (r *StreamEventResult) ParseMessageStart() (*MessageStartEvent, error) {
	var event MessageStartEvent
	if err := json.Unmarshal(r.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseContentBlockStart parses a content_block_start event.
func (r *StreamEventResult) ParseContentBlockStart() (*ContentBlockStartEvent, error) {
	var event ContentBlockStartEvent
	if err := json.Unmarshal(r.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseContentBlockDelta parses a content_block_delta event.
func (r *StreamEventResult) ParseContentBlockDelta() (*ContentBlockDeltaEvent, error) {
	var event ContentBlockDeltaEvent
	if err := json.Unmarshal(r.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseContentBlockStop parses a content_block_stop event.
func (r *StreamEventResult) ParseContentBlockStop() (*ContentBlockStopEvent, error) {
	var event ContentBlockStopEvent
	if err := json.Unmarshal(r.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseMessageDelta parses a message_delta event.
func (r *StreamEventResult) ParseMessageDelta() (*MessageDeltaEvent, error) {
	var event MessageDeltaEvent
	if err := json.Unmarshal(r.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseMessageStop parses a message_stop event.
func (r *StreamEventResult) ParseMessageStop() (*MessageStopEvent, error) {
	var event MessageStopEvent
	if err := json.Unmarshal(r.Data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (c *Client) streamReader(body io.ReadCloser, out chan<- StreamEventResult) {
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	// Increase buffer size for potentially large chunks
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse event type
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			out <- StreamEventResult{
				EventType: currentEvent,
				Data:      json.RawMessage(data),
			}

			// Stop on message_stop
			if currentEvent == "message_stop" {
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		out <- StreamEventResult{Err: fmt.Errorf("stream read error: %w", err)}
	}
}

// ListModels retrieves the list of available models.
func (c *Client) ListModels(ctx context.Context, opts *RequestOptions) (*ModelList, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq, opts)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if apiErr, err := ParseErrorResponse(respBody); err == nil && apiErr != nil {
			return nil, apiErr.ToCanonical()
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result ModelList
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// CountTokens counts tokens for a messages request.
func (c *Client) CountTokens(ctx context.Context, req *CountTokensRequest, opts *RequestOptions) (*CountTokensResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages/count_tokens", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq, opts)

	// Count tokens requires beta header
	httpReq.Header.Set("anthropic-beta", "token-counting-2024-11-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if apiErr, err := ParseErrorResponse(respBody); err == nil && apiErr != nil {
			return nil, apiErr.ToCanonical()
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CountTokensResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (c *Client) setHeaders(req *http.Request, opts *RequestOptions) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.version)

	// Set User-Agent - forward the incoming user agent if provided
	if opts != nil && opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	} else {
		req.Header.Set("User-Agent", "polyglot-llm-gateway/1.0")
	}

	// Set beta features header if specified
	if opts != nil && opts.BetaFeatures != "" {
		req.Header.Set("anthropic-beta", opts.BetaFeatures)
	}
}
