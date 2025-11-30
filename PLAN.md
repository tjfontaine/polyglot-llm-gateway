# **Polyglot LLM Gateway: Deep Dive & Refactoring Plan**

## **1\. Executive Summary**

The polyglot-llm-gateway implements a robust "Hourglass Architecture" using a Canonical Intermediate Representation (IR) to translate between OpenAI and Anthropic formats. It features advanced capabilities like model routing, response persistence (SQLite), and a control plane.

Status: Production-ready for single-tenant use.  
Gaps: Requires hardening for multi-tenant security (SSRF), high-concurrency optimization (lock contention), and structural reorganization to decouple protocol handlers from backend providers.

## **2\. Directory Restructuring Plan**

The proposed structure adheres to the **Standard Go Project Layout**, but emphasizes **Package-Oriented Design** for providers. Each provider (OpenAI, Anthropic, etc.) is a self-contained package within internal/backend/ containing its client, adapter logic, and internal types.

### **2.1 Proposed Tree Structure**

poly-llm-gateway/  
├── cmd/  
│   ├── gateway/              \# Main entrypoint (wires everything together)  
│   └── keygen/               \# Utility for API key hashing  
├── internal/  
│   ├── core/                 \# Pure domain logic (no external deps)  
│   │   ├── domain/           \# Canonical types (Request, Response, Event)  
│   │   ├── errors/           \# Canonical error types  
│   │   └── ports/            \# Interfaces (Provider, Store, TokenCounter)  
│   ├── api/                  \# INBOUND: HTTP Handlers (Frontdoors)  
│   │   ├── server/           \# Server setup & wiring  
│   │   ├── middleware/       \# Auth, RateLimit, OTel, Logging  
│   │   ├── openai/           \# Handler for /v1/chat/completions (decodes to Canonical)  
│   │   ├── anthropic/        \# Handler for /v1/messages (decodes to Canonical)  
│   │   ├── responses/        \# Handler for /v1/responses (Unified API)  
│   │   └── controlplane/     \# Control Plane API Handlers  
│   ├── backend/              \# OUTBOUND: Provider Implementations  
│   │   ├── openai/           \# Self-contained OpenAI implementation  
│   │   │   ├── client.go     \# HTTP Client logic  
│   │   │   ├── provider.go   \# Implements core.ports.Provider  
│   │   │   └── types.go      \# Internal OpenAI-specific types  
│   │   ├── anthropic/        \# Self-contained Anthropic implementation  
│   │   │   ├── client.go  
│   │   │   ├── provider.go  
│   │   │   └── types.go  
│   │   └── passthrough/      \# Wrapper for raw passthrough  
│   ├── router/               \# Routing & Policy Logic  
│   │   ├── dispatcher.go     \# Merged Router \+ ModelMapper logic  
│   │   └── policy.go         \# Rule evaluation  
│   ├── storage/              \# Persistence adapters  
│   │   ├── memory/  
│   │   └── sqlite/  
│   └── pkg/                  \# Shared stateless utilities  
│       ├── codec/            \# Protocol translation logic (OpenAI \<-\> Canonical)  
│       ├── tokens/           \# Token counting & estimation logic  
│       ├── safehttp/         \# SSRF-protected HTTP client  
│       ├── config/           \# Configuration loading  
│       ├── auth/             \# Authentication logic  
│       └── tenant/           \# Tenant management logic  
└── web/  
    └── control-plane/        \# React Dashboard

### **2.2 File Migration Map**

This map details exactly where each existing file should move.

