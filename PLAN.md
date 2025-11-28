# Integration Plan: Responses API & Enhanced API Support

## Overview
This plan addresses:
1. **Conditional Responses API Mounting** - Only mount on apps that explicitly enable it
2. **Latest OpenAI Responses API** - Implement the full OpenAI Responses API spec
3. **Enhanced Intermediate Format** - Support rich content types (multimodal, tool use, etc.)
4. **Improved Event Interfaces** - Better streaming event handling
5. **Latest Anthropic API** - Support current Anthropic API features
6. **Avoid Tragedy of the Commons** - Pass-through when frontdoor matches provider, only adapt when necessary

---

## Phase 1: Enhanced Domain Types ✅ COMPLETED

### 1.1 Rich Content Support in CanonicalRequest/Response ✅
- [x] Add `ContentPart` type for multimodal content (text, image, tool_use, tool_result)
- [x] Update `Message` to support array-based content via `RichContent`
- [x] Add `RawRequest` and `RawResponse` fields for pass-through support
- [x] Add source frontdoor/provider type tracking (`SourceAPIType`, `APIType()`)
- [x] Added `ToolCalls` and `ToolCallID` to Message struct
- [x] Added `ResponseFormat` type with JSON schema support
- [x] Updated `CanonicalEvent` with comprehensive event types

### 1.2 Enhanced Streaming Events ✅
- [x] Add event type field to `CanonicalEvent` (StreamEventType)
- [x] Support for content block events (start, delta, stop)
- [x] Support for message lifecycle events
- [x] Support for Responses API events
- [x] Add response ID tracking for Responses API
- [x] Add RawEvent field for pass-through mode

### 1.3 Provider Interface Updates ✅
- [x] Added `APIType()` method to Provider interface
- [x] Created `PassthroughProvider` interface for raw request handling
- [x] Created `ResponsesProvider` interface for native Responses API
- [x] Created `ProviderCapabilities` struct for capability discovery
- [x] Updated all providers (openai, anthropic, router, model_mapping, wrapper)

---

## Phase 2: Responses API Implementation ✅ COMPLETED

### 2.1 OpenAI Responses API Types ✅
- [x] Create comprehensive request/response types in `domain/responses.go`
- [x] Support input as string, array of messages, or array of items
- [x] Support instructions field
- [x] Support tools, tool_choice, parallel_tool_calls
- [x] Support truncation_strategy, response_format
- [x] Support metadata and previous_response_id

### 2.2 Responses API Handler Updates ✅
- [x] Implement full `/v1/responses` endpoint
- [x] Support streaming with SSE events
- [x] Implement response retrieval endpoint (`GET /v1/responses/{id}`)
- [x] Implement response cancellation (`POST /v1/responses/{id}/cancel`)
- [x] Support conversation continuations via previous_response_id

### 2.3 Responses API Events (Streaming) ✅
- [x] response.created
- [x] response.in_progress
- [x] response.completed
- [x] response.failed
- [x] response.output_item.added
- [x] response.output_item.done
- [x] response.content_part.added
- [x] response.content_part.done
- [x] response.output_text.delta
- [x] response.output_text.done

### 2.4 Storage Updates ✅
- [x] Added `ResponseStore` interface
- [x] Implemented response storage in SQLite
- [x] Implemented response storage in Memory

---

## Phase 3: Pass-Through Mode (Tragedy of the Commons) ✅ COMPLETED

### 3.1 Provider/Frontdoor Type Matching ✅
- [x] Add `SourceAPIType` to CanonicalRequest
- [x] Add `APIType()` method to Provider interface
- [x] Frontdoors set source type and capture raw request body

### 3.2 Pass-Through Provider Implementation ✅
- [x] Created `PassthroughProvider` wrapper in `provider/passthrough.go`
- [x] Supports bypassing canonical conversion when types match
- [x] Preserves original request body via `RawRequest` field
- [x] Preserves original response body via `RawResponse` field
- [x] Parses raw responses to canonical format for audit/recording

### 3.3 Recording Both Sides ✅
- [x] Raw incoming request stored in `CanonicalRequest.RawRequest`
- [x] Canonical conversion still happens for recording
- [x] Raw provider response stored in `CanonicalResponse.RawResponse`
- [x] Raw response can be returned directly to client when types match

---

## Phase 4: Configuration Updates ✅ COMPLETED

### 4.1 App Configuration ✅
- [x] `enable_responses` already works correctly per-app
- [x] Added `enable_passthrough` flag for providers

### 4.2 Provider Configuration ✅
- [x] `supports_responses` already exists for native Responses API support
- [x] Added `enable_passthrough` option to wrap providers with PassthroughProvider
- [x] Updated provider registry to automatically wrap with pass-through when enabled

---

## Phase 5: Anthropic API Alignment ✅ COMPLETED

