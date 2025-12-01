.PHONY: all build frontend frontend-dev backend clean run generate generate-go generate-frontend

# Default to development mode
BUILD_MODE ?= development

all: build

build: generate frontend backend

# Generate all code (GraphQL types for Go and TypeScript)
generate: generate-go generate-frontend

# Generate Go GraphQL types using gqlgen
generate-go:
	@echo "Generating Go GraphQL types..."
	cd internal/api/controlplane/graph && go run github.com/99designs/gqlgen generate

# Generate TypeScript GraphQL types using graphql-codegen
generate-frontend:
	@echo "Generating TypeScript GraphQL types..."
	cd web/control-plane && npm ci && npm run codegen

# Production build (minified)
frontend: generate-frontend
	@echo "Building frontend ($(BUILD_MODE) mode)..."
	cd web/control-plane && npm run build
	@echo "Copying frontend assets..."
	rm -rf internal/api/controlplane/dist
	cp -r web/control-plane/dist internal/api/controlplane/

# Development build (unminified, better error messages)
frontend-dev: generate-frontend
	@echo "Building frontend (development mode)..."
	cd web/control-plane && npm run build:dev
	@echo "Copying frontend assets..."
	rm -rf internal/api/controlplane/dist
	cp -r web/control-plane/dist internal/api/controlplane/

backend: generate-go
	@echo "Building backend..."
	go build -o bin/gateway ./cmd/gateway

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf web/control-plane/dist
	rm -rf web/control-plane/node_modules
	rm -rf internal/api/controlplane/dist

run: build
	./bin/gateway
