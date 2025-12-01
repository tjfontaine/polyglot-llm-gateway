# Event Pipeline & Storage Unification Plan

## Overview

This plan addresses the fragmented event pipeline and storage mechanisms in the gateway. Currently, the system has multiple code paths for recording interactions, inconsistent ID generation, and the Responses API bypasses the Interaction Recorder (IR) layer entirely. This leads to:

- **Tragedy of the commons**: Each frontdoor implements its own recording logic
- **Inconsistent IDs**: Sometimes we use provider IDs, sometimes gateway IDs
- **Fragmented storage**: Conversations, Responses, and Interactions stored separately
- **Missing observability**: Not all frontdoors emit pipeline events

## Goals

1. **Single source of truth**: All API interactions flow through the IR layer
2. **Gateway-owned IDs**: Never use provider IDs as primary keys; store them as metadata
3. **Unified storage**: Deprecate `ConversationStore` and `ResponseStore` in favor of `InteractionStore`
4. **Consistent event logging**: All frontdoors emit the same pipeline events
5. **Preserved shadow functionality**: Shadow results continue to link via stable `InteractionID`

---

## Current State Analysis

### Problem 1: Inconsistent ID Generation

**OpenAI Frontdoor** (`internal/openai/frontdoor.go:153-160`):

```go
interactionID := req.Metadata["interaction_id"]
if interactionID == "" {
    if requestID != "" {
        interactionID = "int_" + strings.ReplaceAll(requestID, "-", "")
    } else {
        interactionID = "int_" + strings.ReplaceAll(uuid.New().String(), "-", "")
    }
    req.Metadata["interaction_id"] = interactionID
}
```

But then `RecordInteraction` (`internal/conversation/interaction_recorder.go:64-66`) **overwrites it**:

```go
interactionID := "int_" + uuid.New().String()
if params.CanonicalResp != nil && params.CanonicalResp.ID != "" {
    interactionID = params.CanonicalResp.ID  // PROBLEM: Uses provider's ID!
}
```

**Result**: The interaction ID used in storage may be a provider-generated ID (e.g., `chatcmpl-xxx` or `msg_xxx`), not our gateway-owned ID.

### Problem 2: Responses API Bypasses IR Layer

**Responses Handler** (`internal/responses/handler.go:102-120`):

```go
// Saves directly to ResponseStore, NOT through IR layer
if respStore, ok := h.store.(storage.ResponseStore); ok {
    record := &storage.ResponseRecord{
        ID:                 responseID,  // resp_xxx
        // ...
    }
    respStore.SaveResponse(r.Context(), record)
}
```

**Result**:

- No `Interaction` record created
- No transformation steps captured
- No pipeline events logged
- Control plane must merge two separate data sources

### Problem 3: Inconsistent Event Logging

| Frontdoor | Events Logged |
|-----------|---------------|
| Anthropic | ✅ frontdoor_decode, provider_encode, provider_decode, frontdoor_encode |
| OpenAI | ❌ None |
| Responses | ❌ None |

### Problem 4: Fragmented Storage

```
┌─────────────────────────────────────────────────────────────────┐
│                     CURRENT STATE                                │
├─────────────────────────────────────────────────────────────────┤
│  ConversationStore     ResponseStore      InteractionStore      │
│  ┌─────────────┐      ┌─────────────┐    ┌─────────────────┐   │
│  │conversations│      │  responses  │    │  interactions   │   │
│  │  (legacy)   │      │(Responses   │    │  (new unified)  │   │
│  │             │      │  API only)  │    │                 │   │
│  └─────────────┘      └─────────────┘    └─────────────────┘   │
│        ↑                    ↑                    ↑              │
│   OpenAI/Anthropic    Responses API      Should be used        │
│   frontdoors (legacy)  (direct save)     by ALL frontdoors     │
└─────────────────────────────────────────────────────────────────┘
```

---

## Target State Architecture