### 5.1 Review Current Anthropic Support ✅
- [x] Added `latestVersion` constant (2024-10-22)
- [x] Added extended thinking support (`ThinkingConfig`, `thinking` content blocks)
- [x] Added computer use tool types (`computer_20241022`, `text_editor_20241022`, `bash_20241022`)
- [x] Added beta features header support (`anthropic-beta`)

### 5.2 Content Type Parity ✅
- [x] Image content blocks already supported
- [x] Tool_use/tool_result blocks already supported
- [x] Cache control already supported (`cache_control`)
- [x] Added thinking content block type

---

## Implementation Summary ✅ ALL PHASES COMPLETED

### New Files Created
- `internal/domain/content.go` - Rich content types (multimodal, images, tool calls)
- `internal/domain/responses.go` - OpenAI Responses API types and streaming events
- `internal/provider/passthrough.go` - Pass-through wrapper for API type matching

### Modified Files
- `internal/domain/types.go` - Enhanced Message, CanonicalRequest, CanonicalResponse, CanonicalEvent
- `internal/domain/interfaces.go` - Enhanced Provider interface with APIType(), PassthroughProvider, ResponsesProvider
- `internal/config/config.go` - Added `enable_passthrough` to provider config
- `internal/frontdoor/responses/handler.go` - Complete Responses API implementation
- `internal/frontdoor/registry.go` - Added response retrieval and cancellation routes
- `internal/frontdoor/openai/handler.go` - Pass-through support
- `internal/frontdoor/anthropic/handler.go` - Pass-through support
- `internal/storage/storage.go` - Added ResponseStore interface and ResponseRecord
- `internal/storage/memory/store.go` - Implemented ResponseStore
- `internal/storage/sqlite/store.go` - Implemented ResponseStore with schema
- `internal/provider/registry.go` - Auto-wrap with PassthroughProvider when enabled
- `internal/provider/wrapper.go` - Added APIType() method
- `internal/provider/model_mapping.go` - Added APIType() method
- `internal/provider/openai/provider.go` - Added APIType() method
- `internal/provider/anthropic/provider.go` - Added APIType() method
- `internal/policy/router.go` - Added APIType() method
- `internal/api/anthropic/client.go` - Added beta features header support
- `internal/api/anthropic/types.go` - Added extended thinking, computer use tools
- `internal/controlplane/server.go` - Added responses API endpoints and enhanced overview
- `internal/codec/openai/codec.go` - Enhanced request/response mapping with tool calls, top_p, etc.
- `internal/codec/anthropic/codec.go` - Enhanced mapping with tool calls, finish reason conversion
- `web/control-plane/src/App.tsx` - Added Responses API explorer and feature indicators

---

## Phase 6: Frontend Updates & Codec Improvements ✅ COMPLETED

### 6.1 Frontend Updates ✅
- [x] Updated control plane UI to show `enable_responses` status on apps
- [x] Updated control plane UI to show `enable_passthrough` status on providers
- [x] Added "Responses API" and "Passthrough mode" indicators in header
- [x] Added Responses API explorer tab alongside conversations
- [x] Implemented response listing and detail view in frontend
- [x] Added control plane API endpoint for listing/viewing responses

### 6.2 OpenAI Codec Improvements ✅
- [x] Added tool call support in request/response conversion
- [x] Added top_p parameter support
- [x] Added stop sequences support
- [x] Added tool_choice support
- [x] Added response_format support
- [x] Added system_fingerprint support
- [x] Support for system prompt and instructions fields

### 6.3 Anthropic Codec Improvements ✅
- [x] Added tool call support (tool_use to function tool calls)
- [x] Added top_p parameter support
- [x] Added stop_sequences support
- [x] Added tool_choice conversion (auto/any/tool)
- [x] Added finish_reason mapping (stop_reason <-> finish_reason)
- [x] Support for system prompt and instructions as system blocks
- [x] Support for tool messages mapping (OpenAI "tool" role -> Anthropic tool_result)

---

## Phase 7: Control Plane Reorganization ✅ COMPLETED

### 7.1 Backend API Improvements ✅
- [x] Added `ListResponses` method to `ResponseStore` interface
- [x] Implemented `ListResponses` in SQLite store
- [x] Implemented `ListResponses` in Memory store
- [x] Created unified `/api/interactions` endpoint merging conversations and responses
- [x] Created `/api/interactions/{id}` detail endpoint
- [x] Interactions sorted by `updated_at` descending with optional type filtering

### 7.2 Frontend Reorganization ✅
- [x] Restructured frontend into components, pages, hooks, and types directories
- [x] Created shared Layout component with navigation
- [x] Created reusable UI components (Pill, InfoCard, StatusBadge, etc.)
- [x] Created Dashboard page with overview cards linking to detail pages
- [x] Created Topology page for apps & providers configuration
- [x] Created Routing page for routing rules & tenant configuration
- [x] Created unified Data page for all interactions (conversations + responses)
- [x] Removed separate Conversations and Responses pages in favor of unified view
- [x] Updated navigation to reflect new page structure