| Current Path | New Path | Notes |
| :---- | :---- | :---- |
| cmd/gateway/main.go | cmd/gateway/main.go | Update imports |
| cmd/keygen/main.go | cmd/keygen/main.go | No changes |
| internal/domain/content.go | internal/core/domain/content.go | **DONE** |
| internal/domain/errors.go | internal/core/domain/errors.go | **DONE** (package remains domain) |
| internal/domain/errors\_test.go | internal/core/domain/errors\_test.go | **DONE** |
| internal/domain/interaction.go | internal/core/domain/interaction.go | **DONE** |
| internal/domain/interfaces.go | internal/core/ports/interfaces.go | **DONE** |
| internal/domain/responses.go | internal/core/domain/responses.go | **DONE** |
| internal/domain/tokens.go | internal/core/domain/tokens.go | **DONE** |
| internal/domain/types.go | internal/core/domain/types.go | **DONE** |
| internal/pkg/config/config.go | internal/pkg/config/config.go | **DONE** |
| internal/pkg/config/config\_test.go | internal/pkg/config/config\_test.go | **DONE** |
| internal/auth/auth.go | internal/pkg/auth/auth.go | **DONE** |
| internal/auth/auth\_test.go | internal/pkg/auth/auth\_test.go | **DONE** |
| internal/tenant/tenant.go | internal/pkg/tenant/tenant.go | **DONE** |
| internal/tenant/registry.go | internal/pkg/tenant/registry.go | **DONE** |
| internal/tenant/registry\_test.go | internal/pkg/tenant/registry\_test.go | **DONE** |
| internal/server/server.go | internal/api/server/server.go | **DONE** |
| internal/server/authmiddleware.go | internal/api/middleware/auth.go | **DONE** |
| internal/server/logging.go | internal/api/middleware/logging.go | **DONE** |
| internal/server/ratelimit.go | internal/api/middleware/ratelimit.go | **DONE** |
| internal/server/requestid.go | internal/api/middleware/requestid.go | **DONE** |
| internal/server/timeout.go | internal/api/middleware/timeout.go | **DONE** |
| internal/server/middleware\_test.go | internal/api/middleware/middleware\_test.go | **DONE** |
| internal/api/middleware/tracer/tracer.go | internal/api/middleware/tracer.go | **DONE** (kept in tracer/tracer.go with package tracer; wiring fixed) |
| internal/backend/openai/frontdoor.go | internal/api/openai/handler.go | Rename to handler.go |
| internal/backend/anthropic/frontdoor.go | internal/api/anthropic/handler.go | Rename to handler.go |
| internal/backend/anthropic/frontdoor\_test.go | internal/api/anthropic/handler\_test.go |  |
| internal/api/responses/handler.go | internal/api/responses/handler.go | **DONE** |
| internal/api/responses/handler\_test.go | internal/api/responses/handler\_test.go | **DONE** |
| internal/controlplane/server.go | internal/api/controlplane/server.go | **DONE** |
| internal/controlplane/dist/ | internal/api/controlplane/dist/ | **DONE** |
| internal/backend/openai/client.go | internal/backend/openai/client.go |  |
| internal/backend/openai/provider.go | internal/backend/openai/provider.go |  |
| internal/backend/openai/types.go | internal/backend/openai/types.go |  |
| internal/backend/openai/factory.go | internal/backend/openai/factory.go | **DONE** (explicit RegisterProviderFactories) |
| internal/backend/anthropic/client.go | internal/backend/anthropic/client.go |  |
| internal/backend/anthropic/provider.go | internal/backend/anthropic/provider.go |  |
| internal/backend/anthropic/types.go | internal/backend/anthropic/types.go |  |
| internal/backend/anthropic/factory.go | internal/backend/anthropic/factory.go | **DONE** (explicit RegisterProviderFactory) |
| internal/provider/passthrough.go | internal/backend/passthrough/provider.go | **DONE** |
| internal/provider/registry.go | internal/provider/registry.go | Keep registry; use explicit registration helpers (no init side effects) |
| internal/provider/registry/registry.go | internal/provider/registry/registry.go | Keep registry; discovery via explicit Register* |
| internal/frontdoor/registry.go | internal/frontdoor/registry.go | Keep registry; explicit frontdoor registration |
| internal/frontdoor/registry/registry.go | internal/frontdoor/registry/registry.go | Keep registry; discovery via explicit Register* |
| internal/policy/router.go | internal/router/router.go | **DONE** (policy router removed) |
| internal/provider/model\_mapping.go | internal/router/mapping.go | **DONE** (logic moved) |
| internal/storage/storage.go | internal/core/ports/storage.go | **DONE** (alias retained for compatibility) |
| internal/storage/memory/store.go | internal/storage/memory/store.go |  |
| internal/storage/sqlite/store.go | internal/storage/sqlite/store.go |  |
| internal/codec/codec.go | internal/pkg/codec/codec.go | **DONE** (moved) |
| internal/backend/openai/codec.go | internal/backend/openai/codec.go | **DONE** |
| internal/backend/anthropic/codec.go | internal/backend/anthropic/codec.go | **DONE** |
| internal/codec/images.go | internal/pkg/codec/images.go | **DONE** (moved; SafeTransport) |
| internal/codec/errors.go | internal/pkg/codec/errors.go | **DONE** (moved) |
| internal/pkg/tokens/registry.go | internal/pkg/tokens/registry.go | **DONE** |
| internal/pkg/tokens/openai.go | internal/pkg/tokens/openai.go | **DONE** |
| internal/pkg/tokens/anthropic.go | internal/pkg/tokens/anthropic.go | **DONE** |

### **2.3 Key Structural Principles**

1. **Backend Isolation:** internal/backend/\<provider\> contains *everything* needed to talk to that provider. It does not know about the HTTP server or frontdoor handlers. It only knows about internal/core/domain.  
   * **Benefit:** To add "Gemini", you create internal/backend/gemini/ and implement the Provider interface. No other code needs to change except cmd/gateway/main.go to wire it up.  
