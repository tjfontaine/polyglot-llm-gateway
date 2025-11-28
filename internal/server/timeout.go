package server

import (
	"context"
	"net/http"
	"time"
)

// TimeoutMiddleware enforces request timeouts.
// If a request exceeds the specified timeout, the context is cancelled.
// Note: This does not forcibly terminate the handler, it relies on the handler
// checking context.Done() for cooperative cancellation.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
