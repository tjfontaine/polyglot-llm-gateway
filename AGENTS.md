# Polyglot LLM Gateway — Agent Guidelines

## Scope
This file applies to the entire repository unless a more specific `AGENTS.md` is introduced deeper in the tree.

## Project Overview
- **Purpose:** A Go-based gateway that normalizes LLM requests into a canonical shape, routes them to configured providers, and optionally serves a React/Vite control-plane UI.
- **Key server layers:**
  - `internal/frontdoor/*` map external protocols (OpenAI, Anthropic, Responses API) into canonical requests.
  - `internal/policy` selects providers based on routing rules from config.
  - `internal/provider/*` translate canonical requests into provider-specific API calls; streaming is handled through `CanonicalEvent` channels.
  - `internal/storage/*` persist conversation data when Responses APIs are enabled (SQLite or in-memory).
  - `internal/auth` and `internal/tenant` enable multi-tenant mode with bearer key hashing.
  - `internal/server` wires chi middleware, auth, and OpenTelemetry instrumentation.
  - `internal/controlplane` serves the admin API and React UI for observing gateway state.
- **Control plane:** `web/control-plane` is a React 19 + Vite app whose built assets are copied into `internal/controlplane/dist` by the `Makefile`.

## Control Plane Architecture

### Backend API (`internal/controlplane`)
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

### Request/Response Flow
The gateway uses a canonical domain model to abstract differences between API providers:

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Client     │────▶│  Frontdoor       │────▶│  Provider        │────▶│  Upstream   │
│  (Anthropic │     │  (Anthropic)     │     │  (OpenAI)        │     │  API        │
│   SDK)      │     │                  │     │                  │     │  (OpenAI)   │
└─────────────┘     └──────────────────┘     └──────────────────┘     └─────────────┘
                           │                        │
                           ▼                        ▼
                    CanonicalRequest          Provider-specific
                    CanonicalResponse         API Request
                    domain.APIError           domain.APIError
```

### Canonical Types (`internal/domain`)
- **`CanonicalRequest`** - Normalized request with model, messages, parameters
- **`CanonicalResponse`** - Normalized response with choices, usage, metadata
- **`CanonicalEvent`** - Streaming event with content deltas and tool calls
- **`APIError`** - Canonical error type that can be mapped to any API format

### Error Handling Architecture

Errors flow through the domain layer for consistent handling across different API types:

**1. API Clients return canonical errors:**
```go
// internal/api/openai/client.go
if apiErr, err := ParseErrorResponse(respBody); err == nil && apiErr != nil {
    return nil, apiErr.ToCanonical()  // Convert to domain.APIError
}
```

**2. Domain error types (`internal/domain/errors.go`):**
```go
type ErrorType string
const (
    ErrorTypeInvalidRequest ErrorType = "invalid_request"
    ErrorTypeAuthentication ErrorType = "authentication"
    ErrorTypeRateLimit      ErrorType = "rate_limit"
    ErrorTypeContextLength  ErrorType = "context_length"
    ErrorTypeMaxTokens      ErrorType = "max_tokens"
    // ...
)

type APIError struct {
    Type       ErrorType
    Code       ErrorCode
    Message    string
    StatusCode int
    SourceAPI  APIType
}
```

**3. Frontdoors format errors for their API type:**
```go
// internal/frontdoor/anthropic/handler.go
codec.WriteError(w, err, domain.APITypeAnthropic)

// internal/frontdoor/openai/handler.go
codec.WriteError(w, err, domain.APITypeOpenAI)
```

**4. Codec formats errors appropriately (`internal/codec/errors.go`):**
```go
func WriteError(w http.ResponseWriter, err error, apiType domain.APIType) {
    var formatter ErrorFormatter
    switch apiType {
    case domain.APITypeAnthropic:
        formatter = &AnthropicErrorFormatter{}
    case domain.APITypeOpenAI:
        formatter = &OpenAIErrorFormatter{}
    }
    resp := formatter.FormatError(err)
    w.WriteHeader(resp.StatusCode)
    w.Write(resp.Body)
}
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

### Adding New Error Types
1. Add the error type constant to `internal/domain/errors.go`
2. Add mapping in `internal/api/*/types.go` `ToCanonical()` methods
3. Add formatting in `internal/codec/errors.go` formatters
4. The frontdoors automatically use the correct format via `codec.WriteError()`

## Coding Conventions
- **Go style:** Keep code `gofmt`-clean and idiomatic Go. Prefer small, focused functions and return descriptive errors. Avoid introducing panics in request paths.
- **Configuration-driven behavior:** Respect the config structs in `internal/config` and the registry helpers when adding new frontdoors or providers; plug into `frontdoor.Registry`/`provider.Registry` instead of hardcoding wiring.
- **Canonical types first:** Work with `internal/domain` types at API boundaries. Frontdoors should fully populate `CanonicalRequest`/`CanonicalResponse` and let providers handle translation details.
- **Error handling:** API clients should convert provider-specific errors to `domain.APIError` using `ToCanonical()`. Frontdoors should use `codec.WriteError()` to format errors for their API type.
- **Streaming:** When adding streaming support, emit `CanonicalEvent` values and propagate provider errors via the channel before closing it.
- **Tests & fixtures:** Integration-style provider tests rely on go-vcr cassettes under `testdata/fixtures`. Use `VCR_MODE=record` with real API keys to refresh cassettes; default `go test` replays without network access.
- **Logging/telemetry:** Use the structured `slog` logger already configured in `cmd/gateway/main.go` and preserve OpenTelemetry middleware hooks in `internal/server` when adding new routes.
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
- **Artifacts:** The frontend build output in `web/control-plane/dist` is copied into `internal/controlplane/dist`; do not commit `node_modules`.

## Review Checklist for Changes
- Code is formatted (`gofmt`, ESLint for frontend) and lints cleanly.
- New routes or providers are registered via the appropriate registry helpers instead of manual router wiring.
- Tests are added or updated, especially when touching routing, provider translations, or storage persistence.
- Configuration defaults and environment variable substitution (`internal/config`) remain consistent with existing behavior.
- Sensitive data (API keys) is referenced via env vars; do not hardcode secrets.
