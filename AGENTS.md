# Polyglot LLM Gateway — Agent Guidelines

## Scope
This file applies to the entire repository unless a more specific `AGENTS.md` is introduced deeper in the tree.

## Project Overview
- **Purpose:** A Go-based gateway that normalizes LLM requests into a canonical shape, routes them to configured providers, and optionally serves a React/Vite control-plane UI.
- **Key server layers:**
  - `internal/frontdoor/*` map external protocols (OpenAI, Anthropic, Responses API) into canonical requests.
  - `internal/router` selects providers based on routing rules and model rewrites.
  - `internal/provider/*` translate canonical requests into provider-specific API calls; streaming is handled through `CanonicalEvent` channels.
  - `internal/storage/*` persist conversation data when Responses APIs are enabled (SQLite or in-memory).
  - `internal/pkg/auth` and `internal/pkg/tenant` enable multi-tenant mode with bearer key hashing.
  - `internal/api/server` wires chi middleware, auth, and OpenTelemetry instrumentation.
  - `internal/api/controlplane` serves the admin API and React UI for observing gateway state.
- **Control plane:** `web/control-plane` is a React 19 + Vite app whose built assets are copied into `internal/api/controlplane/dist` by the `Makefile`.

## Control Plane Architecture

### Backend API (`internal/api/controlplane`)
The control plane server exposes read-only admin APIs under `/admin/api/`:
- `GET /api/stats` — Runtime statistics (uptime, goroutines, memory)
- `GET /api/overview` — Gateway configuration summary (apps, providers, routing, tenants)
- `GET /api/interactions` — Unified list of all stored data (conversations + responses)
- `GET /api/interactions/{id}` — Detail view for any interaction
- `GET /api/threads` — Legacy: list conversations only
- `GET /api/responses` — Legacy: list responses only

### Unified Interactions Model
Conversations (from chat APIs) and Responses (from the Responses API) are unified into a single "Interaction" concept:
- Both share common fields: `id`, `type`, `model`, `metadata`, `created_at`, `updated_at`
- Conversations have `messages[]` and `message_count`
- Responses have `request`, `response`, `status`, and `previous_response_id`
- The `/api/interactions` endpoint merges both, sorted by `updated_at` descending
- Filter by type using `?type=conversation` or `?type=response`

### Frontend Structure (`web/control-plane/src`)
```
src/
├── components/
│   ├── Layout.tsx          # Main layout with header, nav, stats bar
│   ├── Layout.test.tsx     # Layout component tests
│   ├── ui/
│   │   ├── index.tsx       # Reusable UI components (Pill, InfoCard, etc.)
│   │   └── ui.test.tsx     # UI component tests
│   └── index.ts
├── hooks/
│   ├── useApi.tsx          # React context for API data fetching
│   └── useApi.test.tsx     # API hook tests
├── pages/
│   ├── Dashboard.tsx       # Landing page with overview cards
│   ├── Dashboard.test.tsx  # Dashboard tests
│   ├── Topology.tsx        # Apps & providers configuration
│   ├── Topology.test.tsx   # Topology tests
│   ├── Routing.tsx         # Routing rules & tenants
│   ├── Routing.test.tsx    # Routing tests
│   ├── Data.tsx            # Unified data explorer (interactions)
│   └── index.ts
├── test/
│   ├── setup.ts            # Vitest setup (jest-dom, mocks)
│   ├── mocks.ts            # Mock data for tests
│   └── test-utils.tsx      # Custom render with providers
├── types/
│   └── index.ts            # TypeScript interfaces
├── App.tsx                 # Router setup
└── main.tsx                # Entry point
```

### Page Responsibilities
- **Dashboard**: Quick overview with stats and cards linking to detailed pages
- **Topology**: Detailed view of configured apps and providers
- **Routing**: Model routing rules and tenant configuration
- **Data**: Unified explorer for all recorded interactions (conversations + responses)

