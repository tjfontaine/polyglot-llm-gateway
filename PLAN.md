# Technical Debt Remediation Plan

## Overview

This plan addresses accumulated technical debt in the codebase, focusing on package consolidation, reducing indirection, and improving comprehensibility for contributors adding new API types.

## Goals

1. **Consolidated API Packages**: All code for an API type (OpenAI, Anthropic) lives in one package
2. **Reduced Indirection**: Flatten nested registry patterns, remove re-export wrappers
3. **Clear Naming**: Package names reflect their purpose without ambiguity
4. **Easy Extension**: Adding a new API type = copy one package, rename, implement

---

## Phase 1: Package Consolidation

### Current State (Scattered)

```
internal/
├── api/
│   ├── anthropic/handler.go        # Frontdoor handler
│   ├── openai/handler.go           # Frontdoor handler
│   ├── responses/handler.go        # Responses API handler
│   ├── middleware/                  # HTTP middleware
│   └── server/                      # Server setup
├── backend/
│   ├── anthropic/                   # Provider: types, client, codec, provider, factory
│   ├── openai/                      # Provider: types, client, codec, provider, factory
│   └── passthrough/                 # Passthrough provider
├── frontdoor/
│   ├── registry.go                  # Wrapper around registry/registry.go
│   ├── factory.go                   # Re-exports from registry/
│   └── registry/registry.go         # Actual implementation
├── provider/
│   ├── registry.go                  # Wrapper around registry/registry.go
│   ├── factory.go                   # Uses registry
│   └── registry/registry.go         # Actual implementation
```

### Target State (Consolidated)

```
internal/
├── anthropic/                       # ALL Anthropic code together
│   ├── types.go                     # API request/response types
│   ├── client.go                    # HTTP client
│   ├── codec.go                     # Canonical ↔ Anthropic translation
│   ├── provider.go                  # Provider implementation
│   ├── frontdoor.go                 # HTTP handler (was api/anthropic/handler.go)
│   └── registration.go              # Factory registration functions
├── openai/                          # ALL OpenAI code together
│   ├── types.go
│   ├── client.go
│   ├── codec.go
│   ├── provider.go
│   ├── frontdoor.go                 # HTTP handler (was api/openai/handler.go)
│   └── registration.go
├── responses/                       # Responses API (OpenAI-compatible)
│   ├── handler.go                   # Was api/responses/handler.go
│   └── types.go                     # Responses-specific types
├── passthrough/                     # Passthrough provider
│   └── provider.go
├── api/
│   ├── middleware/                  # HTTP middleware (unchanged)
│   ├── server/                      # Server setup (unchanged)
│   └── controlplane/                # Admin UI (unchanged)
├── frontdoor/
│   └── registry.go                  # Flattened: single file, no subpackage
├── provider/
│   └── registry.go                  # Flattened: single file, no subpackage
```

---

## Phase 2: Implementation Steps

### Step 1: Create `internal/anthropic/` consolidated package ✅

- [x] Move `internal/backend/anthropic/*.go` → `internal/anthropic/`
- [x] Move `internal/api/anthropic/handler.go` → `internal/anthropic/frontdoor.go`
- [x] Rename `FrontdoorHandler` to avoid collision, or keep in same package
- [x] Update package name from `anthropic` (already correct)
- [x] Create `registration.go` with both `RegisterProviderFactory()` and `RegisterFrontdoor()`
- [x] Update all imports throughout codebase

### Step 2: Create `internal/openai/` consolidated package ✅

- [x] Move `internal/backend/openai/*.go` → `internal/openai/`
- [x] Move `internal/api/openai/handler.go` → `internal/openai/frontdoor.go`
- [x] Create `registration.go` with both registration functions
- [x] Update all imports throughout codebase

### Step 3: Move `internal/api/responses/` → `internal/responses/` ✅

- [x] Move handler and types
- [x] Update imports

### Step 4: Move `internal/backend/passthrough/` → `internal/passthrough/` ✅

- [x] Move provider
- [x] Update imports

### Step 5: Flatten registry structure ✅

- [x] Merge `internal/frontdoor/registry/registry.go` into `internal/frontdoor/registry.go`
- [x] Remove `internal/frontdoor/registry/` subpackage
- [x] Remove `internal/frontdoor/factory.go` (re-exports no longer needed)
- [x] Merge `internal/provider/registry/registry.go` into `internal/provider/registry.go`
- [x] Remove `internal/provider/registry/` subpackage
- [x] Update all imports

### Step 6: Cleanup ✅

- [x] Remove empty `internal/backend/` directory
- [x] Remove empty `internal/api/anthropic/` directory
- [x] Remove empty `internal/api/openai/` directory
- [x] Remove empty `internal/tokens/` directory
- [x] Remove empty `internal/frontdoor/responses/` directory
- [x] Fix `internal/api/middleware/doc.go` package comment
- [x] Update `internal/registration/builtins.go` to use new package paths

---

## Phase 3: Additional Improvements

### Type Clarification

- [x] Rename `ports/storage.Message` → `StoredMessage` to differentiate from `domain.Message`
- [x] Rename `ports.Interaction` → `InteractionSummary` to differentiate from `domain.Interaction`

### Test Coverage

- [x] Add unit tests for `internal/anthropic/` package
- [x] Add unit tests for `internal/openai/` package
- [x] Add tests for `internal/router/mapping.go`

### Documentation

- [x] Update AGENTS.md to reflect completed structure
- [x] Update memory files (project_overview, style_and_conventions)

---

## Migration Notes

### Import Path Changes

| Old Path | New Path |
|----------|----------|
| `internal/backend/anthropic` | `internal/anthropic` |
| `internal/backend/openai` | `internal/openai` |
| `internal/api/anthropic` | `internal/anthropic` |
| `internal/api/openai` | `internal/openai` |
| `internal/api/responses` | `internal/responses` |
| `internal/backend/passthrough` | `internal/passthrough` |
| `internal/frontdoor/registry` | `internal/frontdoor` |
| `internal/provider/registry` | `internal/provider` |

### Breaking Changes

None expected - this is internal package reorganization only. No public API changes.

### Rollback Strategy

Git revert if needed. All changes are in-tree refactoring with no external dependencies.

---

## Success Criteria

1. `go build ./...` passes
2. `go test ./...` passes
3. All API functionality works unchanged
4. Adding a new API type requires creating only ONE new package
5. No more than 2 levels of package nesting for any API type code