### 7.3 Documentation Updates ✅
- [x] Updated AGENTS.md with control plane architecture documentation
- [x] Documented unified interactions model and design principles
- [x] Documented frontend structure and page responsibilities

---

### Key Features Implemented
1. **Responses API**: Full OpenAI Responses API with streaming, response storage, and conversation continuation
2. **Pass-Through Mode**: Bypass canonical conversion when frontdoor matches provider API type
3. **Rich Content**: Support for multimodal content (images, tool calls, etc.)
4. **Enhanced Events**: Comprehensive streaming event types for all APIs
5. **Anthropic Updates**: Extended thinking, computer use tools, beta features support
6. **Frontend Updates**: Control plane UI with unified data explorer and feature indicators
7. **Codec Parity**: Clean bidirectional mapping between OpenAI and Anthropic formats
8. **Unified Interactions**: Single view for both conversations and responses with filtering

---

## Phase 8: Bug Fixes & Enhanced Testing ✅ COMPLETED

### 8.1 Missing Endpoint Fixes ✅
- [x] Added `/v1/messages/count_tokens` endpoint to Anthropic frontdoor
- [x] Added `CountTokensRequest` and `CountTokensResponse` types in `api/anthropic/types.go`
- [x] Added `CountTokens` method to Anthropic API client with beta header support
- [x] Added `CountTokens` method to Anthropic provider
- [x] Added `CountTokens` delegation to `ModelMappingProvider`
- [x] Added `CountTokens` delegation to policy `Router`
- [x] Registered count_tokens route in frontdoor registry

### 8.2 Model Routing Configuration Fix ✅
- [x] Fixed frontdoor registry to check for `Fallback` in addition to `PrefixProviders` and `Rewrites`
- [x] Previously, configs with only `Fallback` (no `Rewrites`) would not create `ModelMappingProvider`

### 8.3 Comprehensive Test Coverage ✅
- [x] Added VCR-based tests for Anthropic `CountTokens` endpoint
- [x] Added mock server tests for `CountTokens` with header verification
- [x] Added integration tests for full routing chain (`ModelMappingProvider` -> `Router` -> `Provider`)
- [x] Added tests for fallback-only configuration
- [x] Added tests for multiple prefix rules with different providers
- [x] Added tests for exact match precedence over prefix match
- [x] Added tests for response model rewriting
- [x] Added tests for `CountTokens` delegation through `ModelMappingProvider`
- [x] Added tests for slash-prefixed routing (e.g., `openai/gpt-4o`)
- [x] Added tests for combined config (rewrites + prefix providers + fallback)
- [x] Added policy router tests for `CountTokens` delegation
- [x] Added frontdoor registry tests for all handler types
- [x] Added integration test with model rewriting through full request flow

### Test Files Added/Modified
- `internal/provider/anthropic/provider_test.go` - Added VCR and mock tests for CountTokens
- `internal/provider/anthropic/testdata/fixtures/anthropic_count_tokens.yaml` - VCR cassette
- `internal/provider/model_mapping_test.go` - Extended with comprehensive rewrite tests
- `internal/provider/integration_test.go` - New file with full routing chain tests
- `internal/policy/router_test.go` - Added CountTokens and APIType tests
- `internal/frontdoor/registry_test.go` - New file with registry and integration tests

---

## Phase 9: Cross-Provider Token Counting ✅ COMPLETED

### 9.1 Token Counter Interface ✅
- [x] Created `TokenCountRequest` and `TokenCountResponse` types in `domain/tokens.go`
- [x] Created `TokenCounter` interface for provider-agnostic token counting
- [x] Created `TokenCountTool` type for tool definitions in token counting

### 9.2 Token Counter Registry ✅
- [x] Created `internal/tokens/registry.go` with registry pattern
- [x] Implemented `Estimator` as fallback for unsupported models
- [x] Implemented `ModelMatcher` for provider-based model matching

### 9.3 Provider-Specific Adapters ✅
- [x] Created `AnthropicCounter` in `internal/tokens/anthropic.go`
  - Uses native Anthropic `count_tokens` API
  - Supports claude-* model prefixes
  - Provides exact token counts (not estimated)
- [x] Created `OpenAICounter` in `internal/tokens/openai.go`
  - Uses `github.com/tiktoken-go/tokenizer` for accurate token counting
  - Library has native support for GPT-5, GPT-5-Mini, GPT-5-Nano, GPT-4.1, O1, O3, O4-Mini
  - Supports gpt-*, o1-o6, text-embedding-*, davinci/curie/babbage/ada models
  - Supports future models: GPT-5.1+, GPT-6+, O5+, etc. via o200k_base fallback
  - Provides exact token counts using appropriate encodings:
    - o200k_base: GPT-5+, GPT-4.1, GPT-4o, O-series models
    - cl100k_base: GPT-4, GPT-3.5, text-embedding
    - p50k_base: text-davinci-002/003
    - r50k_base: legacy completion models
  - Includes encoding caching for performance

