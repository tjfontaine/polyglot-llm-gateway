package frontdoor_test

import (
	"os"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/registration"
)

func TestMain(m *testing.M) {
	frontdoor.ClearFactories()
	registration.RegisterBuiltins()
	os.Exit(m.Run())
}
