# Polyglot LLM Gateway

A production-ready, **extensible LLM gateway** with pluggable architecture for multi-provider support, multi-tenancy, and advanced request routing.

## ✨ Features

- **Multi-Provider**: OpenAI, Anthropic, and custom providers
- **Multi-Tenant**: API key authentication with per-tenant routing
- **Pluggable Architecture**: Swap any component (config, auth, storage, events)
- **Hot-Reload**: Config changes without restart
- **Shadow Mode**: Run experimental providers in parallel for testing
- **Responses API**: OpenAI-compatible responses for multi-turn interactions
- **Runtime Agnostic**: Works on Node.js, Cloudflare Workers, Deno, Bun
- **GraphQL API**: Built-in GraphQL endpoint with GraphiQL playground

---

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/tjfontaine/polyglot-llm-gateway
cd polyglot-llm-gateway/ts
pnpm install
pnpm build
```

### Run with Config File

```bash
# Create config file
cp ../config.example.yaml apps/gateway-node/config.yaml

# Set your API keys
export OPENAI_API_KEY=your-key
export ANTHROPIC_API_KEY=your-key

# Run the gateway
cd apps/gateway-node
pnpm dev
```

### Run with Environment Variables Only

```bash
cd apps/gateway-node
OPENAI_API_KEY=your-key pnpm dev
```

The gateway starts on **port 8080** with:

- Health check at <http://localhost:8080/health>
- OpenAI API at <http://localhost:8080/v1/>*
- Default dev API key: `dev-api-key`

---

## 📦 Configuration

`config.yaml`:

```yaml
server:
  port: 8080

providers:
  - name: openai
    type: openai
    api_key: ${OPENAI_API_KEY}
  
  - name: anthropic
    type: anthropic
    api_key: ${ANTHROPIC_API_KEY}

apps:
  - name: openai-api
    frontdoor: openai
    path: /v1

  - name: anthropic-api
    frontdoor: anthropic
    path: /anthropic

routing:
  default_provider: openai
  rules:
    - model_prefix: claude
      provider: anthropic
    - model_prefix: gpt
      provider: openai
```

---

## 🏗️ Architecture

The gateway uses a **pluggable adapter pattern** with TypeScript interfaces:

```
┌─────────────────────────────────────────────────────────────┐
│                    Gateway (core)                            │
├─────────────┬─────────────┬─────────────┬──────────────────┤
│ ConfigProvider │ AuthProvider │ StorageProvider │ EventPublisher │
├─────────────┼─────────────┼─────────────┼──────────────────┤
│ File (YAML)  │ API Key     │ Memory      │ Null            │  ← Node.js
│ KV Store     │ KV Store    │ D1/SQLite   │ Queue           │  ← Cloudflare
└─────────────┴─────────────┴─────────────┴──────────────────┘
```

### Packages

| Package | Description |
|---------|-------------|
| `gateway-core` | Runtime-agnostic core library using Web standards |
| `gateway-adapter-node` | Node.js adapters (FileConfig, Memory, etc.) |
| `gateway-adapter-cloudflare` | Cloudflare adapters (KV, D1, Queues) |
| `gateway-node` | Deployable Node.js gateway |
| `gateway-cloudflare` | Deployable Cloudflare Workers gateway |

---

## 🐳 Docker

```bash
# Build
docker build -t llm-gateway .

# Run
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=your-key \
  -v $(pwd)/config.yaml:/app/apps/gateway-node/config.yaml \
  llm-gateway
```

Or with docker-compose:

```bash
docker-compose up
```

---

## 📦 Using as a Library

Embed the gateway in your Node.js application:

```typescript
import { Gateway } from '@polyglot-llm-gateway/gateway-core';
import {
  FileConfigProvider,
  StaticAuthProvider,
  MemoryStorageProvider,
} from '@polyglot-llm-gateway/gateway-adapter-node';

const gateway = new Gateway({
  config: new FileConfigProvider({ path: 'config.yaml' }),
  auth: createAuthProvider(),
  storage: new MemoryStorageProvider(),
});

await gateway.reload();
await gateway.startWatching(); // Hot reload on config change

// Use with any HTTP server
const response = await gateway.fetch(request);
```

---

## 🔧 API Endpoints

### Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### OpenAI Chat Completions

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Anthropic Messages

```bash
curl http://localhost:8080/anthropic/messages \
  -H "Authorization: Bearer dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-sonnet-20240229",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## 🛠️ Development

```bash
cd ts

# Install dependencies
pnpm install

# Build all packages
pnpm build

# Run tests
pnpm -r test

# Dev mode with hot reload
cd apps/gateway-node
pnpm dev
```

---

## 📝 License

See [LICENSE](LICENSE) file.
