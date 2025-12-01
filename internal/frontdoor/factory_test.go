package frontdoor_test

import (
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/frontdoor"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/registration"
)

func init() {
	frontdoor.ClearFactories()
	registration.RegisterBuiltins()
}

func TestListFrontdoorTypes(t *testing.T) {
	types := frontdoor.ListFrontdoorTypes()
	if len(types) < 2 {
		t.Errorf("expected at least 2 frontdoor types, got %d", len(types))
	}

	// Check for expected types
	expected := []string{"anthropic", "openai"}
	typeSet := make(map[string]bool)
	for _, tp := range types {
		typeSet[tp] = true
	}

	for _, exp := range expected {
		if !typeSet[exp] {
			t.Errorf("expected frontdoor type %q to be registered", exp)
		}
	}
}

func TestListFrontdoorFactories(t *testing.T) {
	factories := frontdoor.ListFactories()
	if len(factories) < 2 {
		t.Errorf("expected at least 2 factories, got %d", len(factories))
	}

	// Verify factories have required fields
	for _, f := range factories {
		if f.Type == "" {
			t.Error("factory has empty Type")
		}
		if f.CreateHandlers == nil {
			t.Errorf("factory %q has nil CreateHandlers function", f.Type)
		}
		if f.Description == "" {
			t.Errorf("factory %q has empty Description", f.Type)
		}
	}
}

func TestGetFrontdoorFactory(t *testing.T) {
	tests := []struct {
		frontdoorType string
		wantOk        bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.frontdoorType, func(t *testing.T) {
			f, ok := frontdoor.GetFactory(tt.frontdoorType)
			if ok != tt.wantOk {
				t.Errorf("GetFrontdoorFactory(%q) returned ok=%v, want %v", tt.frontdoorType, ok, tt.wantOk)
			}
			if tt.wantOk {
				if f.Type != tt.frontdoorType {
					t.Errorf("GetFrontdoorFactory(%q) returned factory with Type=%q", tt.frontdoorType, f.Type)
				}
			}
		})
	}
}

func TestIsFrontdoorRegistered(t *testing.T) {
	tests := []struct {
		frontdoorType string
		want          bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"gemini", false},
	}

	for _, tt := range tests {
		t.Run(tt.frontdoorType, func(t *testing.T) {
			got := frontdoor.IsRegistered(tt.frontdoorType)
			if got != tt.want {
				t.Errorf("IsRegistered(%q) = %v, want %v", tt.frontdoorType, got, tt.want)
			}
		})
	}
}

func TestFrontdoorAPITypes(t *testing.T) {
	tests := []struct {
		frontdoorType string
		wantAPIType   domain.APIType
	}{
		{"openai", domain.APITypeOpenAI},
		{"anthropic", domain.APITypeAnthropic},
	}

	for _, tt := range tests {
		t.Run(tt.frontdoorType, func(t *testing.T) {
			f, ok := frontdoor.GetFactory(tt.frontdoorType)
			if !ok {
				t.Fatalf("factory %q not found", tt.frontdoorType)
			}
			if f.APIType != tt.wantAPIType {
				t.Errorf("factory %q has APIType=%v, want %v", tt.frontdoorType, f.APIType, tt.wantAPIType)
			}
		})
	}
}
