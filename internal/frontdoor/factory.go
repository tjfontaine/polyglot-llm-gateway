// Package frontdoor contains the frontdoor factory and registry for API handlers.
//
// # Adding a New Frontdoor
//
// To add a new frontdoor (e.g., for Google Gemini API format), you must implement
// the following components and register them with the factory:
//
//  1. Handler: Create `internal/frontdoor/<api>/handler.go` with HTTP handlers
//     that decode API-specific requests to canonical format.
//
//  2. Codec: Create `internal/codec/<api>/codec.go` implementing the
//     `codec.Codec` interface for request/response translation.
//
//  3. Factory Registration: Add a FrontdoorFactory to this package's init().
//
// Example factory registration:
//
//	func init() {
//	    frontdoor.RegisterFactory(frontdoor.FrontdoorFactory{
//	        Type:        "gemini",
//	        APIType:     domain.APITypeGemini,
//	        Description: "Google Gemini API format",
//	        CreateHandlers: func(cfg HandlerConfig) []HandlerRegistration {
//	            handler := gemini.NewHandler(cfg.Provider, cfg.Store, cfg.AppName, cfg.Models)
//	            return []HandlerRegistration{
//	                {Path: cfg.BasePath + "/v1/models:generateContent", Method: http.MethodPost, Handler: handler.HandleGenerate},
//	                {Path: cfg.BasePath + "/v1/models:streamGenerateContent", Method: http.MethodPost, Handler: handler.HandleStream},
//	            }
//	        },
//	    })
//	}
package frontdoor

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/storage"
)

// HandlerConfig contains the configuration needed to create frontdoor handlers.
type HandlerConfig struct {
	// Provider is the backend provider to route requests to
	Provider domain.Provider

	// Store is the conversation store for persistence
	Store storage.ConversationStore

	// AppName identifies this frontdoor instance
	AppName string

	// BasePath is the URL path prefix for all handlers
	BasePath string

	// Models is the list of models to expose via this frontdoor
	Models []config.ModelListItem
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
// This should be called from init() in this package.
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

// GetFrontdoorFactory returns the factory for a frontdoor type, if registered.
func GetFrontdoorFactory(frontdoorType string) (FrontdoorFactory, bool) {
	frontdoorMu.RLock()
	defer frontdoorMu.RUnlock()

	f, ok := frontdoorMap[frontdoorType]
	return f, ok
}

// ListFrontdoorFactories returns all registered frontdoor factories sorted by type.
func ListFrontdoorFactories() []FrontdoorFactory {
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
	factories := ListFrontdoorFactories()
	types := make([]string, len(factories))
	for i, f := range factories {
		types[i] = f.Type
	}
	return types
}

// IsFrontdoorRegistered returns true if a frontdoor type is registered.
func IsFrontdoorRegistered(frontdoorType string) bool {
	_, ok := GetFrontdoorFactory(frontdoorType)
	return ok
}

// createHandlersFromFactory creates handlers using the registered factory.
func createHandlersFromFactory(frontdoorType string, cfg HandlerConfig) ([]HandlerRegistration, error) {
	f, ok := GetFrontdoorFactory(frontdoorType)
	if !ok {
		return nil, fmt.Errorf("unknown frontdoor type: %s (registered types: %v)", frontdoorType, ListFrontdoorTypes())
	}

	return f.CreateHandlers(cfg), nil
}

// ClearFrontdoorFactories removes all registered factories (for testing only).
func ClearFrontdoorFactories() {
	frontdoorMu.Lock()
	defer frontdoorMu.Unlock()

	frontdoorMap = make(map[string]FrontdoorFactory)
	frontdoorList = nil
}
