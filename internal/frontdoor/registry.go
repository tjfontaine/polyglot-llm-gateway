// Package frontdoor provides frontdoor factory registration and the Registry
// for creating HTTP handlers from configuration.
//
// # Adding a New Frontdoor
//
// To add a new frontdoor (e.g., for Google Gemini API format), implement the
// handler + codec and expose an explicit registration function that calls
// RegisterFactory. Wire that function from cmd/gateway (or tests) so
// registration is explicit instead of relying on init() side effects.
//
// Example in a frontdoor package:
//
//	func RegisterFrontdoor() {
//	    if frontdoor.IsRegistered(FrontdoorType) {
//	        return
//	    }
//	    frontdoor.RegisterFactory(frontdoor.FrontdoorFactory{
//	        Type:           FrontdoorType,
//	        APIType:        FrontdoorAPIType(),
//	        Description:    "Gemini API format",
//	        CreateHandlers: createHandlers,
//	    })
//	}
package frontdoor

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/provider"
	responses_frontdoor "github.com/tjfontaine/polyglot-llm-gateway/internal/responses"
	routerpkg "github.com/tjfontaine/polyglot-llm-gateway/internal/router"
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

// Factory registry: global registration of frontdoor factories
var (
	factoryMu   sync.RWMutex
	factoryMap  = make(map[string]FrontdoorFactory)
	factoryList []FrontdoorFactory
)

// RegisterFactory registers a frontdoor factory for a specific type.
// Panics if a factory with the same type is already registered.
func RegisterFactory(f FrontdoorFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()

	if f.Type == "" {
		panic("frontdoor factory type cannot be empty")
	}
	if f.CreateHandlers == nil {
		panic(fmt.Sprintf("frontdoor factory %q must have a CreateHandlers function", f.Type))
	}

	if _, exists := factoryMap[f.Type]; exists {
		panic(fmt.Sprintf("frontdoor factory %q already registered", f.Type))
	}

	factoryMap[f.Type] = f
	factoryList = append(factoryList, f)
}

// GetFactory returns the factory for a frontdoor type, if registered.
func GetFactory(frontdoorType string) (FrontdoorFactory, bool) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()

	f, ok := factoryMap[frontdoorType]
	return f, ok
}

// ListFactories returns all registered frontdoor factories sorted by type.
func ListFactories() []FrontdoorFactory {
	factoryMu.RLock()
	defer factoryMu.RUnlock()

	result := make([]FrontdoorFactory, len(factoryList))
	copy(result, factoryList)
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
	factoryMu.Lock()
	defer factoryMu.Unlock()

	factoryMap = make(map[string]FrontdoorFactory)
	factoryList = nil
}

// Registry creates frontdoor handlers from configuration.
// It uses registered FrontdoorFactory instances to create handlers.
type Registry struct{}

// NewRegistry creates a new frontdoor registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// CreateHandlers creates frontdoor handlers based on configuration.
// It uses the registered FrontdoorFactory for each specified frontdoor type.
func (r *Registry) CreateHandlers(configs []config.AppConfig, router ports.Provider, providers map[string]ports.Provider, store storage.ConversationStore) ([]HandlerRegistration, error) {
	var registrations []HandlerRegistration

	for _, cfg := range configs {
		// Determine which provider to use
		var p ports.Provider = router
		if cfg.Provider != "" {
			if specificProvider, ok := providers[cfg.Provider]; ok {
				p = specificProvider
			} else {
				return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
			}
		}

		// Apply model override if configured
		if cfg.DefaultModel != "" {
			p = provider.NewModelOverrideProvider(p, cfg.DefaultModel)
		}

		if len(cfg.ModelRouting.PrefixProviders) > 0 || len(cfg.ModelRouting.Rewrites) > 0 || cfg.ModelRouting.Fallback != nil {
			mapper, err := routerpkg.NewMappingProvider(p, providers, cfg.ModelRouting)
			if err != nil {
				return nil, err
			}
			p = mapper
		}

		// Use the factory pattern to create handlers
		handlerCfg := HandlerConfig{
			Provider:     p,
			Store:        store,
			AppName:      cfg.Name,
			BasePath:     cfg.Path,
			Models:       cfg.Models,
			ShadowConfig: &cfg.Shadow,
		}

		handlers, err := CreateHandlersFromFactory(cfg.Frontdoor, handlerCfg)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, handlers...)
	}

	return registrations, nil
}

// ResponsesHandlerOptions configures Responses API handler behavior
type ResponsesHandlerOptions struct {
	ForceStore bool // Force recording even when client sends store:false
}

// CreateResponsesHandlers creates Responses API handlers.
func (r *Registry) CreateResponsesHandlers(basePath string, store ports.InteractionStore, provider ports.Provider, opts ...ResponsesHandlerOptions) []HandlerRegistration {
	var handlerOpts responses_frontdoor.HandlerOptions
	if len(opts) > 0 {
		handlerOpts.ForceStore = opts[0].ForceStore
	}
	handler := responses_frontdoor.NewHandler(store, provider, handlerOpts)

	return []HandlerRegistration{
		// Responses API (new)
		{Path: basePath + "/v1/responses", Method: http.MethodPost, Handler: handler.HandleCreateResponse},
		{Path: basePath + "/v1/responses/{response_id}", Method: http.MethodGet, Handler: handler.HandleGetResponse},
		{Path: basePath + "/v1/responses/{response_id}/cancel", Method: http.MethodPost, Handler: handler.HandleCancelResponse},

		// Threads API (legacy)
		{Path: basePath + "/v1/threads", Method: http.MethodPost, Handler: handler.HandleCreateThread},
		{Path: basePath + "/v1/threads/{thread_id}", Method: http.MethodGet, Handler: handler.HandleGetThread},
		{Path: basePath + "/v1/threads/{thread_id}/messages", Method: http.MethodPost, Handler: handler.HandleCreateMessage},
		{Path: basePath + "/v1/threads/{thread_id}/messages", Method: http.MethodGet, Handler: handler.HandleListMessages},
		{Path: basePath + "/v1/threads/{thread_id}/runs", Method: http.MethodPost, Handler: handler.HandleCreateRun},
	}
}
