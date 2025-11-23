package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/tjfontaine/poly-llm-gateway/internal/auth"
)

// RequestIDKey is the context key for request IDs
type contextKey string

const RequestIDKey contextKey = "request_id"

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

			next.ServeHTTP(wrapped, r)

			// Log request completion
			duration := time.Since(start)
			logger.Info("request completed",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
			)
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
