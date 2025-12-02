package ports

import (
	"context"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// ConfigProvider loads and manages configuration.
// Implementations: file-based (default), remote API, etc.
type ConfigProvider interface {
	Load(ctx context.Context) (*config.Config, error)
	Watch(ctx context.Context, onChange func(*config.Config)) error
	Close() error
}

// AuthProvider manages authentication and authorization.
// Implementations: API key (default), OAuth2, OIDC, etc.
type AuthProvider interface {
	Authenticate(ctx context.Context, token string) (*AuthContext, error)
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
}

// AuthContext contains authenticated request context.
type AuthContext struct {
	TenantID string
	UserID   string
	Scopes   []string
	Metadata map[string]string
}

// Tenant represents a tenant in the system.
type Tenant struct {
	ID        string
	Name      string
	Providers map[string]Provider
	Routing   config.RoutingConfig
}

// StorageProvider manages all storage operations.
// Implementations: SQLite (default), PostgreSQL, MySQL
type StorageProvider interface {
	ConversationStore
	InteractionStore
	ResponseStore
	ShadowStore

	// ThreadState management
	SetThreadState(threadKey, responseID string) error
	GetThreadState(threadKey string) (string, error)

	Close() error
}

// EventPublisher publishes interaction lifecycle events.
// Implementations: direct storage (default), Kafka, NATS, etc.
type EventPublisher interface {
	Publish(ctx context.Context, event *domain.LifecycleEvent) error
	Close() error
}

// QualityPolicy enforces rate limits and policies.
// Implementations: basic (no limits), rate limiter, quota manager
type QualityPolicy interface {
	CheckRequest(ctx context.Context, req *PolicyRequest) (*PolicyDecision, error)
	RecordUsage(ctx context.Context, usage *UsageRecord) error
}

// PolicyRequest contains request context for policy checks.
type PolicyRequest struct {
	TenantID string
	UserID   string
	Model    string
	Tokens   int
}

// PolicyDecision is the result of a policy check.
type PolicyDecision struct {
	Allow         bool
	Reason        string
	RetryAfter    int // seconds
	RateLimitInfo *RateLimitInfo
}

// RateLimitInfo contains rate limit information.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	ResetAt   int64 // Unix timestamp
}

// UsageRecord tracks resource usage.
type UsageRecord struct {
	TenantID         string
	UserID           string
	InteractionID    string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
