package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/auth"
)

// RequestIDKey is the context key for request IDs
type contextKey string

const RequestIDKey contextKey = "request_id"

// logFieldsKey identifies request-scoped logging fields.
type logFieldsKey struct{}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware logs HTTP requests with structured logging
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Attach mutable log fields map to context for handlers to enrich
			fields := make(map[string]string)
			ctxWithFields := context.WithValue(r.Context(), logFieldsKey{}, fields)

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Get request ID from context
			requestID, _ := r.Context().Value(RequestIDKey).(string)

			// Log request start
			logger.Info("request started",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			next.ServeHTTP(wrapped, r.WithContext(ctxWithFields))

			// Log request completion
			duration := time.Since(start)
			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
			}

			if len(fields) > 0 {
				for k, v := range fields {
					attrs = append(attrs, slog.String(k, v))
				}
			}

			logger.LogAttrs(ctxWithFields, slog.LevelInfo, "request completed", attrs...)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush forwards Flush to the underlying ResponseWriter if it supports http.Flusher,
// preserving streaming support (e.g., for SSE).
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// TimeoutMiddleware enforces request timeouts
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AddLogField attaches a key/value to the request-scoped log fields map so LoggingMiddleware can emit it.
// It is safe to call multiple times. No-op if middleware isn't present.
func AddLogField(ctx context.Context, key, value string) {
	if value == "" {
		return
	}
	if fields, ok := ctx.Value(logFieldsKey{}).(map[string]string); ok {
		fields[key] = value
	}
}

// AddError attaches an error message to the request-scoped log fields map so it
// appears in the structured request log emitted by LoggingMiddleware. No-op if
// middleware isn't present or err is nil.
func AddError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	AddLogField(ctx, "error", err.Error())
}

// AuthMiddleware validates API keys and injects tenant context
func AuthMiddleware(authenticator *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from Authorization header
			apiKey := r.Header.Get("Authorization")
			if apiKey == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Remove "Bearer " prefix if present
			if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:]
			}

			// Validate API key and get tenant
			tenant, err := authenticator.ValidateAPIKey(apiKey)
			if err != nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Inject tenant into context
			ctx := context.WithValue(r.Context(), "tenant", tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// rateLimitContextKey is the context key for rate limit info
type rateLimitContextKey struct{}

// SetRateLimits stores rate limit info in context for the middleware to write as headers.
func SetRateLimits(ctx context.Context, rl *RateLimitInfo) context.Context {
	return context.WithValue(ctx, rateLimitContextKey{}, rl)
}

// RateLimitInfo contains normalized rate limit information.
// This is a copy of domain.RateLimitInfo to avoid import cycles.
type RateLimitInfo struct {
	RequestsLimit     int
	RequestsRemaining int
	RequestsReset     string
	TokensLimit       int
	TokensRemaining   int
	TokensReset       string
}

// RateLimitNormalizingMiddleware writes normalized rate limit headers to responses.
// It reads rate limit info from context (set by frontdoors after provider calls)
// and writes standardized x-ratelimit-* headers.
func RateLimitNormalizingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a wrapper that will write headers after the handler sets rate limits
		wrapped := &rateLimitResponseWriter{
			ResponseWriter: w,
			request:        r,
		}
		next.ServeHTTP(wrapped, r)
	})
}

// rateLimitResponseWriter wraps ResponseWriter to write rate limit headers.
type rateLimitResponseWriter struct {
	http.ResponseWriter
	request       *http.Request
	wroteHeaders  bool
}

func (rw *rateLimitResponseWriter) WriteHeader(code int) {
	if !rw.wroteHeaders {
		rw.writeRateLimitHeaders()
		rw.wroteHeaders = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *rateLimitResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeaders {
		rw.writeRateLimitHeaders()
		rw.wroteHeaders = true
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *rateLimitResponseWriter) writeRateLimitHeaders() {
	rl, ok := rw.request.Context().Value(rateLimitContextKey{}).(*RateLimitInfo)
	if !ok || rl == nil {
		return
	}

	h := rw.Header()

	// Write normalized rate limit headers
	// Standard format: x-ratelimit-{limit|remaining|reset}-{requests|tokens}
	if rl.RequestsLimit > 0 {
		h.Set("x-ratelimit-limit-requests", itoa(rl.RequestsLimit))
	}
	if rl.RequestsRemaining > 0 {
		h.Set("x-ratelimit-remaining-requests", itoa(rl.RequestsRemaining))
	}
	if rl.RequestsReset != "" {
		h.Set("x-ratelimit-reset-requests", rl.RequestsReset)
	}

	if rl.TokensLimit > 0 {
		h.Set("x-ratelimit-limit-tokens", itoa(rl.TokensLimit))
	}
	if rl.TokensRemaining > 0 {
		h.Set("x-ratelimit-remaining-tokens", itoa(rl.TokensRemaining))
	}
	if rl.TokensReset != "" {
		h.Set("x-ratelimit-reset-tokens", rl.TokensReset)
	}
}

// itoa converts int to string without importing strconv
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	
	negative := i < 0
	if negative {
		i = -i
	}
	
	var buf [20]byte
	pos := len(buf)
	
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	
	if negative {
		pos--
		buf[pos] = '-'
	}
	
	return string(buf[pos:])
}

// Flush forwards Flush to the underlying ResponseWriter if it supports http.Flusher.
func (rw *rateLimitResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