### 9.4 Frontdoor Integration ✅
- [x] Updated Anthropic frontdoor `HandleCountTokens` to:
  - Use native provider when available (pass-through)
  - Fall back to estimation for other providers
  - Return Anthropic-compatible JSON response format

### 9.5 Test Coverage ✅
- [x] Added comprehensive tests for `Estimator`
- [x] Added comprehensive tests for `OpenAICounter`
- [x] Added tests for `Registry` with multiple counters
- [x] Added tests for `ModelMatcher`
- [x] Added benchmarks for token estimation performance

### New Files Created
- `internal/domain/tokens.go` - Token counting types and interface
- `internal/tokens/registry.go` - Registry and Estimator
- `internal/tokens/anthropic.go` - Anthropic native counter
- `internal/tokens/openai.go` - OpenAI tiktoken-style estimator
- `internal/tokens/tokens_test.go` - Comprehensive tests

---

## Phase 10: CountTokens Model-Based Routing Fix ✅ COMPLETED

### 10.1 Problem
The `count_tokens` endpoint was not respecting model routing configuration. When requests came through an Anthropic frontdoor configured to route to OpenAI (e.g., for model rewriting), the `CountTokens` method would incorrectly fall back to any provider that supported the interface, ignoring the routing rules.

Example config that triggered the bug:
```yaml
- name: claude
  frontdoor: anthropic
  path: /claude
  model_routing:
    fallback:
      provider: openai
      model: gpt-5-mini
```

This would cause `count_tokens` requests to use the Anthropic provider directly (because it implements `CountTokens`), bypassing the OpenAI routing entirely.

### 10.2 Fix
- [x] Updated `ModelMappingProvider.CountTokens` to parse the model from the request body
- [x] Updated `ModelMappingProvider.CountTokens` to use `selectProvider()` for model-based routing
- [x] Updated `Router.CountTokens` to parse the model and use `Route()` for routing
- [x] Both now return an error when the routed provider doesn't support `CountTokens`
- [x] This allows the frontdoor handler to fall back to token estimation

### 10.3 Behavior Change
**Before:** `CountTokens` would try default provider first, then fall back to any provider supporting `CountTokens`
**After:** `CountTokens` routes based on model (same as `Complete`), returns error if routed provider doesn't support it

### 10.4 Test Updates
- [x] Updated `TestRouter_CountTokens` to test model-based routing
- [x] Added test cases for routing to provider with `CountTokens` support
- [x] Added test cases for error when routed provider doesn't support `CountTokens`

---

## Phase 11: CountTokens Fallback & Error Handling Improvements ✅ COMPLETED

### 11.1 Problem
When requests are routed through `ModelMappingProvider` to a provider that doesn't support native `CountTokens` (e.g., OpenAI), the handler was:
1. Returning HTTP 500 instead of falling back to token estimation
2. Returning HTTP 500 for all API errors instead of appropriate status codes (400, 401, 429, etc.)

### 11.2 CountTokens Fallback Fix
- [x] Updated `HandleCountTokens` to detect "count_tokens not supported" errors and fall back to token estimation
- [x] Added `tokenCounter` field to Handler struct with `tokens.Registry`
- [x] Integrated `OpenAICounter` (tiktoken-based) for accurate OpenAI model token counting
- [x] Added `canonicalToTokenRequest` helper for converting canonical requests to token count requests
- [x] Fallback now uses proper tiktoken for OpenAI models, estimation for others

### 11.3 Error Status Code Fix
- [x] Added `writeAPIError` function to map Anthropic API errors to HTTP status codes
- [x] Maps error types to appropriate status codes:
  - `invalid_request_error` → 400 Bad Request
  - `authentication_error` → 401 Unauthorized
  - `permission_error` → 403 Forbidden
  - `not_found_error` → 404 Not Found
  - `rate_limit_error` → 429 Too Many Requests
  - `overloaded_error` → 503 Service Unavailable
  - `api_error` → 500 Internal Server Error
- [x] Updated `HandleMessages`, `handleStream`, and `HandleCountTokens` to use `writeAPIError`
- [x] Error responses now include proper Anthropic JSON format with type and message

### 11.4 Files Modified
- `internal/frontdoor/anthropic/handler.go` - Added token counter, improved error handling

---

## Phase 12: Cross-API Error Translation ✅ COMPLETED

### 12.1 Problem
When requests are routed between different API types (e.g., Anthropic frontdoor → OpenAI provider, or vice versa), error responses were not being translated to the expected format for clients. This caused confusion as clients would receive error formats from a different API than they were using.

### 12.2 Anthropic Frontdoor Improvements
- [x] Added `translateOpenAIError()` to convert OpenAI errors to Anthropic format
- [x] Added `translateErrorMessage()` for OpenAI→Anthropic message translation
- [x] Maps OpenAI error types to Anthropic equivalents:
  - `invalid_request_error` → `invalid_request_error`
  - `authentication_error` → `authentication_error`
  - `permission_denied` → `permission_error`
  - `not_found` → `not_found_error`
  - `rate_limit_error` → `rate_limit_error`
  - `server_error` → `api_error`
