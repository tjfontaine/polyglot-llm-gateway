// Package registry provides frontdoor factory registration and lookup.
//
// # Adding a New Frontdoor
//
// Each frontdoor package should register itself via init():
//
//	func init() {
//	    registry.RegisterFactory(registry.FrontdoorFactory{
//	        Type:           FrontdoorType,
//	        APIType:        APIType(),
//	        Description:    "API format description",
//	        CreateHandlers: createHandlers,
//	    })
//	}
//
// Frontdoor packages must be imported (via blank import) in the parent
// frontdoor package to ensure their init() functions run.
package registry

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// HandlerConfig contains the configuration needed to create frontdoor handlers.
type HandlerConfig struct {
	// Provider is the backend provider to route requests to
	Provider ports.Provider

	// Store is the conversation store for persistence
	Store storage.ConversationStore

	// AppName identifies this frontdoor instance
	AppName string

	// BasePath is the URL path prefix for all handlers
	BasePath string

	// Models is the list of models to expose via this frontdoor
	Models []config.ModelListItem

	// ShadowConfig configures shadow mode for this frontdoor
	ShadowConfig *config.ShadowConfig

	// ProviderLookup is a function to resolve providers by name (for shadow)
	ProviderLookup func(name string) (ports.Provider, error)
}

// HandlerRegistration represents a registered HTTP handler.
type HandlerRegistration struct {
	Path    string
	Method  string
	Handler func(http.ResponseWriter, *http.Request)
}

// FrontdoorFactory defines how to create handlers for a specific API format.
// Each frontdoor type (openai, anthropic, etc.) registers a factory that
// knows how to create HTTP handlers from configuration.
type FrontdoorFactory struct {
	// Type is the frontdoor type identifier used in configuration
	// (e.g., "openai", "anthropic")
	Type string

	// APIType is the canonical API type this frontdoor accepts
	APIType domain.APIType

	// Description provides a human-readable description of the frontdoor
	Description string

	// CreateHandlers creates the HTTP handler registrations for this frontdoor.
	// The handlers should decode requests to canonical format and encode responses.
	CreateHandlers func(cfg HandlerConfig) []HandlerRegistration
}

// frontdoorRegistry holds registered frontdoor factories
var (
	frontdoorMu   sync.RWMutex
	frontdoorMap  = make(map[string]FrontdoorFactory)
	frontdoorList []FrontdoorFactory
)

// RegisterFactory registers a frontdoor factory for a specific type.
// This should be called from init() in each frontdoor package.
// Panics if a factory with the same type is already registered.
func RegisterFactory(f FrontdoorFactory) {
	frontdoorMu.Lock()
	defer frontdoorMu.Unlock()

	if f.Type == "" {
		panic("frontdoor factory type cannot be empty")
	}
	if f.CreateHandlers == nil {
		panic(fmt.Sprintf("frontdoor factory %q must have a CreateHandlers function", f.Type))
	}

	if _, exists := frontdoorMap[f.Type]; exists {
		panic(fmt.Sprintf("frontdoor factory %q already registered", f.Type))
	}

	frontdoorMap[f.Type] = f
	frontdoorList = append(frontdoorList, f)
}

// GetFactory returns the factory for a frontdoor type, if registered.
func GetFactory(frontdoorType string) (FrontdoorFactory, bool) {
	frontdoorMu.RLock()
	defer frontdoorMu.RUnlock()

	f, ok := frontdoorMap[frontdoorType]
	return f, ok
}

// ListFactories returns all registered frontdoor factories sorted by type.
func ListFactories() []FrontdoorFactory {
	frontdoorMu.RLock()
	defer frontdoorMu.RUnlock()

	result := make([]FrontdoorFactory, len(frontdoorList))
	copy(result, frontdoorList)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Type < result[j].Type
	})
	return result
}

// ListFrontdoorTypes returns all registered frontdoor type names.
func ListFrontdoorTypes() []string {
	factories := ListFactories()
	types := make([]string, len(factories))
	for i, f := range factories {
		types[i] = f.Type
	}
	return types
}

// IsRegistered returns true if a frontdoor type is registered.
func IsRegistered(frontdoorType string) bool {
	_, ok := GetFactory(frontdoorType)
	return ok
}

// CreateHandlersFromFactory creates handlers using the registered factory.
func CreateHandlersFromFactory(frontdoorType string, cfg HandlerConfig) ([]HandlerRegistration, error) {
	f, ok := GetFactory(frontdoorType)
	if !ok {
		return nil, fmt.Errorf("unknown frontdoor type: %s (registered types: %v)", frontdoorType, ListFrontdoorTypes())
	}

	return f.CreateHandlers(cfg), nil
}

// ClearFactories removes all registered factories (for testing only).
func ClearFactories() {
	frontdoorMu.Lock()
	defer frontdoorMu.Unlock()

	frontdoorMap = make(map[string]FrontdoorFactory)
	frontdoorList = nil
}
