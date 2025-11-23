package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	origPort := os.Getenv("POLY_SERVER__PORT")
	defer func() {
		if origPort != "" {
			os.Setenv("POLY_SERVER__PORT", origPort)
		} else {
			os.Unsetenv("POLY_SERVER__PORT")
		}
	}()

	t.Run("default port", func(t *testing.T) {
		os.Unsetenv("POLY_SERVER__PORT")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Server.Port != 8080 {
			t.Errorf("Load() port = %v, want 8080", cfg.Server.Port)
		}
	})

	t.Run("env var port override", func(t *testing.T) {
		os.Setenv("POLY_SERVER__PORT", "9000")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Server.Port != 9000 {
			t.Errorf("Load() port = %v, want 9000", cfg.Server.Port)
		}
	})
}

func TestSubstituteEnvVars(t *testing.T) {
	// Set test env var
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple substitution",
			input: "${TEST_VAR}",
			want:  "test-value",
		},
		{
			name:  "substitution in string",
			input: "prefix-${TEST_VAR}-suffix",
			want:  "prefix-test-value-suffix",
		},
		{
			name:  "no substitution",
			input: "plain-string",
			want:  "plain-string",
		},
		{
			name:  "undefined var",
			input: "${UNDEFINED_VAR}",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := substituteEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("substituteEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}
