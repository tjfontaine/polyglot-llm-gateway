package graph

import (
	"time"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/tenant"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver for all GraphQL queries
type Resolver struct {
	StartTime time.Time
	Config    *config.Config
	Store     storage.ConversationStore
	Tenants   []*tenant.Tenant
}

// NewResolver creates a new resolver with the given dependencies
func NewResolver(cfg *config.Config, store storage.ConversationStore, tenants []*tenant.Tenant) *Resolver {
	return &Resolver{
		StartTime: time.Now(),
		Config:    cfg,
		Store:     store,
		Tenants:   tenants,
	}
}