2. **API Layer Separation:** internal/api/\<protocol\> handles the *ingress*. It imports internal/pkg/codec to decode requests into Canonical format, then calls the backend.  
   * **Benefit:** You can expose an OpenAI-compatible endpoint that routes to an Anthropic backend without the two packages importing each other directly.  
3. **Shared Codec:** internal/pkg/codec holds the translation logic (e.g., "convert Canonical Request to OpenAI JSON"). This is used by both the **API** layer (to decode incoming requests) and the **Backend** layer (to encode outgoing requests for passthrough or specific formatting).

## **3\. Critical Findings & Action Items**

### **3.1 Security: SSRF in Image Fetcher (CRITICAL)**

Location: internal/pkg/codec/images.go (moved from internal/codec/images.go)  
Issue: The gateway fetches remote images for format conversion without validating the destination IP.  
**Fix:** Implement a SafeTransport that rejects private IP ranges.

// internal/pkg/safehttp/transport.go  
package safehttp

import (  
	"context"  
	"fmt"  
	"net"  
	"net/http"  
	"time"  
)

var SafeTransport \= \&http.Transport{  
	DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {  
		// Enforce timeout  
		dialer := \&net.Dialer{Timeout: 5 \* time.Second}  
		conn, err := dialer.DialContext(ctx, network, addr)  
		if err \!= nil {  
			return nil, err  
		}

		// Check IP against private ranges  
		host, \_, \_ := net.SplitHostPort(conn.RemoteAddr().String())  
		ip := net.ParseIP(host)  
		  
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {  
			conn.Close()  
			return nil, fmt.Errorf("access to private IP %s is denied", ip)  
		}  
		return conn, nil  
	},  
}

// Usage in internal/pkg/codec/images.go  
func NewImageFetcher() \*ImageFetcher {  
    return \&ImageFetcher{  
        client: \&http.Client{  
            Transport: safehttp.SafeTransport,  
            Timeout:   10 \* time.Second,  
        },  
    }  
}

### **3.2 Performance: SQLite Concurrency Bottleneck**

Location: internal/storage/sqlite/store.go  
Issue: Default SQLite mode blocks readers during writes.  
**Fix:** Enable Write-Ahead Logging (WAL) during initialization.

// internal/storage/sqlite/store.go  
func New(dbPath string) (\*Store, error) {  
	db, err := sql.Open("sqlite", dbPath)  
	if err \!= nil {  
		return nil, err  
	}  
	  
	// Enable WAL mode for high concurrency  
	if \_, err := db.Exec("PRAGMA journal\_mode=WAL; PRAGMA synchronous=NORMAL;"); err \!= nil {  
		db.Close()  
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)  
	}  
    // ...  
}

### **3.3 Architecture: Unify Routing Logic**

Current State: Double dispatch between policy.Router and ModelMappingProvider.  
Target State: A single Router component in internal/router that returns a RouteDecision.  
// internal/router/router.go  
type RouteDecision struct {  
    Provider      ports.Provider  
    UpstreamModel string  
    ShouldRewrite bool  
}

func (r \*Router) Decide(ctx context.Context, model string) (\*RouteDecision, error) {  
    // 1\. Check Rewrite Rules  
    // 2\. Check Prefix Rules  
    // 3\. Fallback  
    // Return explicit decision  
}

## **4\. Implementation Roadmap for Agent**

1. **Refactor Directory Structure (DONE):**  
   * Interfaces now at `internal/core/ports`; legacy empty dirs cleaned up.  
   * Imports updated; go tests pass.  
2. **Telemetry Wiring (DONE):**  
   * Tracer lives at `internal/api/middleware/tracer/tracer.go` (package `tracer`); main imports fixed.  
3. **Security Fix (DONE):**  
   * `internal/pkg/safehttp` in place; `internal/pkg/codec/images.go` uses `SafeTransport`.  
4. **Storage Optimization (DONE):**  
   * WAL pragma applied in `internal/storage/sqlite/store.go`.  
5. **Routing Consolidation (DONE):**  
   * Router logic merged into `internal/router`; legacy `internal/policy` removed.  
6. **Provider Wiring (MOSTLY DONE):**  
   * Factories registered explicitly; verify `registration.RegisterBuiltins()` wires all frontdoors/providers; add tests as needed.  
7. **Control Plane & Null Safety (PARTIAL):**  
   * Admin UI assets rebuilt and copied to `internal/api/controlplane/dist`.  
   * TODO: Add/verify tests covering null arrays for new components consuming API data.

## **5\. Verification Plan**

* **SSRF Test:** Attempt to send a request with an image URL pointing to http://localhost:8080/api/stats. Ensure it returns a 400/403 error.  
* **Concurrency Test:** Run benchmark against SQLite storage. Ensure no database is locked errors.  
* **Regression Test:** Run go test ./.... Ensure logic tests pass after the massive file movement.
