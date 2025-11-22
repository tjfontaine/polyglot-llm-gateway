package main

import (
	"log"

	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	openai_frontdoor "github.com/tjfontaine/poly-llm-gateway/internal/frontdoor/openai"
	openai_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/openai"
	"github.com/tjfontaine/poly-llm-gateway/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Provider
	// In Phase 1, we just use OpenAI provider directly
	provider := openai_provider.New(cfg.OpenAI.APIKey)

	// Initialize Frontdoor
	handler := openai_frontdoor.NewHandler(provider)

	// Initialize Server
	srv := server.New(cfg.Server.Port)

	// Register Routes
	srv.Router.Post("/v1/chat/completions", handler.HandleChatCompletion)

	log.Printf("Starting server on port %d", cfg.Server.Port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
