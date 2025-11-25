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

### Key Features Implemented
1. **Responses API**: Full OpenAI Responses API with streaming, response storage, and conversation continuation
2. **Pass-Through Mode**: Bypass canonical conversion when frontdoor matches provider API type
3. **Rich Content**: Support for multimodal content (images, tool calls, etc.)
4. **Enhanced Events**: Comprehensive streaming event types for all APIs
5. **Anthropic Updates**: Extended thinking, computer use tools, beta features support
6. **Frontend Updates**: Control plane UI with Responses API explorer and feature indicators
7. **Codec Parity**: Clean bidirectional mapping between OpenAI and Anthropic formats
