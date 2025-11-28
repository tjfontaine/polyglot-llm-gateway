package server

import (
	"context"
	"net/http"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/auth"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/tenant"
)

// TenantContextKey is the context key for tenant information.
const TenantContextKey = "tenant"

// AuthMiddleware validates API keys and injects tenant context.
// If the authenticator is nil, the middleware is a no-op.
// The API key is extracted from the Authorization header (Bearer token format).
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
			t, err := authenticator.ValidateAPIKey(apiKey)
			if err != nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Inject tenant into context
			ctx := context.WithValue(r.Context(), TenantContextKey, t)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTenant retrieves the tenant from context.
// Returns nil if no tenant is set.
func GetTenant(ctx context.Context) *tenant.Tenant {
	if t, ok := ctx.Value(TenantContextKey).(*tenant.Tenant); ok {
		return t
	}
	return nil
}
