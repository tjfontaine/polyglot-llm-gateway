package provider_test

import (
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"

	// Import consolidated packages to trigger their init() registration for tests.
	_ "github.com/tjfontaine/polyglot-llm-gateway/internal/anthropic"
	_ "github.com/tjfontaine/polyglot-llm-gateway/internal/openai"
)

func TestRegistry_CreateProvider(t *testing.T) {
	registry := provider.NewRegistry()

	tests := []struct {
		name    string
		cfg     config.ProviderConfig
		wantErr bool
	}{
		{
			name: "openai",
			cfg: config.ProviderConfig{
				Type:   "openai",
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "openai-compatible",
			cfg: config.ProviderConfig{
				Type:    "openai-compatible",
				APIKey:  "test-key",
				BaseURL: "http://localhost:8080/v1",
			},
			wantErr: false,
		},
		{
			name: "anthropic",
			cfg: config.ProviderConfig{
				Type:   "anthropic",
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "unknown",
			cfg: config.ProviderConfig{
				Type: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := registry.CreateProvider(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
