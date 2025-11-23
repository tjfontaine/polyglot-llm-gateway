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
- **Control plane:** `web/control-plane` is a React 19 + Vite app whose built assets are copied into `internal/controlplane/dist` by the `Makefile`.

## Coding Conventions
- **Go style:** Keep code `gofmt`-clean and idiomatic Go. Prefer small, focused functions and return descriptive errors. Avoid introducing panics in request paths.
- **Configuration-driven behavior:** Respect the config structs in `internal/config` and the registry helpers when adding new frontdoors or providers; plug into `frontdoor.Registry`/`provider.Registry` instead of hardcoding wiring.
- **Canonical types first:** Work with `internal/domain` types at API boundaries. Frontdoors should fully populate `CanonicalRequest`/`CanonicalResponse` and let providers handle translation details.
- **Streaming:** When adding streaming support, emit `CanonicalEvent` values and propagate provider errors via the channel before closing it.
- **Tests & fixtures:** Integration-style provider tests rely on go-vcr cassettes under `testdata/fixtures`. Use `VCR_MODE=record` with real API keys to refresh cassettes; default `go test` replays without network access.
- **Logging/telemetry:** Use the structured `slog` logger already configured in `cmd/gateway/main.go` and preserve OpenTelemetry middleware hooks in `internal/server` when adding new routes.
- **Frontend:** Keep React components type-safe (TypeScript) and run linting before committing UI changes.

## Testing Expectations
- **Go:** Run `go test ./...` from the repo root. Provider tests will skip recording if the relevant API key is missing and `VCR_MODE=record` is set; in replay mode they work offline.
- **Frontend:** From `web/control-plane`, run `npm install` once, then `npm run lint` and `npm run build` for UI changes. The `Makefile` target `build` will bundle both frontend and backend if needed.

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