### Unified Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TARGET STATE                                       │
│                                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                     │
│  │   OpenAI     │   │  Anthropic   │   │  Responses   │                     │
│  │  Frontdoor   │   │  Frontdoor   │   │   Handler    │                     │
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────┘                     │
│         │                  │                  │                              │
│         └──────────────────┼──────────────────┘                              │
│                            ▼                                                 │
│                  ┌─────────────────────┐                                     │
│                  │   IR Layer (unified) │                                     │
│                  │  StartInteraction()  │                                     │
│                  │  LogEvent()          │                                     │
│                  │  CompleteInteraction()│                                    │
│                  └──────────┬───────────┘                                    │
│                            │                                                 │
│                            ▼                                                 │
│                  ┌─────────────────────┐                                     │
│                  │  InteractionStore   │                                     │
│                  │  ┌───────────────┐  │                                     │
│                  │  │ interactions  │  │  ← Single table for ALL            │
│                  │  └───────────────┘  │                                     │
│                  │  ┌───────────────┐  │                                     │
│                  │  │interaction_   │  │  ← Pipeline events                  │
│                  │  │events         │  │                                     │
│                  │  └───────────────┘  │                                     │
│                  │  ┌───────────────┐  │                                     │
│                  │  │shadow_results │  │  ← Links via InteractionID         │
│                  │  └───────────────┘  │                                     │
│                  └─────────────────────┘                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### ID Generation Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                    ID HIERARCHY                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Gateway-Owned (Primary Keys):                                   │
│  ├─ InteractionID: "int_<uuid>"    ← ALL interactions           │
│  ├─ EventID:       "evt_<uuid>"    ← Pipeline events            │
│  └─ ShadowID:      "shd_<uuid>"    ← Shadow results             │
│                                                                  │
│  Provider IDs (Metadata Only):                                   │
│  ├─ OpenAI:        "chatcmpl-xxx"  → stored in response.provider_id │
│  ├─ Anthropic:     "msg_xxx"       → stored in response.provider_id │
│  └─ Responses API: (we generate)   → N/A                        │
│                                                                  │
│  Client-Visible IDs:                                             │
│  ├─ OpenAI frontdoor:    Returns provider's ID (unchanged)      │
│  ├─ Anthropic frontdoor: Returns provider's ID (unchanged)      │
│  └─ Responses frontdoor: Returns "resp_<uuid>" (our ID)         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Fix ID Generation (Non-Breaking) ✅ COMPLETED

#### Step 1.1: Update `RecordInteraction` to never override with provider ID ✅

**File**: `internal/conversation/interaction_recorder.go`

**Changes Made**:

- `RecordInteraction` now always uses `int_<uuid>` as the primary key
- Provider's response ID is stored in `interaction.Response.ProviderResponseID` and `interaction.Metadata["provider_response_id"]`
- Frontdoor-assigned interaction IDs (via `CanonicalReq.Metadata["interaction_id"]`) are respected

#### Step 1.2: Add `ProviderResponseID` field to `InteractionResponse` ✅

**File**: `internal/core/domain/interaction.go`

Added `ProviderResponseID` to `InteractionResponse` struct.

**File**: `internal/storage/sqldb/store.go`

Added `response_provider_id` column to interactions table schema.

### Phase 2: Route Responses API Through IR Layer ✅ COMPLETED

#### Step 2.1: Update Responses handler to use IR ✅

**File**: `internal/responses/handler.go`

**Changes Made**:

- Both `handleNonStreamingResponse` and `handleStreamingResponse` now use `conversation.RecordInteraction()`
- Response ID (`resp_<uuid>`) is stored as `ProviderResponseID` in the interaction for client compatibility
- Interaction ID (`int_<uuid>`) is the primary key in storage

#### Step 2.2: Store Responses API-specific data in Interaction ✅

**File**: `internal/core/domain/interaction.go`

Added to `Interaction` struct:

- `PreviousInteractionID string` - links to previous interaction in thread
- `ThreadKey string` - identifies the thread (first interaction ID)

**File**: `internal/conversation/interaction_recorder.go`

Updated `RecordInteractionParams` to accept threading fields:

- `PreviousInteractionID string`
- `ThreadKey string`

**File**: `internal/storage/sqldb/store.go`

Added to interactions table:

- `previous_interaction_id TEXT`
- `thread_key TEXT`
- Indices: `idx_interactions_thread_key`, `idx_interactions_previous`

**File**: `internal/core/ports/storage.go`

Added to `InteractionStore` interface:

- `GetInteractionByProviderResponseID(ctx, providerResponseID) (*Interaction, error)`

**File**: `internal/storage/sqldb/interaction_store.go`

Implemented `GetInteractionByProviderResponseID` to lookup interactions by the client-facing response ID.

**File**: `internal/storage/memory/store.go`

Implemented `GetInteractionByProviderResponseID` for in-memory storage.

### Phase 3: Event Logging Consistency ✅ COMPLETED

#### Step 3.1: Add event logging to OpenAI frontdoor ✅

