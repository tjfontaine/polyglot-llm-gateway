package codec

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageFetcher_FetchAndConvert_HTTPUrl(t *testing.T) {
	// Create a test server that serves an image
	imageData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}))
	defer ts.Close()

	fetcher := NewImageFetcher()
	source, err := fetcher.FetchAndConvert(context.Background(), ts.URL+"/test.png")
	if err != nil {
		t.Fatalf("FetchAndConvert returned error: %v", err)
	}

	if source.Type != "base64" {
		t.Errorf("Expected type 'base64', got %s", source.Type)
	}
	if source.MediaType != "image/png" {
		t.Errorf("Expected media type 'image/png', got %s", source.MediaType)
	}

	// Verify base64 decoding
	decoded, err := base64.StdEncoding.DecodeString(source.Data)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	if len(decoded) != len(imageData) {
		t.Errorf("Decoded data length mismatch: got %d, want %d", len(decoded), len(imageData))
	}
}

func TestImageFetcher_FetchAndConvert_DataURL(t *testing.T) {
	// Create a data URL with base64 encoded data
	originalData := []byte("test image data")
	encoded := base64.StdEncoding.EncodeToString(originalData)
	dataURL := "data:image/jpeg;base64," + encoded

	fetcher := NewImageFetcher()
	source, err := fetcher.FetchAndConvert(context.Background(), dataURL)
	if err != nil {
		t.Fatalf("FetchAndConvert returned error: %v", err)
	}

	if source.Type != "base64" {
		t.Errorf("Expected type 'base64', got %s", source.Type)
	}
	if source.MediaType != "image/jpeg" {
		t.Errorf("Expected media type 'image/jpeg', got %s", source.MediaType)
	}
	if source.Data != encoded {
		t.Errorf("Data mismatch")
	}
}

func TestImageFetcher_FetchAndConvert_DataURL_NormalizeJPG(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("test"))
	dataURL := "data:image/jpg;base64," + encoded

	fetcher := NewImageFetcher()
	source, err := fetcher.FetchAndConvert(context.Background(), dataURL)
	if err != nil {
		t.Fatalf("FetchAndConvert returned error: %v", err)
	}

	// image/jpg should be normalized to image/jpeg
	if source.MediaType != "image/jpeg" {
		t.Errorf("Expected media type 'image/jpeg', got %s", source.MediaType)
	}
}

func TestImageFetcher_FetchAndConvert_InvalidScheme(t *testing.T) {
	fetcher := NewImageFetcher()
	_, err := fetcher.FetchAndConvert(context.Background(), "ftp://example.com/image.png")
	if err == nil {
		t.Error("Expected error for invalid URL scheme")
	}
	if !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Errorf("Expected 'unsupported URL scheme' error, got: %v", err)
	}
}

func TestImageFetcher_FetchAndConvert_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	fetcher := NewImageFetcher()
	_, err := fetcher.FetchAndConvert(context.Background(), ts.URL+"/notfound.png")
	if err == nil {
		t.Error("Expected error for HTTP error response")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("Expected 'status 404' error, got: %v", err)
	}
}

func TestImageFetcher_FetchAndConvert_TooLarge(t *testing.T) {
	// Create a server that returns a large response
	largeData := make([]byte, 1024*1024) // 1MB
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(largeData)
	}))
	defer ts.Close()

	// Set max size to 1KB
	fetcher := NewImageFetcher(WithMaxSize(1024))
	_, err := fetcher.FetchAndConvert(context.Background(), ts.URL+"/large.png")
	if err == nil {
		t.Error("Expected error for too large image")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("Expected 'too large' error, got: %v", err)
	}
}

func TestImageFetcher_FetchAndConvert_UnsupportedMediaType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("not an image"))
	}))
	defer ts.Close()

	fetcher := NewImageFetcher()
	_, err := fetcher.FetchAndConvert(context.Background(), ts.URL+"/file.pdf")
	if err == nil {
		t.Error("Expected error for unsupported media type")
	}
	if !strings.Contains(err.Error(), "unsupported media type") {
		t.Errorf("Expected 'unsupported media type' error, got: %v", err)
	}
}