- [x] Translates common error messages (max_tokens, context length, rate limit)

### 12.3 OpenAI Frontdoor Improvements
- [x] Added `writeAPIError()` function with proper error translation
- [x] Added `translateAnthropicError()` to convert Anthropic errors to OpenAI format
- [x] Maps Anthropic error types to OpenAI equivalents:
  - `invalid_request_error` → `invalid_request_error`
  - `authentication_error` → `authentication_error` (code: `invalid_api_key`)
  - `permission_error` → `permission_denied`
  - `not_found_error` → `not_found` (code: `model_not_found`)
  - `rate_limit_error` → `rate_limit_error` (code: `rate_limit_exceeded`)
  - `overloaded_error` → `service_unavailable`
  - `api_error` → `server_error`
- [x] Returns proper OpenAI JSON error format: `{"error": {"message": "...", "type": "...", "code": "..."}}`

### 12.4 Natural Error Responses
Error messages are now translated to feel native to each API:

**For Anthropic clients:**
- "Could not finish the message because max_tokens..." → "The response was truncated because max_tokens was reached. Please increase max_tokens for longer responses."
- Context length errors get Anthropic-style messaging

**For OpenAI clients:**
- Anthropic errors get OpenAI-style messaging and proper error codes
- Status codes are properly mapped (400, 401, 403, 404, 429, 503, 500)

### 12.5 Files Modified
- `internal/frontdoor/anthropic/handler.go` - OpenAI→Anthropic error translation
- `internal/frontdoor/openai/handler.go` - Anthropic→OpenAI error translation

---

## Phase 13: Domain Error Abstraction ✅ COMPLETED

### 13.1 Problem
Error handling was duplicated across frontdoors with inline translation logic. This made it difficult to maintain consistency and add new error types.

### 13.2 Solution: Canonical Error Types
Created a domain-level error abstraction that:
- Defines canonical error types that are API-agnostic
- Provides conversion from API-specific errors to canonical errors
- Provides formatting from canonical errors to API-specific responses

### 13.3 New Domain Types (`internal/domain/errors.go`)
- `ErrorType` - Canonical error categories (invalid_request, authentication, rate_limit, etc.)
- `ErrorCode` - Specific error codes (context_length_exceeded, rate_limit_exceeded, etc.)
- `APIError` - Canonical error struct with type, code, message, and source API
- Convenience constructors: `ErrInvalidRequest()`, `ErrRateLimit()`, `ErrContextLength()`, etc.

### 13.4 API Client Updates
- Added `ToCanonical()` method to `internal/api/anthropic/types.go` `APIError`
- Added `ToCanonical()` method to `internal/api/openai/types.go` `APIError`
- Both methods map provider-specific error types/codes to domain error types
- API clients now return `domain.APIError` instead of provider-specific errors

### 13.5 Codec Error Formatting (`internal/codec/errors.go`)
- `ErrorFormatter` interface for API-specific error formatting
- `OpenAIErrorFormatter` - Formats errors as OpenAI JSON
- `AnthropicErrorFormatter` - Formats errors as Anthropic JSON
- `WriteError(w, err, apiType)` - Central function for writing error responses

### 13.6 Frontdoor Simplification
- Removed all inline error translation functions from frontdoors
- Both handlers now use simple `codec.WriteError(w, err, domain.APIType*)`
- Error formatting is fully delegated to the codec layer

### 13.7 Architecture Flow
```
Provider Error → ToCanonical() → domain.APIError → codec.WriteError() → API Response
```

### 13.8 Files Created
- `internal/domain/errors.go` - Canonical error types and constructors
- `internal/codec/errors.go` - Error formatters and WriteError function

### 13.9 Files Modified
- `internal/api/anthropic/types.go` - Added ToCanonical() method
- `internal/api/anthropic/client.go` - Return canonical errors
- `internal/api/openai/types.go` - Added ToCanonical() method  
- `internal/api/openai/client.go` - Return canonical errors
- `internal/frontdoor/anthropic/handler.go` - Use codec.WriteError()
- `internal/frontdoor/openai/handler.go` - Use codec.WriteError()
- `AGENTS.md` - Documented comprehensive domain model architecture

### 13.10 Documentation Updates
Updated `AGENTS.md` with comprehensive domain model architecture documentation covering:
- **Core Architecture Pattern**: Visual diagram showing the Frontdoor → Domain → Provider flow
- **The Pattern: Decode → Canonical → Encode**: Table showing how all data types (requests, responses, streaming, errors, token counts) follow the same pattern
- **Canonical Types**: Documented `CanonicalRequest`, `CanonicalResponse`, `CanonicalEvent`, `APIError`
- **Codec Layer**: Documented the bidirectional translation between API-specific formats and canonical types
- **Request Flow Example**: Complete code walkthrough of Anthropic client → OpenAI provider
- **Error Flow**: End-to-end error handling from provider to client
- **Error Type Mapping**: Table mapping domain errors to Anthropic/OpenAI formats and HTTP status codes
- **Pass-Through Optimization**: How to bypass canonical conversion when frontdoor matches provider
- **Adding New API Support**: Step-by-step guide for adding new API types (e.g., Google Gemini)

