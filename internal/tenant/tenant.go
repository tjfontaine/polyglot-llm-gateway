package tenant

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// Tenant represents a client organization using the gateway
type Tenant struct {
	ID        string
	Name      string
	APIKeys   []APIKey
	Providers map[string]domain.Provider
	Routing   config.RoutingConfig
}

// APIKey represents an API key for a tenant
type APIKey struct {
	KeyHash     string
	Description string
}

// contextKey is the type for tenant context keys
type contextKey string

const TenantContextKey contextKey = "tenant"