### Design Principles
- **Unified experience**: Don't separate conversations and responses into different pages; they represent the same concept (LLM interactions) viewed through different API lenses
- **Read-only**: The control plane never modifies gateway state; it's purely observational
- **Filter, don't fragment**: Use filters within a single view rather than creating multiple similar pages

### Backend Registration & Routing
- Providers/frontdoors register via explicit functions (no init side effects). `cmd/gateway` calls `registration.RegisterBuiltins()` to wire OpenAI/Anthropic. New providers/frontdoors should expose `RegisterProviderFactory()` / `RegisterFrontdoor()` and be invoked from that helper or directly in `cmd/gateway`.
- Routing and model mapping live in `internal/router` (legacy `internal/policy` removed). Use `router.NewProviderRouter` for routing and `router.NewMappingProvider` when model rewrites/prefix mapping are needed.

### Frontend Null Safety
The Go backend may return `null` for empty slices (e.g., `routing.rules`, `tenants`, `providers`). Always use optional chaining when accessing nested properties from API responses:

```typescript
// ✅ Correct - handles null arrays
overview?.routing?.rules?.length ?? 0
(overview?.providers ?? []).map(...)

// ❌ Incorrect - will throw if routing or rules is null
overview?.routing.rules.length
overview.providers.map(...)
```

Tests in `src/test/mocks.ts` include `mockNullArraysOverview` to verify null-safety. When adding new components that consume API data, include a test case with null arrays.

## Domain Model Architecture

The gateway's core design principle is **canonicalization through the domain model**. All data flows through canonical types defined in `internal/core/domain`, enabling seamless translation between different API formats.

### Core Architecture Pattern

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              GATEWAY                                             │
│                                                                                  │
│  ┌─────────────┐     ┌──────────────────────────────────────┐     ┌───────────┐ │
│  │  FRONTDOOR  │     │           DOMAIN MODEL               │     │ PROVIDER  │ │
│  │             │     │                                      │     │           │ │
│  │  Anthropic ─┼────▶│  CanonicalRequest                   │────▶│  OpenAI   │ │
│  │  OpenAI     │     │  CanonicalResponse                  │     │  Anthropic│ │
│  │  Responses  │◀────┼  CanonicalEvent                     │◀────│           │ │
│  │             │     │  APIError                           │     │           │ │
│  └─────────────┘     └──────────────────────────────────────┘     └───────────┘ │
│        ▲                           ▲                                    ▲        │
│        │                           │                                    │        │
│     Codec                       Domain                               Codec       │
│   (decode/encode)               Types                            (decode/encode) │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### The Pattern: Decode → Canonical → Encode

Every data type follows the same pattern:

| Data Type | Frontdoor Decodes | Canonical Type | Provider Encodes |
|-----------|-------------------|----------------|------------------|
| Requests | API-specific JSON → | `CanonicalRequest` | → Provider API format |
| Responses | ← API-specific JSON | `CanonicalResponse` | ← Provider API format |
| Streaming | ← API-specific SSE | `CanonicalEvent` | ← Provider SSE |
| Errors | ← API-specific error | `APIError` | ← Provider error |
| Token Counts | API-specific format → | `TokenCountRequest` | → Estimation/API |

### Canonical Types (`internal/core/domain`)

**`CanonicalRequest`** - Normalized request format:
```go
type CanonicalRequest struct {
    Model         string
    Messages      []Message
    MaxTokens     int
    Temperature   float32
    Stream        bool
    Tools         []ToolDefinition
    SourceAPIType APIType      // Which frontdoor received this
    RawRequest    []byte       // Original bytes for pass-through
}
```

**`CanonicalResponse`** - Normalized response format:
```go
type CanonicalResponse struct {
    ID            string
    Model         string
    Choices       []Choice
    Usage         Usage
    SourceAPIType APIType      // Which provider returned this
    RawResponse   []byte       // Original bytes for pass-through
}
```

**`CanonicalEvent`** - Streaming event:
```go
type CanonicalEvent struct {
    Role         string
    ContentDelta string
    ToolCall     *ToolCallChunk
    Usage        *Usage
    Error        error
}
```

