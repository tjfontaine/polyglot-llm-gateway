package server

import (
	"context"
	"net/http"
)

// rateLimitContextKey is the context key for rate limit info
type rateLimitContextKey struct{}

// RateLimitInfo contains normalized rate limit information.
// This struct is used to pass rate limit information from handlers to middleware
// for inclusion in response headers.
type RateLimitInfo struct {
	RequestsLimit     int
	RequestsRemaining int
	RequestsReset     string
	TokensLimit       int
	TokensRemaining   int
	TokensReset       string
}

// SetRateLimits stores rate limit info in context for the middleware to write as headers.
func SetRateLimits(ctx context.Context, rl *RateLimitInfo) context.Context {
	return context.WithValue(ctx, rateLimitContextKey{}, rl)
}

// GetRateLimits retrieves rate limit info from context.
// Returns nil if no rate limits are set.
func GetRateLimits(ctx context.Context) *RateLimitInfo {
	if rl, ok := ctx.Value(rateLimitContextKey{}).(*RateLimitInfo); ok {
		return rl
	}
	return nil
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
	request      *http.Request
	wroteHeaders bool
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
	if rl.RequestsLimit > 0 || rl.RequestsRemaining > 0 {
		// Only set remaining if we have limit info (0 is a valid remaining value)
		h.Set("x-ratelimit-remaining-requests", itoa(rl.RequestsRemaining))
	}
	if rl.RequestsReset != "" {
		h.Set("x-ratelimit-reset-requests", rl.RequestsReset)
	}

	if rl.TokensLimit > 0 {
		h.Set("x-ratelimit-limit-tokens", itoa(rl.TokensLimit))
	}
	if rl.TokensLimit > 0 || rl.TokensRemaining > 0 {
		// Only set remaining if we have limit info (0 is a valid remaining value)
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
