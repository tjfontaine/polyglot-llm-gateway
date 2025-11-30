package registration

import (
	apiAnthropic "github.com/tjfontaine/polyglot-llm-gateway/internal/api/anthropic"
	apiOpenAI "github.com/tjfontaine/polyglot-llm-gateway/internal/api/openai"
	backendAnthropic "github.com/tjfontaine/polyglot-llm-gateway/internal/backend/anthropic"
	backendOpenAI "github.com/tjfontaine/polyglot-llm-gateway/internal/backend/openai"
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
	backendOpenAI.RegisterProviderFactories()
	backendAnthropic.RegisterProviderFactory()
}

// RegisterFrontdoorBuiltins registers built-in frontdoors only.
func RegisterFrontdoorBuiltins() {
	apiOpenAI.RegisterFrontdoor()
	apiAnthropic.RegisterFrontdoor()
}