**`APIError`** - Canonical error:
```go
type APIError struct {
    Type       ErrorType    // invalid_request, rate_limit, etc.
    Code       ErrorCode    // context_length_exceeded, etc.
    Message    string
    StatusCode int
    SourceAPI  APIType
}
```

### Codec Layer (`internal/codec`)

Codecs handle bidirectional translation between API-specific formats and canonical types:

```go
type Codec interface {
    DecodeRequest(data []byte) (*domain.CanonicalRequest, error)
    EncodeRequest(req *domain.CanonicalRequest) ([]byte, error)
    DecodeResponse(data []byte) (*domain.CanonicalResponse, error)
    EncodeResponse(resp *domain.CanonicalResponse) ([]byte, error)
    DecodeStreamChunk(data []byte) (*domain.CanonicalEvent, error)
    EncodeStreamChunk(event *domain.CanonicalEvent, meta *StreamMetadata) ([]byte, error)
}
```

Each API type has its consolidated package containing types, client, codec, provider, and frontdoor:
- `internal/backend/anthropic/` - Anthropic Messages API (types, client, codec, provider, factory, frontdoor)
- `internal/backend/openai/` - OpenAI Chat Completions API (types, client, codec, provider, factory, frontdoor)

### Request Flow Example

**Anthropic client → OpenAI provider:**

```go
// 1. Frontdoor receives Anthropic request
body, _ := io.ReadAll(r.Body)

// 2. Decode to canonical using Anthropic codec
canonReq, _ := h.codec.DecodeRequest(body)  // anthropic.Codec

// 3. Provider translates canonical to OpenAI format
apiReq := openaicodec.CanonicalToAPIRequest(canonReq)

// 4. Send to OpenAI, get response
apiResp, _ := client.CreateChatCompletion(ctx, apiReq)

// 5. Decode OpenAI response to canonical
canonResp := openaicodec.APIResponseToCanonical(apiResp)

// 6. Encode canonical to Anthropic format for client
respBody, _ := h.codec.EncodeResponse(canonResp)  // anthropic.Codec
w.Write(respBody)
```

### Error Flow

Errors follow the same canonical pattern:

```go
// 1. Provider API returns error
if apiErr, _ := openai.ParseErrorResponse(respBody); apiErr != nil {
    return nil, apiErr.ToCanonical()  // Convert to domain.APIError
}

// 2. Frontdoor formats for its API type
codec.WriteError(w, err, domain.APITypeAnthropic)

// 3. Client receives Anthropic-formatted error
// {"type": "error", "error": {"type": "rate_limit_error", "message": "..."}}
```

### Error Type Mapping

| Domain Error | Anthropic | OpenAI | HTTP Status |
|--------------|-----------|--------|-------------|
| `invalid_request` | `invalid_request_error` | `invalid_request_error` | 400 |
| `authentication` | `authentication_error` | `authentication_error` | 401 |
| `permission` | `permission_error` | `permission_denied` | 403 |
| `not_found` | `not_found_error` | `not_found` | 404 |
| `rate_limit` | `rate_limit_error` | `rate_limit_error` | 429 |
| `context_length` | `invalid_request_error` | `invalid_request_error` | 400 |
| `max_tokens` | `invalid_request_error` | `invalid_request_error` | 400 |
| `overloaded` | `overloaded_error` | `service_unavailable` | 503 |
| `server` | `api_error` | `server_error` | 500 |

### Pass-Through Optimization

When frontdoor and provider use the same API type, the gateway can skip canonical conversion:

```go
// Check if we can pass through directly
if req.SourceAPIType == provider.APIType() {
    // Use RawRequest/RawResponse directly
    resp.RawResponse = upstreamBody
}

// Frontdoor checks for pass-through
if len(resp.RawResponse) > 0 && resp.SourceAPIType == domain.APITypeAnthropic {
    w.Write(resp.RawResponse)  // Direct pass-through
} else {
    respBody, _ := h.codec.EncodeResponse(resp)  // Canonical conversion
    w.Write(respBody)
}
```

