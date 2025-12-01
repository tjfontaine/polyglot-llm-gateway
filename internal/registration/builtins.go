package registration

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/anthropic"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/openai"
)

// RegisterBuiltins registers built-in providers and frontdoors explicitly.
// This replaces init-based side effects and is intended to be called from
// cmd/gateway and tests before wiring registries.
func RegisterBuiltins() {
	RegisterProviderBuiltins()
	RegisterFrontdoorBuiltins()
}

// RegisterProviderBuiltins registers built-in providers only.
func RegisterProviderBuiltins() {
	openai.RegisterProviderFactories()
	anthropic.RegisterProviderFactory()
}

// RegisterFrontdoorBuiltins registers built-in frontdoors only.
func RegisterFrontdoorBuiltins() {
	openai.RegisterFrontdoor()
	anthropic.RegisterFrontdoor()
}
