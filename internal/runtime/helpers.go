package runtime

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/ports"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/pkg/config"
)

// attachThreadStore attaches thread/event stores to providers that support state management.
func attachThreadStore(threadStore interface{ SetThreadState(string, string) error }, eventStore ports.StorageProvider, providers map[string]ports.Provider, configs []config.ProviderConfig) {
	if threadStore == nil && eventStore == nil {
		return
	}

	for _, cfg := range configs {
		prov, ok := providers[cfg.Name]
		if !ok {
			continue
		}

		// Attach thread store if provider supports it
		if cfg.ResponsesThreadPersistence {
			if setter, ok := prov.(interface {
				SetThreadStore(interface{ SetThreadState(string, string) error })
			}); ok {
				setter.SetThreadStore(threadStore)
			}
		}

		// Attach event store if provider supports it
		if eventStore != nil {
			if setter, ok := prov.(interface{ SetEventStore(ports.InteractionStore) }); ok {
				if interactionStore, ok := eventStore.(ports.InteractionStore); ok {
					setter.SetEventStore(interactionStore)
				}
			}
		}
	}
}
