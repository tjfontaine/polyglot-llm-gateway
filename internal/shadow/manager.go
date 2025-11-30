package shadow

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/codec"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// Manager coordinates shadow execution across the gateway.
// It provides a simple interface for handlers to trigger shadow execution.
type Manager struct {
	store          ports.ShadowStore
	providerLookup ProviderLookup
	codecLookup    CodecLookup
	executor       *Executor
	logger         *slog.Logger
}

// ManagerConfig configures a shadow Manager.
type ManagerConfig struct {
	Store          ports.ShadowStore
	ProviderLookup ProviderLookup
	CodecLookup    CodecLookup
	Logger         *slog.Logger
}

// NewManager creates a new shadow Manager.
func NewManager(cfg ManagerConfig) *Manager {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	m := &Manager{
		store:          cfg.Store,
		providerLookup: cfg.ProviderLookup,
		codecLookup:    cfg.CodecLookup,
		logger:         logger,
	}

	if cfg.Store != nil && cfg.ProviderLookup != nil && cfg.CodecLookup != nil {
		m.executor = NewExecutor(cfg.Store, cfg.ProviderLookup, cfg.CodecLookup, WithLogger(logger))
	}

	return m
}

// TriggerShadow triggers shadow execution for a request.
// It runs asynchronously in a goroutine and doesn't block the caller.
// This should be called after the primary response is obtained.
func (m *Manager) TriggerShadow(
	_ context.Context, // Original context (not used - we use background context to avoid cancellation)
	shadowConfig *config.ShadowConfig,
	interactionID string,
	canonReq *domain.CanonicalRequest,
	primaryResp *domain.CanonicalResponse,
	frontdoorAPIType domain.APIType,
) {
	if m.executor == nil || shadowConfig == nil || !shadowConfig.Enabled {
		return
	}

	if len(shadowConfig.Providers) == 0 {
		return
	}

	// Run shadow execution in background with its own context
	// We use Background() instead of the request context to avoid cancellation when the response is sent
	go func() {
		// Create a new context with timeout from config
		ctx := context.Background()
		if shadowConfig.Timeout != "" {
			timeout, err := time.ParseDuration(shadowConfig.Timeout)
			if err == nil && timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}

		req := &ExecuteRequest{
			InteractionID:    interactionID,
			CanonicalRequest: canonReq,
			ShadowConfig:     shadowConfig,
			FrontdoorAPIType: frontdoorAPIType,
			PrimaryResponse:  primaryResp,
		}

		result := m.executor.Execute(ctx, req)

		// Log results
		for _, sr := range result.Results {
			if sr.Error != nil {
				m.logger.Warn("shadow execution error",
					"interaction_id", interactionID,
					"shadow_provider", sr.ProviderName,
					"error_type", sr.Error.Type,
					"error_message", sr.Error.Message,
				)
			} else if sr.HasStructuralDivergence {
				m.logger.Info("shadow divergence detected",
					"interaction_id", interactionID,
					"shadow_provider", sr.ProviderName,
					"divergence_count", len(sr.Divergences),
				)
			} else {
				m.logger.Debug("shadow execution completed",
					"interaction_id", interactionID,
					"shadow_provider", sr.ProviderName,
					"duration_ms", sr.Duration.Milliseconds(),
				)
			}
		}

		// Log any execution errors
		for _, err := range result.Errors {
			m.logger.Error("shadow execution failed",
				"interaction_id", interactionID,
				"error", err,
			)
		}
	}()
}

// Global manager instance (set by main)
var globalManager *Manager

// SetGlobalManager sets the global shadow manager.
func SetGlobalManager(m *Manager) {
	globalManager = m
}

// GetGlobalManager returns the global shadow manager.
func GetGlobalManager() *Manager {
	return globalManager
}

// TriggerGlobalShadow triggers shadow execution using the global manager.
// This is a convenience function for handlers that don't have direct access to the manager.
func TriggerGlobalShadow(
	ctx context.Context,
	shadowConfig *config.ShadowConfig,
	interactionID string,
	canonReq *domain.CanonicalRequest,
	primaryResp *domain.CanonicalResponse,
	frontdoorAPIType domain.APIType,
) {
	if globalManager != nil {
		globalManager.TriggerShadow(ctx, shadowConfig, interactionID, canonReq, primaryResp, frontdoorAPIType)
	}
}

// DefaultCodecLookup returns a codec lookup function using the standard codecs.
// Note: This requires the codec registry to be set up by the main package.
func DefaultCodecLookup() CodecLookup {
	return func(apiType domain.APIType) (codec.Codec, error) {
		return codecRegistry.Get(apiType)
	}
}

// codecRegistry is a simple registry for codecs
var codecRegistry = &codecRegistryImpl{
	codecs: make(map[domain.APIType]codec.Codec),
}

type codecRegistryImpl struct {
	codecs map[domain.APIType]codec.Codec
}

func (r *codecRegistryImpl) Register(apiType domain.APIType, c codec.Codec) {
	r.codecs[apiType] = c
}

func (r *codecRegistryImpl) Get(apiType domain.APIType) (codec.Codec, error) {
	c, ok := r.codecs[apiType]
	if !ok {
		return nil, fmt.Errorf("codec not found for API type: %s", apiType)
	}
	return c, nil
}

// RegisterCodec registers a codec for an API type.
func RegisterCodec(apiType domain.APIType, c codec.Codec) {
	codecRegistry.Register(apiType, c)
}
