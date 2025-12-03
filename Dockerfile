FROM node:22 AS frontend-builder

# Build mode: 'development' (default) or 'production'
ARG BUILD_MODE=development
ENV NODE_ENV=${BUILD_MODE}

WORKDIR /app/web/control-plane

COPY web/control-plane/package.json web/control-plane/package-lock.json ./
RUN npm ci

# Copy GraphQL schema from Go backend for codegen
COPY internal/api/controlplane/graph/schema.graphqls /app/internal/api/controlplane/graph/schema.graphqls

COPY web/control-plane/ ./
# Generate TypeScript types from GraphQL schema
RUN npm run codegen
# Use 'build:dev' for development (unminified, source maps) or 'build' for production
RUN if [ "$BUILD_MODE" = "production" ]; then npm run build; else npm run build:dev; fi

# Build stage
FROM golang:latest AS builder

WORKDIR /app

# Download dependencies first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate Go GraphQL types using gqlgen
RUN cd internal/api/controlplane/graph && go run github.com/99designs/gqlgen generate

# Copy frontend build and build the v2 gateway binary
RUN rm -rf internal/api/controlplane/dist
COPY --from=frontend-builder /app/web/control-plane/dist internal/api/controlplane/dist
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/gateway ./cmd/gateway-v2

# Runtime stage
FROM debian:bookworm-slim AS runtime

# Install certificates for outbound HTTPS requests and sqlite3 for debugging
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Create an unprivileged user
RUN useradd --system --create-home --uid 1000 gateway

WORKDIR /app
RUN mkdir -p /app/data && chown -R gateway:gateway /app

COPY --from=builder /app/bin/gateway /usr/local/bin/gateway
COPY config.yaml config.example.yaml config-singletenant.yaml config-multitenant.yaml ./

USER gateway
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]
