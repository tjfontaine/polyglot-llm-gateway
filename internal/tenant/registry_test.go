package tenant

import (
	"context"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
)

// mockProviderRegistry for testing
type mockProviderRegistry struct{}

func (m *mockProviderRegistry) CreateProviders(configs []config.ProviderConfig) (map[string]domain.Provider, error) {
	providers := make(map[string]domain.Provider)
	for _, cfg := range configs {
		providers[cfg.Name] = &mockProvider{name: cfg.Name}
	}
	return providers, nil
}

// mockProvider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) APIType() domain.APIType {
	return domain.APITypeOpenAI
}

func (m *mockProvider) Complete(ctx context.Context, req *domain.CanonicalRequest) (*domain.CanonicalResponse, error) {
	return nil, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *domain.CanonicalRequest) (<-chan domain.CanonicalEvent, error) {
	return nil, nil
}

func (m *mockProvider) ListModels(context.Context) (*domain.ModelList, error) {
	return &domain.ModelList{Object: "list"}, nil
}

func TestRegistry_LoadTenants(t *testing.T) {
	registry := NewRegistry()
	providerReg := provider.NewRegistry()

	tenantConfigs := []config.TenantConfig{
		{
			ID:   "tenant-1",
			Name: "Test Tenant 1",
			APIKeys: []config.APIKeyConfig{
				{
					KeyHash:     "hash1",
					Description: "Key 1",
				},
			},
			Providers: []config.ProviderConfig{
				{
					Name:   "openai-1",
					Type:   "openai",
					APIKey: "test-key",
				},
			},
			Routing: config.RoutingConfig{
				DefaultProvider: "openai-1",
			},
		},
		{
			ID:   "tenant-2",
			Name: "Test Tenant 2",
			APIKeys: []config.APIKeyConfig{
				{
					KeyHash:     "hash2",
					Description: "Key 2",
				},
			},
			Providers: []config.ProviderConfig{
				{
					Name:   "anthropic-2",
					Type:   "anthropic",
					APIKey: "test-key-2",
				},
			},
			Routing: config.RoutingConfig{
				DefaultProvider: "anthropic-2",
			},
		},
	}

	tenants, err := registry.LoadTenants(tenantConfigs, providerReg)
	if err != nil {
		t.Fatalf("LoadTenants() error = %v", err)
	}

	if len(tenants) != 2 {
		t.Errorf("LoadTenants() returned %d tenants, want 2", len(tenants))
	}

	// Check first tenant
	if tenants[0].ID != "tenant-1" {
		t.Errorf("Tenant 0 ID = %v, want tenant-1", tenants[0].ID)
	}
	if tenants[0].Name != "Test Tenant 1" {
		t.Errorf("Tenant 0 Name = %v, want Test Tenant 1", tenants[0].Name)
	}
	if len(tenants[0].APIKeys) != 1 {
		t.Errorf("Tenant 0 has %d API keys, want 1", len(tenants[0].APIKeys))
	}
	if len(tenants[0].Providers) != 1 {
		t.Errorf("Tenant 0 has %d providers, want 1", len(tenants[0].Providers))
	}

	// Check second tenant
	if tenants[1].ID != "tenant-2" {
		t.Errorf("Tenant 1 ID = %v, want tenant-2", tenants[1].ID)
	}
}

func TestRegistry_GetTenant(t *testing.T) {
	registry := NewRegistry()
	providerReg := provider.NewRegistry()

	tenantConfigs := []config.TenantConfig{
		{
			ID:   "tenant-1",
			Name: "Test Tenant",
			Providers: []config.ProviderConfig{
				{
					Name:   "openai",
					Type:   "openai",
					APIKey: "key",
				},
			},
		},
	}

	_, err := registry.LoadTenants(tenantConfigs, providerReg)
	if err != nil {
		t.Fatalf("LoadTenants() error = %v", err)
	}

	t.Run("existing tenant", func(t *testing.T) {
		tenant, ok := registry.GetTenant("tenant-1")
		if !ok {
			t.Error("GetTenant() returned false for existing tenant")
		}
		if tenant.ID != "tenant-1" {
			t.Errorf("GetTenant() ID = %v, want tenant-1", tenant.ID)
		}
	})

	t.Run("non-existing tenant", func(t *testing.T) {
		_, ok := registry.GetTenant("non-existent")
		if ok {
			t.Error("GetTenant() returned true for non-existing tenant")
		}
	})
}
