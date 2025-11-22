package main

import (
	"log"

	"github.com/tjfontaine/poly-llm-gateway/internal/config"
	openai_frontdoor "github.com/tjfontaine/poly-llm-gateway/internal/frontdoor/openai"
	"github.com/tjfontaine/poly-llm-gateway/internal/policy"
	anthropic_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/anthropic"
	openai_provider "github.com/tjfontaine/poly-llm-gateway/internal/provider/openai"
	"github.com/tjfontaine/poly-llm-gateway/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Providers
	openaiP := openai_provider.New(cfg.OpenAI.APIKey)
	anthropicP := anthropic_provider.New(cfg.Anthropic.APIKey)

	// Initialize Router
	router := policy.NewRouter(openaiP, anthropicP)

	// Initialize Frontdoor with Router
	handler := openai_frontdoor.NewHandler(router)

	// Initialize Server
	srv := server.New(cfg.Server.Port)

	// Register Routes
	srv.Router.Post("/v1/chat/completions", handler.HandleChatCompletion)

	log.Printf("Starting server on port %d", cfg.Server.Port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