### Adding New API Support

To add a new API type (e.g., Google Gemini), create a consolidated package containing all related code:

#### 1. Create Consolidated Package
Create `internal/gemini/` with the following files:

**`types.go`** - API request/response structures:
```go
package gemini

// Request and response types for the Gemini API
type GenerateContentRequest struct { /* ... */ }
type GenerateContentResponse struct { /* ... */ }
// ... etc
```

**`client.go`** - HTTP client for API calls:
```go
package gemini

type Client struct { /* ... */ }
func NewClient(apiKey string, opts ...ClientOption) *Client { /* ... */ }
func (c *Client) GenerateContent(ctx context.Context, req *GenerateContentRequest) (*GenerateContentResponse, error) { /* ... */ }
```

**`codec.go`** - Codec implementing bidirectional translation:
```go
package gemini

type Codec struct{}
func NewCodec() *Codec { return &Codec{} }
// Implement codec.Codec interface
func (c *Codec) DecodeRequest(data []byte) (*domain.CanonicalRequest, error) { /* ... */ }
func (c *Codec) EncodeResponse(resp *domain.CanonicalResponse) ([]byte, error) { /* ... */ }
// ... etc
```

**`provider.go`** - Provider implementation:
```go
package gemini

type Provider struct { /* ... */ }

func NewProvider(apiKey string, opts ...ProviderOption) *Provider { /* ... */ }
func (p *Provider) Name() string { return "gemini" }
func (p *Provider) APIType() domain.APIType { return domain.APITypeGemini }
func (p *Provider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) { /* ... */ }
func (p *Provider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) { /* ... */ }
func (p *Provider) ListModels(ctx context.Context) (*domain.ModelList, error) { /* ... */ }
```

**`factory.go`** - Self-registering factory:
```go
package gemini

import (
    "github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
    "github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
    providerregistry "github.com/tjfontaine/polyglot-llm-gateway/internal/provider/registry"
)

const ProviderType = "gemini"

// RegisterProviderFactory registers the Gemini provider explicitly.
func RegisterProviderFactory() {
    if providerregistry.IsRegistered(ProviderType) {
        return
    }
    providerregistry.RegisterFactory(providerregistry.ProviderFactory{
        Type:           ProviderType,
        APIType:        domain.APITypeGemini,
        Description:    "Google Gemini API provider",
        Create:         CreateProviderFromConfig,
        ValidateConfig: ValidateProviderConfig,
    })
}

func CreateProviderFromConfig(cfg config.ProviderConfig) (domain.Provider, error) {
    var opts []ProviderOption
    if cfg.BaseURL != "" {
        opts = append(opts, WithProviderBaseURL(cfg.BaseURL))
    }
    return NewProvider(cfg.APIKey, opts...), nil
}

func ValidateProviderConfig(cfg config.ProviderConfig) error {
    if cfg.APIKey == "" {
        return fmt.Errorf("api_key is required")
    }
    return nil
}
```

#### 2. Register Package Import
Add a blank import in `internal/provider/registry.go` to trigger init():
```go
import (
    // ... existing imports ...
    _ "github.com/tjfontaine/polyglot-llm-gateway/internal/gemini"
)
```

#### 3. Add Frontdoor Handler (If exposing a new API format)
Frontdoor handlers can now be included in the same consolidated package. Add `frontdoor.go` to `internal/gemini/`:

