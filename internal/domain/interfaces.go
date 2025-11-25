package domain

import (
	"context"
)

// Provider defines the interface for LLM providers.
type Provider interface {
	Name() string

	// APIType returns the type of API this provider implements.
	// Used for pass-through optimization when frontdoor matches provider.
	APIType() APIType

	// Complete handles unary requests (non-streaming)
	Complete(ctx context.Context, req *CanonicalRequest) (*CanonicalResponse, error)

	// Stream returns a channel of events.
	// The channel MUST be closed by the provider when done.
	Stream(ctx context.Context, req *CanonicalRequest) (<-chan CanonicalEvent, error)

	// ListModels returns the models supported by the provider/frontdoor pairing.
	ListModels(ctx context.Context) (*ModelList, error)
}

// PassthroughProvider extends Provider with raw request/response handling.
// Providers implementing this interface can bypass canonical conversion
// when the source API type matches their type.
type PassthroughProvider interface {
	Provider

	// SupportsPassthrough returns true if the provider can handle raw requests
	// from the given API type.
	SupportsPassthrough(sourceType APIType) bool

	// CompleteRaw handles a raw request body and returns a raw response body.
	// This bypasses canonical conversion for matched API types.
	CompleteRaw(ctx context.Context, req *CanonicalRequest) ([]byte, error)

	// StreamRaw handles a raw streaming request and returns canonical events
	// with RawEvent populated for pass-through to the client.
	StreamRaw(ctx context.Context, req *CanonicalRequest) (<-chan CanonicalEvent, error)
}

// ResponsesProvider extends Provider with Responses API support.
// Providers implementing this interface natively support the OpenAI Responses API.
type ResponsesProvider interface {
	Provider

	// SupportsResponses returns true if the provider natively supports
	// the Responses API.
	SupportsResponses() bool

	// CreateResponse handles a Responses API request.
	CreateResponse(ctx context.Context, req *ResponsesAPIRequest) (*ResponsesAPIResponse, error)

	// StreamResponse handles a streaming Responses API request.
	StreamResponse(ctx context.Context, req *ResponsesAPIRequest) (<-chan CanonicalEvent, error)

	// GetResponse retrieves a stored response by ID.
	GetResponse(ctx context.Context, id string) (*ResponsesAPIResponse, error)

	// CancelResponse cancels an in-progress response.
	CancelResponse(ctx context.Context, id string) (*ResponsesAPIResponse, error)
}

// ProviderCapabilities describes what features a provider supports.
type ProviderCapabilities struct {
	// Streaming indicates if the provider supports SSE streaming.
	Streaming bool

	// Vision indicates if the provider supports image inputs.
	Vision bool

	// FunctionCalling indicates if the provider supports tool/function calls.
	FunctionCalling bool

	// ParallelToolCalls indicates if multiple tools can be called at once.
	ParallelToolCalls bool

	// JSONMode indicates if the provider supports JSON response format.
	JSONMode bool

	// JSONSchema indicates if the provider supports JSON schema validation.
	JSONSchema bool

	// ResponsesAPI indicates if the provider natively supports Responses API.
	ResponsesAPI bool

	// MaxContextLength is the maximum context window size.
	MaxContextLength int

	// MaxOutputTokens is the default maximum output tokens.
	MaxOutputTokens int
}

// CapableProvider extends Provider with capability discovery.
type CapableProvider interface {
	Provider

	// Capabilities returns the provider's capabilities.
	Capabilities() ProviderCapabilities
}
