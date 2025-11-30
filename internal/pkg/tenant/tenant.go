package tenant

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Tenant represents a client organization using the gateway
type Tenant struct {
	ID        string
	Name      string
	APIKeys   []APIKey
	Providers map[string]ports.Provider
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
