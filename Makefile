.PHONY: all build frontend backend clean run

all: build

build: frontend backend

frontend:
	@echo "Building frontend..."
	cd web/control-plane && npm ci && npm run build
	@echo "Copying frontend assets..."
	rm -rf internal/controlplane/dist
	cp -r web/control-plane/dist internal/controlplane/

backend:
	@echo "Building backend..."
	go build -o bin/gateway ./cmd/gateway

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf web/control-plane/dist
	rm -rf web/control-plane/node_modules
	rm -rf internal/controlplane/dist

run: build
	./bin/gateway
