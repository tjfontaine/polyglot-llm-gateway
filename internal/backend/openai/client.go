package openai

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
	defaultBaseURL = "https://api.openai.com/v1"
	defaultTimeout = 120 // seconds
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

// Client is a custom HTTP client for the OpenAI API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new OpenAI API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
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
}

// CreateChatCompletion sends a chat completion request.
func (c *Client) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest, opts *RequestOptions) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
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

	var result ChatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Store the raw response for debugging
	result.RawBody = respBody

	return &result, nil
}

// StreamChatCompletion sends a streaming chat completion request and returns a channel of chunks.
func (c *Client) StreamChatCompletion(ctx context.Context, req *ChatCompletionRequest, opts *RequestOptions) (<-chan StreamResult, error) {
	req.Stream = true
	if req.StreamOptions == nil {
		req.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
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

	out := make(chan StreamResult)
	go c.streamReader(resp.Body, out)
	return out, nil
}

// StreamResult wraps a chunk or error from streaming.
type StreamResult struct {
	Chunk *ChatCompletionChunk
	Err   error
}

func (c *Client) streamReader(body io.ReadCloser, out chan<- StreamResult) {
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	// Increase buffer size for potentially large chunks
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip non-data lines
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			return
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			out <- StreamResult{Err: fmt.Errorf("failed to unmarshal chunk: %w", err)}
			return
		}

		out <- StreamResult{Chunk: &chunk}
	}

	if err := scanner.Err(); err != nil {
		out <- StreamResult{Err: fmt.Errorf("stream read error: %w", err)}
	}
}

// ListModels retrieves the list of available models.
func (c *Client) ListModels(ctx context.Context, opts *RequestOptions) (*ModelList, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
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

func (c *Client) setHeaders(req *http.Request, opts *RequestOptions) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Set User-Agent - forward the incoming user agent if provided
	if opts != nil && opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	} else {
		req.Header.Set("User-Agent", "polyglot-llm-gateway/1.0")
	}
}

// ========== Responses API ==========

// CreateResponse sends a request to the Responses API.
func (c *Client) CreateResponse(ctx context.Context, req *ResponsesRequest, opts *RequestOptions) (*ResponsesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
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

	var result ResponsesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Store the raw response for debugging
	result.RawBody = respBody

	return &result, nil
}

// ResponsesStreamResult wraps a streaming event or error.
type ResponsesStreamResult struct {
	Event *ResponsesStreamEvent
	Err   error
}

// ResponsesStreamEvent represents a streaming event from the Responses API.
type ResponsesStreamEvent struct {
	Type string
	Data json.RawMessage
}

// StreamResponse sends a streaming request to the Responses API.
// Note: The Responses API uses SSE events and includes usage in "response.completed" events,
// so stream_options is not needed (and not supported by the API).
func (c *Client) StreamResponse(ctx context.Context, req *ResponsesRequest, opts *RequestOptions) (<-chan ResponsesStreamResult, error) {
	req.Stream = true
	// Don't set stream_options for Responses API - it's not supported.
	// Usage is automatically included in "response.completed" events.

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
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

	out := make(chan ResponsesStreamResult)
	go c.responsesStreamReader(resp.Body, out)
	return out, nil
}

func (c *Client) responsesStreamReader(body io.ReadCloser, out chan<- ResponsesStreamResult) {
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var currentEvent string
	var currentData strings.Builder

	// The following lines are from a different context and are not syntactically valid here.
	// fmt.Printf("[DEBUG] Codec: apiResp.RawBody length=%d\n", len(apiResp.RawBody))
	// canonResp := &domain.CanonicalResponse{
	// 	ID:                apiResp.ID,
	// 	Object:            apiResp.Object,
	// 	Created:           apiResp.Created,
	// 	Model:             apiResp.Model,
	// 	Choices:           choices,
	// 	SystemFingerprint: apiResp.SystemFingerprint,
	// 	Usage: domain.Usage{
	// 		PromptTokens:     apiResp.Usage.PromptTokens,
	// 		CompletionTokens: apiResp.Usage.CompletionTokens,
	// 		TotalTokens:      apiResp.Usage.TotalTokens,
	// 	},
	// 	SourceAPIType: domain.APITypeOpenAI,
	// 	RawResponse:   apiResp.RawBody,
	// }
	// fmt.Printf("[DEBUG] Codec: canonResp.RawResponse length=%d\n", len(canonResp.RawResponse))
	// return canonResp

	for scanner.Scan() {
		line := scanner.Text()

		// Empty line indicates end of event
		if line == "" {
			if currentEvent != "" && currentData.Len() > 0 {
				data := currentData.String()
				if data != "[DONE]" {
					out <- ResponsesStreamResult{
						Event: &ResponsesStreamEvent{
							Type: currentEvent,
							Data: json.RawMessage(data),
						},
					}
				}
			}
			currentEvent = ""
			currentData.Reset()
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
			currentData.WriteString(data)
		}
	}

	if err := scanner.Err(); err != nil {
		out <- ResponsesStreamResult{Err: fmt.Errorf("stream read error: %w", err)}
	}
}
