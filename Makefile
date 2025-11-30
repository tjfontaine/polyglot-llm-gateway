.PHONY: all build frontend frontend-dev backend clean run

# Default to development mode
BUILD_MODE ?= development

all: build

build: frontend backend

# Production build (minified)
frontend:
	@echo "Building frontend ($(BUILD_MODE) mode)..."
	cd web/control-plane && npm ci && npm run build
	@echo "Copying frontend assets..."
	rm -rf internal/api/controlplane/dist
	cp -r web/control-plane/dist internal/api/controlplane/

# Development build (unminified, better error messages)
frontend-dev:
	@echo "Building frontend (development mode)..."
	cd web/control-plane && npm ci && npm run build:dev
	@echo "Copying frontend assets..."
	rm -rf internal/api/controlplane/dist
	cp -r web/control-plane/dist internal/api/controlplane/

backend:
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
