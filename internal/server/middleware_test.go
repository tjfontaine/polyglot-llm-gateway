package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/auth"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tenant"
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

// =============================================================================
// RateLimitNormalizingMiddleware Tests
// =============================================================================

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

func TestGetRateLimits(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected *RateLimitInfo
	}{
		{
			name: "returns rate limits when set",
			setup: func() context.Context {
				info := &RateLimitInfo{RequestsLimit: 100}
				return SetRateLimits(context.Background(), info)
			},
			expected: &RateLimitInfo{RequestsLimit: 100},
		},
		{
			name: "returns nil when not set",
			setup: func() context.Context {
				return context.Background()
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetRateLimits(ctx)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("expected %+v, got nil", tt.expected)
				} else if result.RequestsLimit != tt.expected.RequestsLimit {
					t.Errorf("RequestsLimit = %d, want %d", result.RequestsLimit, tt.expected.RequestsLimit)
				}
			}
		})
	}
}

// =============================================================================
// RequestIDMiddleware Tests
// =============================================================================

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Verify X-Request-ID header is set
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestRequestIDMiddleware_UniqueIDs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestIDMiddleware(handler)

	// Make two requests
	req1 := httptest.NewRequest("GET", "/", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest("GET", "/", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	id1 := rec1.Header().Get("X-Request-ID")
	id2 := rec2.Header().Get("X-Request-ID")

	if id1 == id2 {
		t.Errorf("Expected unique request IDs, got same: %s", id1)
	}
}

func TestGetRequestID_NotSet(t *testing.T) {
	ctx := context.Background()
	if id := GetRequestID(ctx); id != "" {
		t.Errorf("Expected empty string, got %q", id)
	}
}

// =============================================================================
// TimeoutMiddleware Tests
// =============================================================================

func TestTimeoutMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has deadline
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("Expected context to have deadline")
		}
		if deadline.IsZero() {
			t.Error("Expected non-zero deadline")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := TimeoutMiddleware(30 * time.Second)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestTimeoutMiddleware_ContextCancelled(t *testing.T) {
	// Create a handler that checks if context is cancelled
	contextCancelled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			contextCancelled = true
		case <-time.After(100 * time.Millisecond):
			// Context should be cancelled before this
		}
		w.WriteHeader(http.StatusOK)
	})

	// Very short timeout
	wrapped := TimeoutMiddleware(10 * time.Millisecond)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !contextCancelled {
		t.Error("Expected context to be cancelled due to timeout")
	}
}

// =============================================================================
// AuthMiddleware Tests
// =============================================================================

func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	// Create authenticator with a known API key
	hash := auth.HashAPIKey("valid-key-123")
	tenants := []*tenant.Tenant{
		{
			Name: "tenant1",
			APIKeys: []tenant.APIKey{
				{KeyHash: hash},
			},
		},
	}
	authenticator := auth.NewAuthenticator(tenants)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tenant is in context
		tenant := GetTenant(r.Context())
		if tenant == nil {
			t.Error("Expected tenant in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := AuthMiddleware(authenticator)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-key-123")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	hash := auth.HashAPIKey("valid-key-123")
	tenants := []*tenant.Tenant{
		{
			Name: "tenant1",
			APIKeys: []tenant.APIKey{
				{KeyHash: hash},
			},
		},
	}
	authenticator := auth.NewAuthenticator(tenants)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for invalid API key")
	})

	wrapped := AuthMiddleware(authenticator)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	hash := auth.HashAPIKey("valid-key-123")
	tenants := []*tenant.Tenant{
		{
			Name: "tenant1",
			APIKeys: []tenant.APIKey{
				{KeyHash: hash},
			},
		},
	}
	authenticator := auth.NewAuthenticator(tenants)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without auth header")
	})

	wrapped := AuthMiddleware(authenticator)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	// No Authorization header
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Missing Authorization") {
		t.Errorf("Expected 'Missing Authorization' in response, got %s", rec.Body.String())
	}
}

func TestAuthMiddleware_WithoutBearerPrefix(t *testing.T) {
	hash := auth.HashAPIKey("valid-key-123")
	tenants := []*tenant.Tenant{
		{
			Name: "tenant1",
			APIKeys: []tenant.APIKey{
				{KeyHash: hash},
			},
		},
	}
	authenticator := auth.NewAuthenticator(tenants)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := AuthMiddleware(authenticator)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "valid-key-123")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should still work (API key without Bearer prefix)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetTenant_NotSet(t *testing.T) {
	ctx := context.Background()
	if tenant := GetTenant(ctx); tenant != nil {
		t.Errorf("Expected nil, got %v", tenant)
	}
}

// =============================================================================
// LoggingMiddleware Tests
// =============================================================================

func TestLoggingMiddleware(t *testing.T) {
	var buf strings.Builder
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Chain RequestIDMiddleware -> LoggingMiddleware -> handler
	wrapped := RequestIDMiddleware(LoggingMiddleware(logger)(testHandler))

	req := httptest.NewRequest("GET", "/test-path", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	output := buf.String()

	// Verify both start and completion logs are present
	if !strings.Contains(output, "request started") {
		t.Error("Expected 'request started' in log output")
	}
	if !strings.Contains(output, "request completed") {
		t.Error("Expected 'request completed' in log output")
	}
	if !strings.Contains(output, "/test-path") {
		t.Error("Expected path in log output")
	}
}

func TestAddLogField(t *testing.T) {
	var buf strings.Builder
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddLogField(r.Context(), "custom_field", "custom_value")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := LoggingMiddleware(logger)(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "custom_field") || !strings.Contains(output, "custom_value") {
		t.Errorf("Expected custom field in log output, got: %s", output)
	}
}

func TestAddLogField_EmptyValue(t *testing.T) {
	var buf strings.Builder
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddLogField(r.Context(), "empty_field", "")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := LoggingMiddleware(logger)(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	output := buf.String()
	// Empty values should not be added
	if strings.Contains(output, "empty_field") {
		t.Errorf("Empty field should not be in log output, got: %s", output)
	}
}

func TestAddLogField_NoContext(t *testing.T) {
	// Should not panic when called with a context that doesn't have log fields
	ctx := context.Background()
	AddLogField(ctx, "key", "value") // Should be a no-op
}

func TestAddError(t *testing.T) {
	var buf strings.Builder
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddError(r.Context(), errors.New("test error message"))
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrapped := LoggingMiddleware(logger)(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "error") || !strings.Contains(output, "test error message") {
		t.Errorf("Expected error in log output, got: %s", output)
	}
}

func TestAddError_Nil(t *testing.T) {
	// Should not panic when called with nil error
	ctx := context.Background()
	AddError(ctx, nil) // Should be a no-op
}

// =============================================================================
// itoa Tests
// =============================================================================

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{12345, "12345"},
		{-1, "-1"},
		{-12345, "-12345"},
		{2147483647, "2147483647"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("itoa(%d)", tt.input), func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func checkHeader(t *testing.T, rec *httptest.ResponseRecorder, name, expected string) {
	t.Helper()
	actual := rec.Header().Get(name)
	if actual != expected {
		t.Errorf("Header %s = %q, want %q", name, actual, expected)
	}
}