---

## Phase 14: Model Mapping Streaming & OpenAI Responses API ✅ COMPLETED

### 14.1 Problem
1. **Streaming model tracking**: The `served_model` in logs was showing the requested model instead of the actual model being used by the provider
2. **OpenAI API type**: Requests routed to OpenAI were using Chat Completions API (`/v1/chat/completions`) instead of the newer Responses API (`/v1/responses`)

### 14.2 Streaming Model Capture
- [x] Updated `APIChunkToCanonical` in `codec/openai/codec.go` to capture `Model` and `ResponseID` from streaming chunks
- [x] Updated `APIChunkToCanonical` to capture `FinishReason` from streaming chunks
- [x] Updated `ModelMappingProvider.Stream` to wrap upstream channel and rewrite model in events when `rewrite_response_model: true`
- [x] Updated Anthropic frontdoor `handleStream` to capture `servedModel` from streaming events
- [x] Logs now correctly show `served_model` as the actual model used (e.g., the rewritten model name)

### 14.3 OpenAI Responses API Client
- [x] Added Responses API types to `api/openai/types.go`:
  - `ResponsesRequest` - request format for `/v1/responses`
  - `ResponsesResponse` - response format
  - `ResponsesInputItem`, `ResponsesContentPart` - input types
  - `ResponsesOutputItem` - output types
  - `ResponsesUsage`, `ResponsesError` - usage and error types
  - `ResponsesTool`, `TruncationStrategy` - configuration types
- [x] Added client methods to `api/openai/client.go`:
  - `CreateResponse()` - non-streaming Responses API call
  - `StreamResponse()` - streaming Responses API call
  - `responsesStreamReader()` - SSE event parser for Responses API

### 14.4 OpenAI Provider Responses API Support
- [x] Added `WithResponsesAPI(bool)` option to OpenAI provider
- [x] Updated `Provider.Complete()` to use Responses API when enabled
- [x] Updated `Provider.Stream()` to use Responses API when enabled
- [x] Added `canonicalToResponsesRequest()` - converts canonical request to Responses API format
- [x] Added `responsesResponseToCanonical()` - converts Responses API response to canonical format
- [x] Added `streamWithResponses()` - handles streaming from Responses API with proper event parsing

### 14.5 Configuration
- [x] Added `use_responses_api` field to `ProviderConfig` in `config/config.go`
- [x] Updated provider registry to apply `WithResponsesAPI` option when configured
- [x] Updated `config.yaml` to enable Responses API for OpenAI provider:
  ```yaml
  providers:
    - name: openai
      type: openai
      api_key: ${OPENAI_API_KEY}
      use_responses_api: true
  ```

### 14.6 Files Modified
- `internal/codec/openai/codec.go` - Capture model/response ID from streaming chunks
- `internal/provider/model_mapping.go` - Rewrite model in streaming events
- `internal/frontdoor/anthropic/handler.go` - Capture served model from streams
- `internal/api/openai/types.go` - Add Responses API types
- `internal/api/openai/client.go` - Add Responses API client methods
- `internal/provider/openai/provider.go` - Add Responses API support with conversion functions
- `internal/provider/registry.go` - Pass `use_responses_api` config to provider
- `internal/config/config.go` - Add `UseResponsesAPI` field
- `config.yaml` - Enable Responses API for OpenAI

---

## Phase 15: 2025 Spec Alignment ✅ COMPLETED

Implementing features from the "Gateway Implementation Roadmap: Aligning with 2025 Spec" to improve interoperability between OpenAI v1/responses and Anthropic v1/messages APIs.

### 15.1 Anthropic 529 Overload Retry/Backoff ✅
- [x] Added exponential backoff retry logic for Anthropic 529 (overloaded) errors
- [x] Configurable max retries (default: 2 attempts)
- [x] Backoff delays: 500ms, 1s, 2s (exponential with 5s max cap)
- [x] Retry logic applies to both `Complete` and `Stream` methods
- [x] Context cancellation respected during backoff waits
- [x] Returns 503 Service Unavailable after exhausting retries
- [x] Added `WithMaxRetries()` and `WithLogger()` provider options

### 15.2 Responses API Status Mapping for Tool Calls ✅
- [x] Map Anthropic `tool_use` stop_reason to OpenAI Responses API `status: incomplete`
- [x] Map OpenAI `tool_calls` finish_reason to `status: incomplete`
- [x] Updated `ToResponsesAPIResponse()` in `domain/responses.go`
- [x] Updated streaming handler to capture `FinishReason` and set appropriate status
- [x] Updated storage record status to reflect actual response status

