package runtime

import (
	"fmt"
	"log/slog"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/auth/apikey"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/config/file"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/events/direct"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/policy/basic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/adapters/storage/sqlite"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
)

// Option is a functional option for configuring a Gateway.
type Option func(*Gateway) error

// WithFileConfig uses file-based configuration with hot-reload (default).
// The path should point to a config.yaml file that will be watched for changes.
func WithFileConfig(path string) Option {
	return func(g *Gateway) error {
		provider, err := file.NewProvider(path)
		if err != nil {
			return fmt.Errorf("create file config provider: %w", err)
		}
		g.config = provider
		return nil
	}
}

// WithRemoteConfig uses remote API-based configuration.
// Useful for centralized config management in distributed deployments.
func WithRemoteConfig(baseURL string) Option {
	return func(g *Gateway) error {
		// TODO: Implement remote config provider
		// g.config = adapters.NewRemoteConfigProvider(baseURL)
		return nil
	}
}

// WithAPIKeyAuth uses API key-based authentication (default).
func WithAPIKeyAuth() Option {
	return func(g *Gateway) error {
		if g.config == nil {
			return fmt.Errorf("config provider must be set before auth provider")
		}
		provider, err := apikey.NewProvider(g.config)
		if err != nil {
			return fmt.Errorf("create apikey auth provider: %w", err)
		}
		g.auth = provider
		return nil
	}
}

// WithExternalAuth uses an external authentication provider.
// Supports OAuth2, OIDC, and other auth protocols.
func WithExternalAuth(provider string) Option {
	return func(g *Gateway) error {
		// TODO: Implement external auth
		// g.auth = adapters.NewExternalAuthProvider(provider)
		return nil
	}
}

// WithSQLite uses SQLite storage (default for single-instance deployments).
func WithSQLite(path string) Option {
	return func(g *Gateway) error {
		store, err := sqlite.NewProvider(path)
		if err != nil {
			return fmt.Errorf("create sqlite storage: %w", err)
		}
		g.storage = store
		return nil
	}
}

// WithPostgres uses PostgreSQL storage.
// Recommended for distributed deployments.
func WithPostgres(dsn string) Option {
	return func(g *Gateway) error {
		// TODO: Implement PostgreSQL storage adapter
		// store, err := adapters.NewPostgresStorage(dsn)
		// if err != nil {
		// 	return err
		// }
		// g.storage = store
		return nil
	}
}

// WithMySQL uses MySQL storage.
func WithMySQL(dsn string) Option {
	return func(g *Gateway) error {
		// TODO: Implement MySQL storage adapter
		// store, err := adapters.NewMySQLStorage(dsn)
		// if err != nil {
		// 	return err
		// }
		// g.storage = store
		return nil
	}
}

// WithDirectEvents writes events directly to storage (default).
// No separate event bus, events are written synchronously to storage.
func WithDirectEvents() Option {
	return func(g *Gateway) error {
		if g.storage == nil {
			return fmt.Errorf("storage provider must be set before event publisher")
		}
		publisher, err := direct.NewPublisher(g.storage)
		if err != nil {
			return fmt.Errorf("create direct event publisher: %w", err)
		}
		g.events = publisher
		return nil
	}
}

// WithKafkaEvents uses Kafka for event publishing.
// Enables decoupled consumers (billing, analytics, etc.).
func WithKafkaEvents(brokers []string, topic string) Option {
	return func(g *Gateway) error {
		// TODO: Implement Kafka publisher
		// g.events = adapters.NewKafkaPublisher(brokers, topic)
		return nil
	}
}

// WithNATSEvents uses NATS for event publishing.
// Lightweight alternative to Kafka for event streaming.
func WithNATSEvents(url string, subject string) Option {
	return func(g *Gateway) error {
		// TODO: Implement NATS publisher
		// g.events = adapters.NewNATSPublisher(url, subject)
		return nil
	}
}

// WithBasicPolicy uses the basic quality policy (no rate limiting).
func WithBasicPolicy() Option {
	return func(g *Gateway) error {
		g.policy = basic.NewPolicy()
		return nil
	}
}

// WithRateLimitPolicy enables rate limiting.
// Requires Redis for distributed rate limiting.
func WithRateLimitPolicy(redisURL string) Option {
	return func(g *Gateway) error {
		// TODO: Implement rate limit policy
		// g.policy = adapters.NewRateLimitPolicy(redisURL)
		return nil
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(g *Gateway) error {
		g.logger = logger
		return nil
	}
}

// WithConfigProvider sets a custom config provider.
// For advanced use cases where you need full control over config loading.
func WithConfigProvider(provider ports.ConfigProvider) Option {
	return func(g *Gateway) error {
		g.config = provider
		return nil
	}
}

// WithAuthProvider sets a custom auth provider.
func WithAuthProvider(provider ports.AuthProvider) Option {
	return func(g *Gateway) error {
		g.auth = provider
		return nil
	}
}

// WithStorageProvider sets a custom storage provider.
func WithStorageProvider(provider ports.StorageProvider) Option {
	return func(g *Gateway) error {
		g.storage = provider
		return nil
	}
}

// WithEventPublisher sets a custom event publisher.
func WithEventPublisher(publisher ports.EventPublisher) Option {
	return func(g *Gateway) error {
		g.events = publisher
		return nil
	}
}

// WithQualityPolicy sets a custom quality policy.
func WithQualityPolicy(policy ports.QualityPolicy) Option {
	return func(g *Gateway) error {
		g.policy = policy
		return nil
	}
}
