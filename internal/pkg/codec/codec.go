// Package codec provides interfaces and implementations for converting between
// API-specific formats (OpenAI, Anthropic) and the internal canonical (IR) format.
//
// The codec pattern allows frontdoors and providers to share conversion logic:
//   - Frontdoor receives request → Codec.DecodeRequest() → CanonicalRequest
//   - CanonicalRequest → Codec.EncodeRequest() → Provider API call
//   - Provider response → Codec.DecodeResponse() → CanonicalResponse
//   - CanonicalResponse → Codec.EncodeResponse() → Frontdoor response
package codec

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

// Codec handles bidirectional conversion between API-specific types and canonical types.
// Each API format (OpenAI, Anthropic) has its own codec implementation.
type Codec interface {
	// Name returns the codec name (e.g., "openai", "anthropic")
	Name() string

	// Request conversion
	DecodeRequest(data []byte) (*domain.CanonicalRequest, error)
	EncodeRequest(req *domain.CanonicalRequest) ([]byte, error)

	// Response conversion
	DecodeResponse(data []byte) (*domain.CanonicalResponse, error)
	EncodeResponse(resp *domain.CanonicalResponse) ([]byte, error)

	// Streaming - decode provider chunks to canonical events
	DecodeStreamChunk(data []byte) (*domain.CanonicalEvent, error)
	// Streaming - encode canonical events to frontdoor format
	EncodeStreamChunk(event *domain.CanonicalEvent, metadata *StreamMetadata) ([]byte, error)
}

// StreamMetadata contains metadata needed for encoding stream chunks
type StreamMetadata struct {
	ID      string
	Model   string
	Created int64
}

// RequestCodec provides only request encoding/decoding.
// Useful when you only need request conversion.
type RequestCodec interface {
	DecodeRequest(data []byte) (*domain.CanonicalRequest, error)
	EncodeRequest(req *domain.CanonicalRequest) ([]byte, error)
}

// ResponseCodec provides only response encoding/decoding.
// Useful when you only need response conversion.
type ResponseCodec interface {
	DecodeResponse(data []byte) (*domain.CanonicalResponse, error)
	EncodeResponse(resp *domain.CanonicalResponse) ([]byte, error)
}

// StreamCodec provides streaming chunk encoding/decoding.
type StreamCodec interface {
	DecodeStreamChunk(data []byte) (*domain.CanonicalEvent, error)
	EncodeStreamChunk(event *domain.CanonicalEvent, metadata *StreamMetadata) ([]byte, error)
}