### 15.3 Rate Limit Header Normalization ✅
- [x] Added `RateLimitInfo` struct to `domain/types.go` for capturing upstream rate limits
- [x] Updated Anthropic API client to parse rate limit headers from responses
- [x] Added `RateLimitHeaders` struct and `parseRateLimitHeaders()` function
- [x] Updated `APIResponseToCanonicalWithRateLimits()` in Anthropic codec
- [x] Added `RateLimitNormalizingMiddleware` in `server/middleware.go`
- [x] Updated Anthropic frontdoor handler to write normalized rate limit headers
- [x] Header mapping:
  - `anthropic-ratelimit-requests-limit` → `x-ratelimit-limit-requests`
  - `anthropic-ratelimit-requests-remaining` → `x-ratelimit-remaining-requests`
  - `anthropic-ratelimit-requests-reset` → `x-ratelimit-reset-requests`
  - `anthropic-ratelimit-tokens-limit` → `x-ratelimit-limit-tokens`
  - `anthropic-ratelimit-tokens-remaining` → `x-ratelimit-remaining-tokens`
  - `anthropic-ratelimit-tokens-reset` → `x-ratelimit-reset-tokens`

### 15.4 Image URL-to-Base64 Conversion ✅
- [x] Created `ImageFetcher` utility in `internal/codec/images.go`
- [x] Support for fetching HTTP/HTTPS image URLs
- [x] Support for parsing data: URLs (already base64 encoded)
- [x] Configurable HTTP client and max image size (default 20MB)
- [x] Media type detection from Content-Type header or URL extension
- [x] Support for image/jpeg, image/png, image/gif, image/webp
- [x] Added `CanonicalToAPIRequestWithImageFetching()` to Anthropic codec
- [x] Added `convertRichContentToAnthropic()` helper for multimodal content

### 15.5 Tool Calling Support in Streaming ✅
- [x] Enhanced Anthropic provider streaming to handle `tool_use` content blocks
- [x] Added `content_block_start` event handling for tool call initiation
- [x] Added `input_json_delta` handling for streaming tool call arguments
- [x] Added `content_block_stop` event handling for tool call completion
- [x] Capture `stop_reason` from `message_delta` and map to `FinishReason`
- [x] Added `mapStopReason()` helper for Anthropic→OpenAI finish reason mapping
- [x] Updated Responses API streaming handler to:
  - Emit `function_call` output items with proper SSE events
  - Stream `function_call_arguments.delta` events
  - Stream `function_call_arguments.done` events
  - Properly order output items (message first, then function calls)
  - Handle mixed text content and tool calls in same response

### 15.6 Files Created
- `internal/codec/images.go` - Image URL fetching and base64 conversion utility

### 15.7 Files Modified
- `internal/provider/anthropic/provider.go` - Added 529 retry/backoff logic, tool call streaming
- `internal/domain/responses.go` - Updated status mapping for tool calls
- `internal/domain/types.go` - Added `RateLimitInfo` struct
- `internal/api/anthropic/client.go` - Added rate limit header parsing
- `internal/codec/anthropic/codec.go` - Added image conversion and rate limit support
- `internal/server/middleware.go` - Added rate limit normalization middleware
- `internal/frontdoor/anthropic/handler.go` - Added rate limit header writing
- `internal/frontdoor/responses/handler.go` - Complete rewrite of streaming handler with tool call support

---

## Phase 16: Comprehensive Testing & Web UI Improvements ✅ COMPLETED

### 16.1 Anthropic Provider Tests ✅
- [x] Added tests for 529 retry/backoff success after transient failures
- [x] Added tests for 529 retry exhaustion with proper error
- [x] Added tests for non-retryable errors (auth, etc.) not retrying
- [x] Added tests for rate limit header capture in responses
- [x] Added tests for tool call extraction from non-streaming responses
- [x] Added tests for tool call streaming events
- [x] Added tests for `mapStopReason` function

### 16.2 Image URL Conversion Tests ✅
- [x] Added tests for HTTP URL fetching and base64 conversion
- [x] Added tests for data URL parsing (base64 encoded)
- [x] Added tests for media type normalization (image/jpg → image/jpeg)
- [x] Added tests for invalid URL schemes
- [x] Added tests for HTTP errors
- [x] Added tests for oversized images
- [x] Added tests for unsupported media types
- [x] Added tests for Content-Type header parsing
- [x] Added unit tests for `inferMediaType`, `isSupportedMediaType`, `normalizeMediaType`, `parseDataURL`

### 16.3 Rate Limit Middleware Tests ✅
- [x] Added tests for rate limit header normalization
- [x] Added tests for partial rate limit info (requests but not tokens)
- [x] Added tests for no rate limits in context
- [x] Added integration tests for SetRateLimits + middleware chain

