/*
Package middleware provides HTTP middleware components for the polyglot LLM gateway.

# Overview

The middleware package contains HTTP middleware components that can be chained
together to form a request processing pipeline.

# Middleware Components

The middleware is organized into separate files for better maintainability:

## Request ID (requestid.go)

RequestIDMiddleware generates a unique UUID for each request and adds it to:
  - The request context (accessible via GetRequestID)
  - The X-Request-ID response header

This enables request tracing across the gateway.

## Logging (logging.go)

LoggingMiddleware provides structured request logging using slog:
  - Logs request start (method, path, remote_addr)
  - Logs request completion (status, duration)
  - Supports custom log fields via AddLogField/AddError

## Authentication (authmiddleware.go)

AuthMiddleware validates API keys and injects tenant context:
  - Extracts API key from Authorization header (Bearer token format)
  - Validates key against registered tenants
  - Injects tenant into request context

## Timeout (timeout.go)

TimeoutMiddleware enforces request timeouts:
  - Creates context with deadline
  - Handlers should check context.Done() for cooperative cancellation

## Rate Limiting (ratelimit.go)

RateLimitNormalizingMiddleware normalizes rate limit headers:
  - Reads rate limit info from context (set by frontdoor handlers)
  - Writes standardized x-ratelimit-* headers to response

# Middleware Chain Order

The recommended middleware order is:
 1. RequestIDMiddleware (first, to generate request IDs)
 2. LoggingMiddleware (logs all requests)
 3. AuthMiddleware (validates API keys)
 4. TimeoutMiddleware (enforces timeouts)
 5. Recoverer (catches panics)
 6. OTel instrumentation (OpenTelemetry)

# Context Keys

The package defines several context keys:
  - RequestIDKey: string UUID for the request
  - TenantContextKey: tenant information from auth
  - rateLimitContextKey: rate limit info for headers

# Example Usage

	// Create server with middleware chain
	server := New(port, logger, authenticator)

	// Register routes
	server.Router.Post("/v1/messages", handler.HandleMessages)

	// Start server
	server.Start()
*/
package middleware
