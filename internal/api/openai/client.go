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
