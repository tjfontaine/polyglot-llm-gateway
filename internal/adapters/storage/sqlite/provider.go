// Package sqlite provides SQLite storage adapter for the gateway.
package sqlite

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage/sqldb"
)

// Provider implements ports.StorageProvider using SQLite.
// It wraps the existing sqldb implementation.
type Provider struct {
	*sqldb.Store
}

// NewProvider creates a new SQLite storage provider.
func NewProvider(path string) (*Provider, error) {
	store, err := sqldb.NewSQLite(path)
	if err != nil {
		return nil, err
	}

	return &Provider{
		Store: store,
	}, nil
}

// Ensure Provider implements ports.StorageProvider at compile time.
var _ ports.StorageProvider = (*Provider)(nil)
