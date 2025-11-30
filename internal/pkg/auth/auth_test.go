package auth

import (
	"net/http"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/tenant"
)

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "simple key",
			apiKey:   "test-key-123",
			expected: "625faa3fbbc3d2bd9d6ee7678d04cc5339cb33dc68d9b58451853d60046e226a",
		},
		{
			name:     "empty key",
			apiKey:   "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashAPIKey(tt.apiKey)
			if hash != tt.expected {
				t.Errorf("HashAPIKey() = %v, want %v", hash, tt.expected)
			}
		})
	}
}

func TestAuthenticator_ValidateAPIKey(t *testing.T) {
	// Create test tenants
	tenant1 := &tenant.Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant 1",
		APIKeys: []tenant.APIKey{
			{
				KeyHash:     HashAPIKey("valid-key-1"),
				Description: "Test key 1",
			},
		},
	}

	tenant2 := &tenant.Tenant{
		ID:   "tenant-2",
		Name: "Test Tenant 2",
		APIKeys: []tenant.APIKey{
			{
				KeyHash:     HashAPIKey("valid-key-2"),
				Description: "Test key 2",
			},
		},
	}

	auth := NewAuthenticator([]*tenant.Tenant{tenant1, tenant2})

	tests := []struct {
		name      string
		apiKey    string
		wantID    string
		wantError bool
	}{
		{
			name:      "valid key for tenant 1",
			apiKey:    "valid-key-1",
			wantID:    "tenant-1",
			wantError: false,
		},
		{
			name:      "valid key for tenant 2",
			apiKey:    "valid-key-2",
			wantID:    "tenant-2",
			wantError: false,
		},
		{
			name:      "invalid key",
			apiKey:    "invalid-key",
			wantError: true,
		},
		{
			name:      "empty key",
			apiKey:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := auth.ValidateAPIKey(tt.apiKey)

			if tt.wantError {
				if err == nil {
					t.Error("ValidateAPIKey() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateAPIKey() unexpected error: %v", err)
				return
			}

			if tenant.ID != tt.wantID {
				t.Errorf("ValidateAPIKey() tenant ID = %v, want %v", tenant.ID, tt.wantID)
			}
		})
	}
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       string
		wantError  bool
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer test-key-123",
			want:       "test-key-123",
			wantError:  false,
		},
		{
			name:       "bearer lowercase",
			authHeader: "bearer test-key-456",
			want:       "test-key-456",
			wantError:  false,
		},
		{
			name:       "missing bearer prefix",
			authHeader: "test-key-789",
			wantError:  true,
		},
		{
			name:       "empty header",
			authHeader: "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a real http.Request
			req, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			got, err := ExtractAPIKey(req)

			if tt.wantError {
				if err == nil {
					t.Error("ExtractAPIKey() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractAPIKey() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ExtractAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
