package frontdoor

import (
	"os"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/registration"
)

func TestMain(m *testing.M) {
	ClearFrontdoorFactories()
	registration.RegisterBuiltins()
	os.Exit(m.Run())
}
