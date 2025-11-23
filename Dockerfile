# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

# Download dependencies first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build the gateway binary
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/gateway ./cmd/gateway

# Runtime stage
FROM debian:bookworm-slim AS runtime

# Install certificates for outbound HTTPS requests
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates \
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
