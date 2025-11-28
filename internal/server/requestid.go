package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDKey is the context key for request IDs
type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestIDMiddleware adds a unique request ID to each request.
// The request ID is stored in the context and set as the X-Request-ID response header.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context.
// Returns an empty string if no request ID is set.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
