# Polyglot LLM Gateway

A high-performance, self-hosted Go service that acts as a universal translation layer for LLMs. It decouples clients from providers, allowing dynamic routing, failover, and policy enforcement without changing client code.

## Features

- **Multi-Protocol Support**: OpenAI and Anthropic API protocols
- **Multi-Provider Backend**: Route to OpenAI, Anthropic, or any configured provider
- **Configuration-Driven**: Providers, routing, and frontdoors all configurable via YAML
- **Streaming Support**: Server-Sent Events (SSE) for real-time responses
- **Secure Secrets**: Environment variable substitution for API keys
- **Hourglass Architecture**: All traffic flows through a Canonical Intermediate Representation

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd poly-llm-gateway
```

### 2. Configure Environment Variables

Copy the example environment file and add your API keys:

```bash
cp .env.example .env
```

Edit `.env` and add your API keys:

```env
OPENAI_API_KEY=your_openai_api_key_here
ANTHROPIC_API_KEY=your_anthropic_api_key_here
```

### 3. Run the Gateway

```bash
go run cmd/gateway/main.go
```

The server will start on port 8080 by default.

### Run with Docker Compose

Build and start the gateway with Docker Compose (ensure `.env` contains your API keys):

```bash
docker compose up --build
```

The service will be available on http://localhost:8080 and will reload automatically when configuration files or environment variables change on container restarts.

## Configuration

The gateway is configured via `config.yaml`:

```yaml
server:
  port: 8080

apps:
  - name: playground
    frontdoor: openai
    path: /openai
    model_routing:
      prefix_providers:
        openai: openai
        anthropic: anthropic
      rewrites:
        - model_exact: claude-haiku-4.5
          provider: openai
          model: gpt-5-mini
          rewrite_response_model: true
        - model_prefix: claude-sonnet-
          provider: openai
          model: gpt-5-mini
          rewrite_response_model: true
    models:
      - id: claude-haiku-4.5
        object: model
        owned_by: gateway

providers:
  - name: openai
    type: openai
    api_key: ${OPENAI_API_KEY}
  
  - name: anthropic
    type: anthropic
    api_key: ${ANTHROPIC_API_KEY}

routing:
  rules:
    - model_prefix: "claude"
      provider: anthropic
    
    - model_prefix: "gpt"
      provider: openai

  default_provider: openai
```

Each entry under `apps` defines a mounted frontdoor with its own model routing rules. Incoming model names can include a provider
prefix like `openai/gpt-4o` to route directly to the OpenAI provider while passing only `gpt-4o` upstream. The `rewrites` list
lets you alias a model to a different provider and upstream model (e.g., mapping `claude-haiku-4.5` to `gpt-5-mini` on
OpenAI). Set `rewrite_response_model: true` to make responses (and model listings) report the alias instead of the upstream
model. You can also seed the `/v1/models` listing for a frontdoor by providing `models` entries with OpenAI-/Anthropic-compatible fields.

## API Usage

### OpenAI-Compatible Endpoint

```bash
curl -X POST http://localhost:8080/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Anthropic Messages API Endpoint

```bash
curl -X POST http://localhost:8080/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 100
  }'
```

### Streaming Requests

Add `"stream": true` to enable streaming:

```bash
curl -N -X POST http://localhost:8080/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Count to 10"}],
    "stream": true
  }'
```

## Architecture

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│      Frontdoor Adapters         │
│  (OpenAI / Anthropic Protocol)  │
└──────────┬──────────────────────┘
           │
           ▼
    ┌──────────────┐
    │ Canonical IR │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐
    │    Router    │
    └──────┬───────┘
           │
           ▼
┌──────────────────────────────────┐
│      Provider Adapters           │
│  (OpenAI / Anthropic / Custom)   │
└──────────┬───────────────────────┘
           │
           ▼
    ┌──────────────┐
    │  Upstream    │
    │  LLM APIs    │
    └──────────────┘
```

## Project Structure

```
poly-llm-gateway/
├── config.yaml              # Configuration file
├── .env                     # Environment variables (gitignored)
├── .env.example             # Example environment file
├── cmd/gateway/main.go      # Application entrypoint
└── internal/
    ├── config/              # Configuration loading
    ├── domain/              # Canonical types & interfaces
    ├── frontdoor/           # Inbound protocol adapters
    │   ├── openai/          # OpenAI protocol
    │   └── anthropic/       # Anthropic protocol
    ├── provider/            # Outbound provider adapters
    │   ├── openai/          # OpenAI provider
    │   └── anthropic/       # Anthropic provider
    ├── policy/              # Routing logic
    └── server/              # HTTP server
```

## Development

### Build

```bash
go build ./...
```

### Run Tests

```bash
go test ./...
```

## Use Cases

- **API Consolidation**: Single endpoint for multiple LLM providers
- **Protocol Migration**: Migrate between providers without changing client code
- **A/B Testing**: Route different models to different providers
- **Cost Optimization**: Route to cheapest provider based on model
- **Failover**: Automatic fallback to backup providers

## License

[Add your license here]
