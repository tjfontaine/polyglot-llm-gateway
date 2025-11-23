package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/tjfontaine/poly-llm-gateway/internal/tenant"
)

// Authenticator validates API keys and extracts tenant information
type Authenticator struct {
	tenants map[string]*tenant.Tenant // keyhash -> tenant
}

// NewAuthenticator creates a new authenticator with tenant mappings
func NewAuthenticator(tenants []*tenant.Tenant) *Authenticator {
	auth := &Authenticator{
		tenants: make(map[string]*tenant.Tenant),
	}

	// Build keyhash -> tenant mapping
	for _, t := range tenants {
		for _, key := range t.APIKeys {
			auth.tenants[key.KeyHash] = t
		}
	}

	return auth
}

// ValidateAPIKey validates an API key and returns the associated tenant
func (a *Authenticator) ValidateAPIKey(apiKey string) (*tenant.Tenant, error) {
	// Hash the provided key
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Look up tenant by hash
	t, ok := a.tenants[keyHash]
	if !ok {
		return nil, fmt.Errorf("invalid API key")
	}

	// Constant-time comparison to prevent timing attacks
	for _, key := range t.APIKeys {
		if subtle.ConstantTimeCompare([]byte(keyHash), []byte(key.KeyHash)) == 1 {
			return t, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// ExtractAPIKey extracts the API key from the Authorization header
func ExtractAPIKey(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	// Support "Bearer <key>" format
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("unsupported authorization scheme")
	}

	return parts[1], nil
}

// HashAPIKey creates a SHA-256 hash of an API key for storage
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
