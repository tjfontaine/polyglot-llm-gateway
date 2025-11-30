package provider_test

import (
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
)

func TestListProviderTypes(t *testing.T) {
	types := provider.ListProviderTypes()
	if len(types) < 3 {
		t.Errorf("expected at least 3 provider types, got %d", len(types))
	}

	// Check for expected types
	expected := []string{"anthropic", "openai", "openai-compatible"}
	typeSet := make(map[string]bool)
	for _, tp := range types {
		typeSet[tp] = true
	}

	for _, exp := range expected {
		if !typeSet[exp] {
			t.Errorf("expected provider type %q to be registered", exp)
		}
	}
}

func TestListFactories(t *testing.T) {
	factories := provider.ListFactories()
	if len(factories) < 3 {
		t.Errorf("expected at least 3 factories, got %d", len(factories))
	}

	// Verify factories have required fields
	for _, f := range factories {
		if f.Type == "" {
			t.Error("factory has empty Type")
		}
		if f.Create == nil {
			t.Errorf("factory %q has nil Create function", f.Type)
		}
		if f.Description == "" {
			t.Errorf("factory %q has empty Description", f.Type)
		}
	}
}

func TestGetFactory(t *testing.T) {
	tests := []struct {
		providerType string
		wantOk       bool
	}{
		{"openai", true},
		{"openai-compatible", true},
		{"anthropic", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			f, ok := provider.GetFactory(tt.providerType)
			if ok != tt.wantOk {
				t.Errorf("GetFactory(%q) returned ok=%v, want %v", tt.providerType, ok, tt.wantOk)
			}
			if tt.wantOk {
				if f.Type != tt.providerType {
					t.Errorf("GetFactory(%q) returned factory with Type=%q", tt.providerType, f.Type)
				}
			}
		})
	}
}

func TestIsRegistered(t *testing.T) {
	tests := []struct {
		providerType string
		want         bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"gemini", false},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			got := provider.IsRegistered(tt.providerType)
			if got != tt.want {
				t.Errorf("IsRegistered(%q) = %v, want %v", tt.providerType, got, tt.want)
			}
		})
	}
}

func TestValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid openai config",
			cfg: config.ProviderConfig{
				Type:   "openai",
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "valid anthropic config",
			cfg: config.ProviderConfig{
				Type:   "anthropic",
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "unknown provider type",
			cfg: config.ProviderConfig{
				Type: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateProviderConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFactoryAPITypes(t *testing.T) {
	tests := []struct {
		providerType string
		wantAPIType  domain.APIType
	}{
		{"openai", domain.APITypeOpenAI},
		{"openai-compatible", domain.APITypeOpenAI},
		{"anthropic", domain.APITypeAnthropic},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			f, ok := provider.GetFactory(tt.providerType)
			if !ok {
				t.Fatalf("factory %q not found", tt.providerType)
			}
			if f.APIType != tt.wantAPIType {
				t.Errorf("factory %q has APIType=%v, want %v", tt.providerType, f.APIType, tt.wantAPIType)
			}
		})
	}
}
