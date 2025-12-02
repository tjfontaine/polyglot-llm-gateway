// Package apikey provides API key-based authentication.
package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Provider implements ports.AuthProvider using API key authentication.
type Provider struct {
	configProvider ports.ConfigProvider
	mu             sync.RWMutex
	tenants        map[string]*ports.Tenant // tenantID -> tenant
	keyHashMap     map[string]string        // keyHash -> tenantID
}

// NewProvider creates a new API key auth provider.
func NewProvider(configProvider ports.ConfigProvider) (*Provider, error) {
	if configProvider == nil {
		return nil, fmt.Errorf("config provider required")
	}

	p := &Provider{
		configProvider: configProvider,
		tenants:        make(map[string]*ports.Tenant),
		keyHashMap:     make(map[string]string),
	}

	// Load initial tenants
	ctx := context.Background()
	cfg, err := configProvider.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := p.loadTenants(cfg); err != nil {
		return nil, fmt.Errorf("load tenants: %w", err)
	}

	return p, nil
}

// Authenticate validates an API key and returns the tenant context.
func (p *Provider) Authenticate(ctx context.Context, token string) (*ports.AuthContext, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Hash the token
	hash := sha256.Sum256([]byte(token))
	keyHash := hex.EncodeToString(hash[:])

	// Look up tenant by hash
	tenantID, ok := p.keyHashMap[keyHash]
	if !ok {
		return nil, fmt.Errorf("invalid API key")
	}

	tenant, ok := p.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant not found")
	}

	return &ports.AuthContext{
		TenantID: tenant.ID,
		Metadata: map[string]string{
			"tenant_name": tenant.Name,
		},
	}, nil
}

// GetTenant returns a tenant by ID.
func (p *Provider) GetTenant(ctx context.Context, tenantID string) (*ports.Tenant, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tenant, ok := p.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}

	return tenant, nil
}

// loadTenants loads tenants from configuration.
func (p *Provider) loadTenants(cfg *config.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear existing mappings
	p.tenants = make(map[string]*ports.Tenant)
	p.keyHashMap = make(map[string]string)

	// Load tenants from config
	for _, tenantCfg := range cfg.Tenants {
		// Convert config.TenantConfig to ports.Tenant
		t := &ports.Tenant{
			ID:      tenantCfg.ID,
			Name:    tenantCfg.Name,
			Routing: tenantCfg.Routing,
			// Providers will be populated later during gateway initialization
			Providers: make(map[string]ports.Provider),
		}

		p.tenants[t.ID] = t

		// Build key hash mappings
		for _, apiKey := range tenantCfg.APIKeys {
			p.keyHashMap[apiKey.KeyHash] = t.ID
		}
	}

	return nil
}

// ReloadFromConfig reloads tenants from new configuration.
// This is called by the gateway when config changes.
func (p *Provider) ReloadFromConfig(cfg *config.Config) error {
	return p.loadTenants(cfg)
}

// HashAPIKey creates a SHA-256 hash of an API key for storage.
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