**`frontdoor.go`** - HTTP handlers for the API format:
```go
package gemini

import (
    "net/http"
    "github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
    "github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor/registry"
    // ... other imports
)

// FrontdoorType is the frontdoor type identifier used in configuration.
const FrontdoorType = "gemini"

// FrontdoorAPIType returns the canonical API type for this frontdoor.
func FrontdoorAPIType() domain.APIType {
    return domain.APITypeGemini
}

func RegisterFrontdoor() {
    if registry.IsRegistered(FrontdoorType) {
        return
    }
    registry.RegisterFactory(registry.FrontdoorFactory{
        Type:           FrontdoorType,
        APIType:        FrontdoorAPIType(),
        Description:    "Google Gemini API format",
        CreateHandlers: createFrontdoorHandlers,
    })
}

// FrontdoorHandler handles Gemini API requests.
type FrontdoorHandler struct {
    provider domain.Provider
    codec    codec.Codec  // Uses NewCodec() from same package
    // ...
}

func NewFrontdoorHandler(provider domain.Provider, store storage.ConversationStore, appName string, models []config.ModelListItem) *FrontdoorHandler {
    return &FrontdoorHandler{
        provider: provider,
        codec:    NewCodec(),  // Direct call - no package alias needed
        // ...
    }
}

func createFrontdoorHandlers(cfg registry.HandlerConfig) []registry.HandlerRegistration {
    handler := NewFrontdoorHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models)
    return []registry.HandlerRegistration{
        {Path: cfg.BasePath + "/v1:generateContent", Method: http.MethodPost, Handler: handler.HandleGenerateContent},
    }
}

func (h *FrontdoorHandler) HandleGenerateContent(w http.ResponseWriter, r *http.Request) { /* ... */ }
```

**Note on import cycles:** The consolidated package approach works because:
1. Provider registration happens in `factory.go` which only imports `config`, `domain`, and `provider/registry`
2. Frontdoor registration in `frontdoor.go` imports additional packages like `conversation`, `server`, etc.
3. The `provider/registry.go` no longer imports the consolidated packages directly - instead, `cmd/gateway/main.go` imports them to trigger registration
4. Tests that need factory registration use external test packages (e.g., `package gemini_test`) to avoid cycles

#### 4. Wire Registration
Call the explicit registration functions (e.g., `gemini.RegisterProviderFactory()` and `gemini.RegisterFrontdoor()`) from `cmd/gateway` or the shared helper `internal/registration.RegisterBuiltins()` used by main/tests. Avoid init-based side effects and blank-import wiring.

#### 5. Add Error Mapping
Add `ToCanonical()` method for provider errors and update `codec/errors.go` formatter.

### Verification

After adding a new provider/frontdoor:
- Run `go build ./...` to verify compilation
- Run `go test ./internal/provider/... ./internal/frontdoor/...` to verify tests pass
- Check that `provider.ListProviderTypes()` includes your new type
- Check that `frontdoor.ListFrontdoorTypes()` includes your new type (if applicable)

The factory pattern ensures compile-time validation and makes the required components explicit.

## Server Layer & Middleware (`internal/api/server`)

The server package provides the HTTP infrastructure for the gateway, including middleware components that form the request processing pipeline.

### Package Structure

```
internal/api/server/
├── server.go         # Server setup and middleware chain configuration
├── doc.go            # Package documentation
├── requestid.go      # Request ID generation middleware
├── logging.go        # Structured logging middleware
├── authmiddleware.go # API key authentication middleware
├── timeout.go        # Request timeout middleware
├── ratelimit.go      # Rate limit header normalization
└── middleware_test.go # Comprehensive middleware tests
```

### Middleware Components

| Middleware | File | Purpose |
|------------|------|---------|
| `RequestIDMiddleware` | `requestid.go` | Generates UUID for each request, adds to context and X-Request-ID header |
| `LoggingMiddleware` | `logging.go` | Logs request start/completion with structured fields |
| `AuthMiddleware` | `authmiddleware.go` | Validates Bearer tokens, injects tenant into context |
| `TimeoutMiddleware` | `timeout.go` | Enforces request deadline via context cancellation |
| `RateLimitNormalizingMiddleware` | `ratelimit.go` | Writes standardized x-ratelimit-* headers |

### Middleware Chain Order

