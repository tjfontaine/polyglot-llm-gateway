// Package gateway provides the public API for embedding the LLM gateway.
// This is the stable API for external consumers.
package gateway

import (
	"github.com/tjfontaine/polyglot-llm-gateway/internal/runtime"
)

// Gateway is the main entry point for running the LLM gateway.
// See internal/runtime.Gateway for full documentation.
type Gateway = runtime.Gateway

// Option is a functional option for configuring a Gateway.
type Option = runtime.Option

// New creates a new Gateway with the given options.
// Example:
//
//	gw, err := gateway.New(
//	    gateway.WithFileConfig("config.yaml"),
//	    gateway.WithSQLite("./data/gateway.db"),
//	)
var New = runtime.New

// Configuration options
var (
	// Config sources
	WithFileConfig   = runtime.WithFileConfig
	WithRemoteConfig = runtime.WithRemoteConfig

	// Authentication
	WithAPIKeyAuth   = runtime.WithAPIKeyAuth
	WithExternalAuth = runtime.WithExternalAuth

	// Storage
	WithSQLite   = runtime.WithSQLite
	WithPostgres = runtime.WithPostgres
	WithMySQL    = runtime.WithMySQL

	// Events
	WithDirectEvents = runtime.WithDirectEvents
	WithKafkaEvents  = runtime.WithKafkaEvents
	WithNATSEvents   = runtime.WithNATSEvents

	// Policy
	WithBasicPolicy     = runtime.WithBasicPolicy
	WithRateLimitPolicy = runtime.WithRateLimitPolicy

	// Advanced options
	WithLogger          = runtime.WithLogger
	WithConfigProvider  = runtime.WithConfigProvider
	WithAuthProvider    = runtime.WithAuthProvider
	WithStorageProvider = runtime.WithStorageProvider
	WithEventPublisher  = runtime.WithEventPublisher
	WithQualityPolicy   = runtime.WithQualityPolicy
)
