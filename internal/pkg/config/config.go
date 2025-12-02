package config

import (
	"os"
	"regexp"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server     ServerConfig      `koanf:"server"`
	Storage    StorageConfig     `koanf:"storage"`
	Tenants    []TenantConfig    `koanf:"tenants"`
	Apps       []AppConfig       `koanf:"apps"`
	Frontdoors []FrontdoorConfig `koanf:"frontdoors"`
	Providers  []ProviderConfig  `koanf:"providers"`
	Routing    RoutingConfig     `koanf:"routing"`
	// Legacy fields for backwards compatibility
	OpenAI    OpenAIConfig    `koanf:"openai"`
	Anthropic AnthropicConfig `koanf:"anthropic"`
}

type ServerConfig struct {
	Port int `koanf:"port"`
}

type StorageConfig struct {
	Type   string       `koanf:"type"` // sqlite, postgres, mysql, memory, none
	SQLite SQLiteConfig `koanf:"sqlite"`
	// Database is the generic database configuration for multi-dialect support
	Database DatabaseConfig `koanf:"database"`
}

type SQLiteConfig struct {
	Path string `koanf:"path"`
}

// DatabaseConfig is the generic database configuration supporting multiple dialects.
type DatabaseConfig struct {
	Driver string `koanf:"driver"` // sqlite, postgres, mysql
	DSN    string `koanf:"dsn"`    // Data source name / connection string
}

type TenantConfig struct {
	ID        string           `koanf:"id"`
	Name      string           `koanf:"name"`
	APIKeys   []APIKeyConfig   `koanf:"api_keys"`
	Providers []ProviderConfig `koanf:"providers"`
	Routing   RoutingConfig    `koanf:"routing"`
}

type APIKeyConfig struct {
	KeyHash     string `koanf:"key_hash"`
	Description string `koanf:"description"`
}

type AppConfig struct {
	Name            string             `koanf:"name"`
	Frontdoor       string             `koanf:"frontdoor"`
	Path            string             `koanf:"path"`
	Provider        string             `koanf:"provider"`      // Optional: force specific provider
	DefaultModel    string             `koanf:"default_model"` // Optional: force/default model
	ModelRouting    ModelRoutingConfig `koanf:"model_routing"`
	Models          []ModelListItem    `koanf:"models"`
	EnableResponses bool               `koanf:"enable_responses"` // Optional: mount Responses API for this frontdoor
	ForceStore      bool               `koanf:"force_store"`      // Optional: force recording even when client sends store:false
	Shadow          ShadowConfig       `koanf:"shadow"`           // Optional: shadow mode configuration
}

// ShadowConfig configures shadow mode for an app.
// When enabled, requests are also sent to shadow providers for comparison.
type ShadowConfig struct {
	Enabled           bool                   `koanf:"enabled"`
	Providers         []ShadowProviderConfig `koanf:"providers"`
	Timeout           string                 `koanf:"timeout"`             // Duration string like "30s"
	StoreStreamChunks bool                   `koanf:"store_stream_chunks"` // Whether to store individual stream chunks
}

// ShadowProviderConfig configures a single shadow provider.
type ShadowProviderConfig struct {
	Name                string   `koanf:"name"`                  // Provider name (must match a configured provider)
	Model               string   `koanf:"model"`                 // Optional: override model for this shadow
	MaxTokensMultiplier *float64 `koanf:"max_tokens_multiplier"` // Multiplier for max_tokens (0/nil=unlimited, 1=same, >1=increase). Default 0.
}

type FrontdoorConfig struct {
	Type            string `koanf:"type"`
	Path            string `koanf:"path"`
	Provider        string `koanf:"provider"`         // Optional: force specific provider
	DefaultModel    string `koanf:"default_model"`    // Optional: force/default model
	EnableResponses bool   `koanf:"enable_responses"` // Optional: mount Responses API for this frontdoor
}

type ProviderConfig struct {
	Name                       string `koanf:"name"`
	Type                       string `koanf:"type"`
	APIKey                     string `koanf:"api_key"`
	BaseURL                    string `koanf:"base_url"`                     // Custom API endpoint
	SupportsResponses          bool   `koanf:"supports_responses"`           // Flag for native Responses API support
	EnablePassthrough          bool   `koanf:"enable_passthrough"`           // Enable pass-through mode when frontdoor matches
	UseResponsesAPI            bool   `koanf:"use_responses_api"`            // Use Responses API instead of Chat Completions (OpenAI)
	ResponsesThreadKeyPath     string `koanf:"responses_thread_key_path"`    // Optional dotted JSON path to derive Responses threading key
	ResponsesThreadPersistence bool   `koanf:"responses_thread_persistence"` // Persist thread state across restarts when storage is available
}

type RoutingConfig struct {
	Rules           []RoutingRule `koanf:"rules"`
	DefaultProvider string        `koanf:"default_provider"`
}

type RoutingRule struct {
	ModelPrefix string `koanf:"model_prefix"`
	ModelExact  string `koanf:"model_exact"`
	Provider    string `koanf:"provider"`
}

type ModelRoutingConfig struct {
	PrefixProviders map[string]string  `koanf:"prefix_providers"`
	Rewrites        []ModelRewriteRule `koanf:"rewrites"`
	Fallback        *ModelRewriteRule  `koanf:"fallback"`
}

type ModelRewriteRule struct {
	Match                string `koanf:"match"` // Deprecated: use ModelExact or ModelPrefix
	ModelExact           string `koanf:"model_exact"`
	ModelPrefix          string `koanf:"model_prefix"`
	Provider             string `koanf:"provider"`
	Model                string `koanf:"model"`
	RewriteResponseModel bool   `koanf:"rewrite_response_model"`
}

type ModelListItem struct {
	ID      string `koanf:"id"`
	Object  string `koanf:"object"`
	OwnedBy string `koanf:"owned_by"`
	Created int64  `koanf:"created"`
}

type OpenAIConfig struct {
	APIKey string `koanf:"api_key"`
}

type AnthropicConfig struct {
	APIKey string `koanf:"api_key"`
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func Load() (*Config, error) {
	k := koanf.New(".")

	// Try to load from config.yaml file first
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		// File not found is OK, we'll use env vars
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Load environment variables (can override file config)
	if err := k.Load(env.Provider("POLY_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "POLY_")), "__", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	// Default values
	if !k.Exists("server.port") {
		k.Set("server.port", 8080)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	// Substitute environment variables in provider API keys
	for i := range cfg.Providers {
		cfg.Providers[i].APIKey = substituteEnvVars(cfg.Providers[i].APIKey)
	}

	return &cfg, nil
}

func substituteEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}