The server applies middleware in this order:
1. **RequestIDMiddleware** - First, to enable request tracing
2. **LoggingMiddleware** - Captures all requests for observability
3. **AuthMiddleware** - Validates API keys (if multi-tenant mode enabled)
4. **TimeoutMiddleware** - Enforces request timeouts
5. **Recoverer** - Catches panics (from chi/middleware)
6. **otelhttp** - OpenTelemetry instrumentation

### Context Utilities

The package provides utilities for request-scoped data:

```go
// In request handlers:
requestID := server.GetRequestID(r.Context())
server.AddLogField(r.Context(), "model", modelName)
server.AddError(r.Context(), err)

// For rate limits:
ctx := server.SetRateLimits(r.Context(), &server.RateLimitInfo{...})
```

### Adding New Middleware

When adding new middleware:
1. Create a new file (e.g., `newmiddleware.go`) following the existing pattern
2. Use the standard `func(http.Handler) http.Handler` signature
3. Add tests in `middleware_test.go`
4. Update `server.go` to include in the chain if it should apply to all routes
5. Update this documentation

## Coding Conventions
- **Go style:** Keep code `gofmt`-clean and idiomatic Go. Prefer small, focused functions and return descriptive errors. Avoid introducing panics in request paths.
- **Configuration-driven behavior:** Respect the config structs in `internal/pkg/config` and the registry helpers when adding new frontdoors or providers; plug into `frontdoor.Registry`/`provider.Registry` using the factory pattern instead of hardcoding wiring.
- **Factory pattern:** New providers and frontdoors must register themselves via `RegisterFactory()` in their respective package init() functions.
- **Canonical types first:** Work with `internal/core/domain` types at API boundaries. Frontdoors should fully populate `CanonicalRequest`/`CanonicalResponse` and let providers handle translation details.
- **Error handling:** API clients should convert provider-specific errors to `domain.APIError` using `ToCanonical()`. Frontdoors should use `codec.WriteError()` to format errors for their API type.
- **Streaming:** When adding streaming support, emit `CanonicalEvent` values and propagate provider errors via the channel before closing it.
- **Tests & fixtures:** Integration-style provider tests rely on go-vcr cassettes under `testdata/fixtures`. Use `VCR_MODE=record` with real API keys to refresh cassettes; default `go test` replays without network access.
- **Logging/telemetry:** Use the structured `slog` logger already configured in `cmd/gateway/main.go` and preserve OpenTelemetry middleware hooks in `internal/api/server` when adding new routes.
- **Frontend:** Keep React components type-safe (TypeScript) and run linting before committing UI changes.

## Testing Expectations
- **Go:** Run `go test ./...` from the repo root. Provider tests will skip recording if the relevant API key is missing and `VCR_MODE=record` is set; in replay mode they work offline.
- **Frontend:** From `web/control-plane`:
  - `npm install` — Install dependencies (once)
  - `npm run lint` — Run ESLint
  - `npm run test` — Run Vitest tests
  - `npm run test:watch` — Run tests in watch mode
  - `npm run test:coverage` — Run tests with coverage report
  - `npm run build` — Production build
- **Frontend testing stack:** Vitest + React Testing Library + jsdom. Tests are co-located with components (e.g., `Component.test.tsx`).
- The `Makefile` target `build` will bundle both frontend and backend if needed.

## Dependency & Build Notes
- **Go version:** Module targets Go 1.25.3; ensure toolchain compatibility.
- **Binaries:** Backend entrypoints live in `cmd/gateway` (server) and `cmd/keygen` (API key hashing helper).
- **Artifacts:** The frontend build output in `web/control-plane/dist` is copied into `internal/api/controlplane/dist`; do not commit `node_modules`.

## Review Checklist for Changes
- Code is formatted (`gofmt`, ESLint for frontend) and lints cleanly.
- New routes or providers are registered via the appropriate registry helpers using the factory pattern instead of manual router wiring.
- Tests are added or updated, especially when touching routing, provider translations, or storage persistence.
- Configuration defaults and environment variable substitution (`internal/pkg/config`) remain consistent with existing behavior.
- Sensitive data (API keys) is referenced via env vars; do not hardcode secrets.
