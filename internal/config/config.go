package config

import (
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server ServerConfig `koanf:"server"`
	OpenAI OpenAIConfig `koanf:"openai"`
}

type ServerConfig struct {
	Port int `koanf:"port"`
}

type OpenAIConfig struct {
	APIKey string `koanf:"api_key"`
}

func Load() (*Config, error) {
	k := koanf.New(".")

	// Load environment variables
	if err := k.Load(env.Provider("POLY_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "POLY_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	// Default values
	if !k.Exists("server.port") {
		k.Set("server.port", 8080)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
