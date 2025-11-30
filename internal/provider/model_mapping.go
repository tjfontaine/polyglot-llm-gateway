package provider

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	routerpkg "github.com/tjfontaine/polyglot-llm-gateway/internal/router"
)

// ModelMappingProvider is kept for backward compatibility; it delegates to router.MappingProvider.
type ModelMappingProvider = routerpkg.MappingProvider

// NewModelMappingProvider wraps router.NewMappingProvider.
func NewModelMappingProvider(defaultProvider ports.Provider, providers map[string]ports.Provider, cfg config.ModelRoutingConfig) (*routerpkg.MappingProvider, error) {
	return routerpkg.NewMappingProvider(defaultProvider, providers, cfg)
}