**File**: `internal/openai/frontdoor.go`

Added the same event logging pattern as Anthropic frontdoor:

- `frontdoor_decode` - after decoding the incoming request
- `provider_encode` - before sending to provider (if available)
- `provider_decode` - after receiving provider response
- `frontdoor_encode` - before returning response to client

#### Step 3.2: Add event logging to Responses handler ✅

**File**: `internal/responses/handler.go`

Added event logging for both streaming and non-streaming responses:

- `frontdoor_decode` - after decoding the Responses API request
- `provider_encode`, `provider_decode`, `frontdoor_encode` - for response pipeline

### Phase 4: Control Plane Unification ✅ COMPLETED

#### Step 4.1: Simplify `handleListInteractions` ✅

**File**: `internal/api/controlplane/server.go`

Removed legacy fallback code that merged conversations and responses. Now only uses `InteractionStore`:

- Single data source for all interaction types
- No more merge logic or sorting of heterogeneous data
- Cleaner, simpler implementation

#### Step 4.2: Simplify `handleInteractionDetail` ✅

**File**: `internal/api/controlplane/server.go`

Removed legacy fallback code that tried conversations and responses. Now only uses `InteractionStore`:

- Single lookup path for interaction details
- Returns 404 if interaction not found in unified store

**Note**: Legacy endpoints `/api/threads` and `/api/responses` remain for backward compatibility but could be deprecated in future.

### Phase 5: Shadow Pipeline Verification ✅ COMPLETED

Verified that the shadow pipeline correctly uses gateway-owned interaction IDs:

- `shadow.TriggerGlobalShadow()` receives the interaction ID from `RecordInteraction()`
- `Executor.Execute()` passes interaction ID to shadow provider execution
- `SaveShadowResult()` stores with `interaction_id` column
- `GetShadowResults()` retrieves by interaction ID
- Control plane correctly fetches shadow results by interaction ID

---

## Data Model Changes

### Updated `Interaction` struct

```go
type Interaction struct {
    // Core identification (gateway-owned)
    ID       string `json:"id"`        // "int_<uuid>" - always gateway-generated
    TenantID string `json:"tenant_id"`
    
    // API classification
    Frontdoor APIType `json:"frontdoor"` // openai, anthropic, responses
    Provider  string  `json:"provider"`
    AppName   string  `json:"app_name,omitempty"`
    
    // Model info
    RequestedModel string `json:"requested_model"`
    ServedModel    string `json:"served_model,omitempty"`
    ProviderModel  string `json:"provider_model,omitempty"`
    
    // Request/Response (with provider ID tracking)
    Request  *InteractionRequest  `json:"request"`
    Response *InteractionResponse `json:"response,omitempty"`
    Error    *InteractionError    `json:"error,omitempty"`
    
    // Responses API threading (replaces ResponseRecord.PreviousResponseID)
    PreviousInteractionID string `json:"previous_interaction_id,omitempty"`
    ThreadKey            string `json:"thread_key,omitempty"`
    
    // ... rest unchanged
}

type InteractionResponse struct {
    Raw              json.RawMessage `json:"raw,omitempty"`
    CanonicalJSON    json.RawMessage `json:"canonical,omitempty"`
    ProviderRequest  json.RawMessage `json:"provider_request,omitempty"` // renamed from Request.ProviderRequest
    ClientResponse   json.RawMessage `json:"client_response,omitempty"`
    ProviderResponseID string        `json:"provider_response_id,omitempty"` // NEW: e.g., "chatcmpl-xxx"
    UnmappedFields   []string        `json:"unmapped_fields,omitempty"`
    FinishReason     string          `json:"finish_reason,omitempty"`
    Usage            *Usage          `json:"usage,omitempty"`
}
```

### Database Schema Changes

```sql
-- Add columns to interactions table
ALTER TABLE interactions ADD COLUMN previous_interaction_id TEXT;
ALTER TABLE interactions ADD COLUMN thread_key TEXT;
ALTER TABLE interactions ADD COLUMN response_provider_id TEXT;

-- Index for thread lookups
CREATE INDEX idx_interactions_thread_key ON interactions(thread_key);
CREATE INDEX idx_interactions_previous ON interactions(previous_interaction_id);
```

---

## Migration Strategy

### Phase 1: Parallel Operation

1. New interactions stored in `interactions` table
2. Legacy `ResponseStore` calls transparently write to both tables
3. Control plane reads from `interactions` first, falls back to legacy

### Phase 2: Backfill

