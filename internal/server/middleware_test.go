package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupRateLimitsMiddleware creates a middleware that sets rate limits in context
// before the rate limit normalizing middleware runs
func setupRateLimitsMiddleware(info *RateLimitInfo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := SetRateLimits(r.Context(), info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestRateLimitNormalizingMiddleware(t *testing.T) {
	// Final handler that just writes a response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Set up rate limits before the normalizing middleware
	info := &RateLimitInfo{
		RequestsLimit:     100,
		RequestsRemaining: 95,
		RequestsReset:     "2024-01-01T00:00:00Z",
		TokensLimit:       100000,
		TokensRemaining:   99000,
		TokensReset:       "2024-01-01T00:01:00Z",
	}

	// Chain: setupMiddleware -> RateLimitNormalizingMiddleware -> handler
	wrapped := setupRateLimitsMiddleware(info)(RateLimitNormalizingMiddleware(handler))

	// Make request
	req := httptest.NewRequest("POST", "/v1/messages", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check headers are set
	checkHeader(t, rec, "x-ratelimit-limit-requests", "100")
	checkHeader(t, rec, "x-ratelimit-remaining-requests", "95")
	checkHeader(t, rec, "x-ratelimit-reset-requests", "2024-01-01T00:00:00Z")
	checkHeader(t, rec, "x-ratelimit-limit-tokens", "100000")
	checkHeader(t, rec, "x-ratelimit-remaining-tokens", "99000")
	checkHeader(t, rec, "x-ratelimit-reset-tokens", "2024-01-01T00:01:00Z")
}

func TestRateLimitNormalizingMiddleware_NoRateLimits(t *testing.T) {
	// Handler that doesn't set rate limits
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RateLimitNormalizingMiddleware(handler)

	req := httptest.NewRequest("POST", "/v1/messages", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// No rate limit headers should be set
	if rec.Header().Get("x-ratelimit-limit-requests") != "" {
		t.Error("Expected no rate limit headers when not set in context")
	}
}

func TestRateLimitNormalizingMiddleware_PartialRateLimits(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Only set some rate limits
	info := &RateLimitInfo{
		RequestsLimit:     100,
		RequestsRemaining: 95,
		// Other fields are zero/empty
	}

	wrapped := setupRateLimitsMiddleware(info)(RateLimitNormalizingMiddleware(handler))

	req := httptest.NewRequest("POST", "/v1/messages", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Only non-zero/non-empty headers should be set
	checkHeader(t, rec, "x-ratelimit-limit-requests", "100")
	checkHeader(t, rec, "x-ratelimit-remaining-requests", "95")

	// Zero values should not produce headers
	if rec.Header().Get("x-ratelimit-limit-tokens") != "" {
		t.Error("Expected no tokens-limit header when zero")
	}
}

func TestSetRateLimits_Integration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	info := &RateLimitInfo{
		RequestsLimit:     100,
		RequestsRemaining: 95,
	}

	wrapped := setupRateLimitsMiddleware(info)(RateLimitNormalizingMiddleware(handler))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Verify the middleware picked up the rate limits
	checkHeader(t, rec, "x-ratelimit-limit-requests", "100")
	checkHeader(t, rec, "x-ratelimit-remaining-requests", "95")
}

func checkHeader(t *testing.T, rec *httptest.ResponseRecorder, name, expected string) {
	t.Helper()
	actual := rec.Header().Get(name)
	if actual != expected {
		t.Errorf("Header %s = %q, want %q", name, actual, expected)
	}
}
