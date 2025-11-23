package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/keygen/main.go <api-key>")
		fmt.Println("Generates a SHA-256 hash of the provided API key for use in config.yaml")
		os.Exit(1)
	}

	apiKey := os.Args[1]
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("SHA-256 Hash: %s\n", keyHash)
	fmt.Println("\nAdd this to your config.yaml:")
	fmt.Printf("  api_keys:\n")
	fmt.Printf("    - key_hash: \"%s\"\n", keyHash)
	fmt.Printf("      description: \"Generated key\"\n")
}