### 16.4 Responses API Handler Tests ✅
- [x] Added tests for non-streaming response creation
- [x] Added tests for streaming text content with SSE events
- [x] Added tests for streaming tool calls with all events:
  - `response.output_item.added` for function_call
  - `response.function_call_arguments.delta`
  - `response.function_call_arguments.done`
  - `response.output_item.done`
- [x] Added tests for mixed content (text + tool calls)
- [x] Added tests for non-streaming tool calls
- [x] Added tests for `incomplete` status mapping

### 16.5 Web UI Tool Call Display ✅
- [x] Updated `InteractionDetail` type to include typed `ResponseData`
- [x] Added `ResponseOutputItem` and `ResponseContentPart` types
- [x] Created `OutputItemCard` component for rendering output items
- [x] Created `ResponseSection` component for structured response display
- [x] Tool calls displayed with:
  - Violet color scheme to distinguish from messages
  - Function name and call ID
  - Pretty-printed JSON arguments
  - Status badge for completion status
- [x] Tool results displayed with:
  - Cyan color scheme
  - Call ID reference
  - Output content
- [x] Added tool call indicator badge in interaction list (for `incomplete` status)
- [x] Added token usage display when available

### 16.6 Test Files Created
- `internal/codec/images_test.go` - Image URL conversion tests
- `internal/server/middleware_test.go` - Rate limit middleware tests
- `internal/frontdoor/responses/handler_test.go` - Responses API handler tests

### 16.7 Test Files Modified
- `internal/provider/anthropic/provider_test.go` - Added retry/backoff, rate limits, tool call tests

### 16.8 Web UI Files Modified
- `web/control-plane/src/types/index.ts` - Added response output types
- `web/control-plane/src/pages/Data.tsx` - Added tool call rendering components

---

## Phase 17: Bug Fixes & Spec Alignment ✅ COMPLETED

### 17.1 OpenAI Responses API stream_options Fix ✅
- [x] Fixed error "Unknown parameter: 'stream_options.include_usage'" when streaming via Responses API
- [x] Root cause: `StreamResponse()` in `api/openai/client.go` was auto-adding `stream_options.include_usage` which is not supported by the Responses API
- [x] The Responses API uses SSE events and automatically includes usage in `response.done` events
- [x] Fix: Removed auto-population of `stream_options` from `StreamResponse()` method
- [x] Chat Completions API streaming still correctly sets `stream_options` (it's only unsupported in Responses API)

### 17.2 OpenAI Responses API Spec v2.0 Alignment ✅
Aligned streaming events with OpenAI Responses API Specification v2.0:

**Streaming Event Changes:**
- [x] `response.created` - Now returns `{"id": "...", "model": "..."}` (simplified from full response object)
- [x] `response.output_item.added` - Changed `output_index` to `item_index` per spec
- [x] `response.output_item.delta` - Unified event for both text (`delta.content`) and tool args (`delta.arguments`)
- [x] `response.output_item.done` - Changed `output_index` to `item_index` per spec
- [x] `response.done` - Replaced `response.completed` with `{"usage": {...}, "finish_reason": "..."}` format

**Removed Non-Spec Events:**
- [x] Removed `response.in_progress` (not in spec)
- [x] Removed `response.content_part.added` (not in spec)
- [x] Removed `response.content_part.done` (not in spec)
- [x] Removed `response.output_text.delta` (replaced by `response.output_item.delta`)
- [x] Removed `response.output_text.done` (not in spec)
- [x] Removed `response.function_call_arguments.delta` (replaced by `response.output_item.delta`)
- [x] Removed `response.function_call_arguments.done` (not in spec)
- [x] Removed `response.completed` (replaced by `response.done`)

**Updated Domain Types:**
- [x] `ResponseCreatedEvent` - Simplified to `{ID, CreatedAt, Model}`
- [x] `OutputItemAddedEvent` - Uses `ItemIndex` instead of `OutputIndex`
- [x] `OutputItemDeltaEvent` - New unified type with `Delta.Content` and `Delta.Arguments`
- [x] `OutputItemDoneEvent` - Uses `ItemIndex` instead of `OutputIndex`
- [x] `ResponseDoneEvent` - New type with `Usage` and `FinishReason`

**Updated StreamEventType Constants:**
- [x] Added `EventTypeResponseOutputItemDelta`
- [x] Added `EventTypeResponseDone`
- [x] Removed obsolete event type constants

**Provider Updates:**
- [x] Updated OpenAI provider to consume new spec-compliant events from upstream
- [x] Added backwards compatibility for legacy event names

### 17.3 Files Modified
- `internal/api/openai/client.go` - Removed stream_options auto-add from StreamResponse()
- `internal/domain/responses.go` - Updated event types to match spec
- `internal/domain/types.go` - Updated StreamEventType constants
- `internal/frontdoor/responses/handler.go` - Updated streaming to emit spec-compliant events
- `internal/provider/openai/provider.go` - Updated to consume spec-compliant events
- `internal/frontdoor/responses/handler_test.go` - Updated tests for new event format