func TestImageFetcher_FetchAndConvert_WithContentTypeHeader(t *testing.T) {
	// Server that sets Content-Type correctly
	imageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse media type from query parameter
		mediaType := r.URL.Query().Get("type")
		if mediaType != "" {
			w.Header().Set("Content-Type", mediaType)
		}
		w.Write(imageData)
	}))
	defer ts.Close()

	tests := []struct {
		name      string
		queryType string
		mediaType string
	}{
		{"explicit jpeg", "image/jpeg", "image/jpeg"},
		{"explicit png", "image/png", "image/png"},
		{"explicit gif", "image/gif", "image/gif"},
		{"explicit webp", "image/webp", "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewImageFetcher()
			source, err := fetcher.FetchAndConvert(context.Background(), ts.URL+"?type="+tt.queryType)
			if err != nil {
				t.Fatalf("FetchAndConvert returned error: %v", err)
			}
			if source.MediaType != tt.mediaType {
				t.Errorf("Expected media type %s, got %s", tt.mediaType, source.MediaType)
			}
		})
	}
}

func TestInferMediaType(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://example.com/image.jpg", "image/jpeg"},
		{"http://example.com/image.JPEG", "image/jpeg"},
		{"http://example.com/image.png", "image/png"},
		{"http://example.com/image.PNG", "image/png"},
		{"http://example.com/image.gif", "image/gif"},
		{"http://example.com/image.webp", "image/webp"},
		{"http://example.com/image.unknown", "image/jpeg"}, // default
		{"http://example.com/image", "image/jpeg"},         // no extension
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := inferMediaType(tt.url)
			if result != tt.expected {
				t.Errorf("inferMediaType(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsSupportedMediaType(t *testing.T) {
	tests := []struct {
		mediaType string
		supported bool
	}{
		{"image/jpeg", true},
		{"image/jpg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"IMAGE/JPEG", true},
		{"image/jpeg; charset=utf-8", true},
		{"application/pdf", false},
		{"text/html", false},
		{"image/svg+xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			result := isSupportedMediaType(tt.mediaType)
			if result != tt.supported {
				t.Errorf("isSupportedMediaType(%q) = %v, want %v", tt.mediaType, result, tt.supported)
			}
		})
	}
}

func TestNormalizeMediaType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"image/jpeg", "image/jpeg"},
		{"image/jpg", "image/jpeg"},
		{"IMAGE/JPG", "image/jpeg"},
		{"image/png", "image/png"},
		{"image/jpeg; charset=utf-8", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeMediaType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeMediaType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDataURL(t *testing.T) {
	fetcher := NewImageFetcher()

	tests := []struct {
		name      string
		url       string
		wantType  string
		wantMedia string
		wantErr   bool
	}{
		{
			name:      "valid jpeg",
			url:       "data:image/jpeg;base64,/9j/4AAQSkZ",
			wantType:  "base64",
			wantMedia: "image/jpeg",
		},
		{
			name:      "valid png",
			url:       "data:image/png;base64,iVBORw0KGgo",
			wantType:  "base64",
			wantMedia: "image/png",
		},
		{
			name:    "missing base64 marker",
			url:     "data:image/jpeg,/9j/4AAQSkZ",
			wantErr: true,
		},
		{
			name:    "not a data URL",
			url:     "http://example.com/image.png",
			wantErr: true,
		},
		{
			name:    "missing comma",
			url:     "data:image/jpeg;base64",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := fetcher.parseDataURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if source.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", source.Type, tt.wantType)
			}
			if source.MediaType != tt.wantMedia {
				t.Errorf("MediaType = %q, want %q", source.MediaType, tt.wantMedia)
			}
		})
	}
}
