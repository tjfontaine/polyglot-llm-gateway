package tenant

import (
	"fmt"

	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	"github.com/tjfontaine/poly-llm-gateway/internal/provider"
)

// Registry manages tenant instances
type Registry struct {
	tenants map[string]*Tenant
}

// NewRegistry creates a new tenant registry
func NewRegistry() *Registry {
	return &Registry{
		tenants: make(map[string]*Tenant),
	}
}

// LoadTenants loads tenants from configuration
func (r *Registry) LoadTenants(configs []config.TenantConfig, providerRegistry *provider.Registry) ([]*Tenant, error) {
	var tenants []*Tenant

	for _, cfg := range configs {
		// Create providers for this tenant
		providers, err := providerRegistry.CreateProviders(cfg.Providers)
		if err != nil {
			return nil, fmt.Errorf("failed to create providers for tenant %s: %w", cfg.ID, err)
		}

		// Convert API key configs
		apiKeys := make([]APIKey, len(cfg.APIKeys))
		for i, keyCfg := range cfg.APIKeys {
			apiKeys[i] = APIKey{
				KeyHash:     keyCfg.KeyHash,
				Description: keyCfg.Description,
			}
		}

		tenant := &Tenant{
			ID:        cfg.ID,
			Name:      cfg.Name,
			APIKeys:   apiKeys,
			Providers: providers,
			Routing:   cfg.Routing,
		}

		tenants = append(tenants, tenant)
		r.tenants[cfg.ID] = tenant
	}

	return tenants, nil
}

// GetTenant retrieves a tenant by ID
func (r *Registry) GetTenant(id string) (*Tenant, bool) {
	t, ok := r.tenants[id]
	return t, ok
}
