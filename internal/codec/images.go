// Package codec provides utilities for converting between API formats.
package codec

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ImageFetcher handles fetching remote images and converting them to base64.
type ImageFetcher struct {
	client  *http.Client
	maxSize int64 // Maximum allowed image size in bytes
}

// ImageFetcherOption configures the image fetcher.
type ImageFetcherOption func(*ImageFetcher)

// WithHTTPClient sets a custom HTTP client for the fetcher.
func WithImageHTTPClient(client *http.Client) ImageFetcherOption {
	return func(f *ImageFetcher) {
		f.client = client
	}
}

// WithMaxSize sets the maximum allowed image size.
func WithMaxSize(maxSize int64) ImageFetcherOption {
	return func(f *ImageFetcher) {
		f.maxSize = maxSize
	}
}

// NewImageFetcher creates a new image fetcher.
func NewImageFetcher(opts ...ImageFetcherOption) *ImageFetcher {
	f := &ImageFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxSize: 20 * 1024 * 1024, // 20MB default max
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// FetchAndConvert fetches an image from a URL and converts it to base64 format.
// Returns an ImageSource suitable for Anthropic's API.
func (f *ImageFetcher) FetchAndConvert(ctx context.Context, url string) (*domain.ImageSource, error) {
	// Check if it's already a data URL (base64 encoded)
	if strings.HasPrefix(url, "data:") {
		return f.parseDataURL(url)
	}

	// Validate URL scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("unsupported URL scheme: must be http:// or https://")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	// Check content length if available
	if resp.ContentLength > f.maxSize {
		return nil, fmt.Errorf("image too large: %d bytes (max %d)", resp.ContentLength, f.maxSize)
	}

	// Determine media type from Content-Type header
	mediaType := resp.Header.Get("Content-Type")
	if mediaType == "" {
		// Try to infer from URL
		mediaType = inferMediaType(url)
	}

	// Validate media type
	if !isSupportedMediaType(mediaType) {
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	// Read the image data with size limit
	limitReader := io.LimitReader(resp.Body, f.maxSize+1)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	if int64(len(data)) > f.maxSize {
		return nil, fmt.Errorf("image too large: exceeds %d bytes", f.maxSize)
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	return &domain.ImageSource{
		Type:      "base64",
		MediaType: normalizeMediaType(mediaType),
		Data:      encoded,
	}, nil
}

// parseDataURL parses a data URL and extracts the base64 content.
func (f *ImageFetcher) parseDataURL(url string) (*domain.ImageSource, error) {
	// Format: data:image/jpeg;base64,/9j/4AAQSkZ...
	if !strings.HasPrefix(url, "data:") {
		return nil, fmt.Errorf("not a data URL")
	}

	// Remove "data:" prefix
	content := url[5:]

	// Find the comma separator
	commaIdx := strings.Index(content, ",")
	if commaIdx == -1 {
		return nil, fmt.Errorf("invalid data URL: missing comma separator")
	}

	metadata := content[:commaIdx]
	data := content[commaIdx+1:]

	// Parse metadata
	parts := strings.Split(metadata, ";")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid data URL: missing media type")
	}

	mediaType := parts[0]
	if !isSupportedMediaType(mediaType) {
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	// Check for base64 encoding
	isBase64 := false
	for _, part := range parts[1:] {
		if part == "base64" {
			isBase64 = true
			break
		}
	}

	if !isBase64 {
		return nil, fmt.Errorf("data URL must be base64 encoded")
	}

	return &domain.ImageSource{
		Type:      "base64",
		MediaType: normalizeMediaType(mediaType),
		Data:      data,
	}, nil
}

// inferMediaType attempts to infer the media type from a URL.
func inferMediaType(url string) string {
	urlLower := strings.ToLower(url)

	switch {
	case strings.HasSuffix(urlLower, ".jpg") || strings.HasSuffix(urlLower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(urlLower, ".png"):
		return "image/png"
	case strings.HasSuffix(urlLower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(urlLower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg" // Default assumption
	}
}

// isSupportedMediaType checks if the media type is supported by Anthropic.
func isSupportedMediaType(mediaType string) bool {
	// Normalize by taking only the main type (ignore parameters like charset)
	mainType := strings.Split(mediaType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))

	switch mainType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

// normalizeMediaType normalizes the media type to a standard format.
func normalizeMediaType(mediaType string) string {
	mainType := strings.Split(mediaType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))

	// Normalize image/jpg to image/jpeg
	if mainType == "image/jpg" {
		return "image/jpeg"
	}
	return mainType
}

// ConvertContentPart converts an image_url content part to base64 image for Anthropic.
// If the content part is not an image_url or conversion fails, returns the original.
func (f *ImageFetcher) ConvertContentPart(ctx context.Context, part *domain.ContentPart) (*domain.ContentPart, error) {
	if part.Type != domain.ContentTypeImageURL || part.ImageURL == nil {
		return part, nil
	}

	source, err := f.FetchAndConvert(ctx, part.ImageURL.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to convert image URL: %w", err)
	}

	return &domain.ContentPart{
		Type:   domain.ContentTypeImage,
		Source: source,
	}, nil
}