1. Migrate existing `responses` → `interactions`
2. Migrate existing `conversations` → `interactions` (if any remain)

### Phase 3: Deprecation (Future Work)

1. Remove writes to `responses` table
2. Remove `ConversationStore`/`ResponseStore` interfaces
3. Remove legacy `/api/threads` and `/api/responses` endpoints

---

## Testing Strategy

### Unit Tests

- [x] `RecordInteraction` always uses gateway ID
- [x] Provider ID stored in metadata, not as primary key
- [x] Responses API flows through IR layer
- [x] Event logging consistent across all frontdoors

### Integration Tests

- [x] Full request lifecycle through each frontdoor
- [x] Shadow execution links correctly to interactions
- [x] Thread continuation works with new ID scheme
- [x] Control plane displays all interaction types

### Backward Compatibility

- [x] Existing `resp_` IDs still retrievable (via `GetInteractionByProviderResponseID`)
- [x] Legacy API endpoints still work (`/api/threads`, `/api/responses`)
- [x] No data loss - all data stored in unified `interactions` table

---

## Phase 6: UI Adaptation ✅ COMPLETED

### Overview

The frontend has been updated to work with the unified interactions model. Instead of filtering by legacy types (`conversation`, `response`, `interaction`), the UI now filters by **frontdoor type** (`openai`, `anthropic`, `responses`).

### Changes Made

#### Step 6.1: Update Types (`web/control-plane/src/types/index.ts`) ✅

- Marked legacy types as `DEPRECATED`: `ResponseSummary`, `ResponseDetail`, `ThreadSummary`, `ThreadDetail`
- Updated `InteractionSummary.type` from union `'conversation' | 'response' | 'interaction'` to always `'interaction'`
- Updated `NewInteractionDetail.type` from optional to required `'interaction'`
- Simplified `InteractionDetailUnion` to just `NewInteractionDetail` (no more legacy union)

#### Step 6.2: Update API Hooks (`web/control-plane/src/hooks/useApi.tsx`) ✅

- Changed filter type from `'conversation' | 'response' | 'interaction' | ''` to `'openai' | 'anthropic' | 'responses' | ''`
- Updated `refreshInteractions` to use `?frontdoor=` parameter instead of `?type=`
- Updated `fetchInteractionDetail` return type to `NewInteractionDetail`

#### Step 6.3: Update Data Explorer (`web/control-plane/src/pages/Data.tsx`) ✅

- Replaced filter buttons: `Chats` → `OpenAI`, `Responses (legacy)` → `Anthropic`, `Interactions` → `Responses`
- Updated stats bar to show counts by frontdoor type instead of legacy types
- Simplified interaction list rendering to use frontdoor-based icons and colors
- Removed legacy conversation and response detail views - now only uses `UnifiedInteractionDetail`
- Cleaned up type casts and removed dead code paths

### UI Before vs After

**Before (legacy)**:

- Filter tabs: All | Chats | Responses | Interactions
- Stats: X conversations, Y responses, Z interactions
- List icons: MessageSquare (chat), Bot (response), ArrowLeftRight (interaction)

**After (unified)**:

- Filter tabs: All | OpenAI | Anthropic | Responses
- Stats: X OpenAI, Y Anthropic, Z Responses
- List icons: Terminal (OpenAI), MessageSquare (Anthropic), Bot (Responses)

### Test Results

- ✅ ESLint passes
- ✅ All 137 Vitest tests pass
- ✅ All 27 Go test packages pass

---

## Success Criteria ✅ ALL MET

1. ✅ **Single ID namespace**: All interactions use `int_<uuid>` as primary key
2. ✅ **Provider IDs preserved**: Stored in `response.provider_response_id`
3. ✅ **Unified storage**: Only `interactions` table used for new data
4. ✅ **Consistent events**: All frontdoors emit same pipeline events
5. ✅ **Shadow linking works**: Shadow results correctly reference interactions
6. ✅ **Control plane simplified**: No more merge logic, single data source
7. ✅ **UI unified**: Frontend uses unified interaction model with frontdoor-based filtering

---

## Rollback Strategy

Each phase is independently revertible:

- Phase 1: Revert ID generation logic
- Phase 2: Re-enable direct ResponseStore writes
- Phase 3: Keep legacy interfaces
- Phase 4: Remove event logging calls
- Phase 5: Restore control plane merge logic
- Phase 6: Restore legacy filter types in UI

Git tags will be created at each phase boundary for easy rollback.
