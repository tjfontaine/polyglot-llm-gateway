# @polyglot-llm-gateway/gateway-core

Runtime-agnostic LLM gateway library for proxying and transforming requests between different LLM providers (OpenAI, Anthropic). Built with Web Standard APIs for portability across Cloudflare Workers, Node.js, Deno, and Bun.

## Features

- **Multi-provider support** - OpenAI and Anthropic with consistent API surface
- **Bidirectional translation** - Convert between OpenAI and Anthropic formats
- **Streaming support** - Full SSE streaming with event transformation
- **Middleware pipeline** - Transform requests/responses with webhooks, filters, logging
- **Responses API** - OpenAI Responses API with threading support
- **Shadow mode** - Compare responses from multiple providers
- **Pluggable adapters** - Runtime-specific storage, auth, and config

## Installation

```bash
npm install @polyglot-llm-gateway/gateway-core
# or
pnpm add @polyglot-llm-gateway/gateway-core
```

## Quick Start

### Cloudflare Workers

```typescript
import { Gateway } from '@polyglot-llm-gateway/gateway-core';
import {
  KVConfigProvider,
  KVAuthProvider,
  D1StorageProvider,
} from '@polyglot-llm-gateway/gateway-adapter-cloudflare';

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const gateway = new Gateway({
      config: new KVConfigProvider(env.CONFIG_KV),
      auth: new KVAuthProvider(env.AUTH_KV),
      storage: new D1StorageProvider(env.DB),
    });
    
    return gateway.fetch(request);
  },
};
```

### Node.js

```typescript
import { createServer } from 'http';
import { Gateway } from '@polyglot-llm-gateway/gateway-core';
import {
  EnvConfigProvider,
  StaticAuthProvider,
  MemoryStorageProvider,
} from '@polyglot-llm-gateway/gateway-adapter-node';

const gateway = new Gateway({
  config: new EnvConfigProvider(),
  auth: new StaticAuthProvider({ 'sk-test': { tenantId: 'demo' } }),
  storage: new MemoryStorageProvider(),
});

createServer(async (req, res) => {
  const request = /* convert to Web Request */;
  const response = await gateway.fetch(request);
  /* convert Web Response back */
}).listen(8080);
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Gateway                              │
├─────────────────────────────────────────────────────────────┤
│  Request → Auth → Router → Frontdoor → Provider → Response  │
│                      ↓                                       │
│              Middleware Pipeline                             │
│              (pre/post hooks)                                │
└─────────────────────────────────────────────────────────────┘
```

### Components

| Component | Purpose |
|-----------|---------|
| **Gateway** | Main entry point, orchestrates request handling |
| **Router** | Matches apps and selects providers based on model |
| **Frontdoors** | API format handlers (OpenAI, Anthropic, Responses) |
| **Codecs** | Bidirectional translation between API formats |
| **Providers** | LLM API clients with streaming support |
| **Middleware** | Request/response transformation pipeline |

## Configuration

```typescript
interface GatewayConfig {
  version: string;
  
  providers: ProviderConfig[];  // Provider credentials
  apps?: AppConfig[];           // App-specific routing
  routing?: RoutingConfig;      // Global model routing
}

interface ProviderConfig {
  name: string;
  apiType: 'openai' | 'anthropic';
  apiKey: string;
  baseUrl?: string;
}

interface AppConfig {
  name: string;
  frontdoor: 'openai' | 'anthropic' | 'responses';
  path: string;
  provider?: string;  // Force specific provider
  modelRouting?: ModelRoutingConfig;
  shadow?: ShadowConfig;
}
```

## Middleware

```typescript
import { 
  PipelineExecutor, 
  createWebhookStep,
  createContentFilterStep,
} from '@polyglot-llm-gateway/gateway-core';

const executor = new PipelineExecutor();

// Add pre-request webhook
executor.addPreStage({
  name: 'auth-webhook',
  type: 'pre',
  step: createWebhookStep({
    type: 'webhook',
    url: 'https://api.example.com/validate',
    timeoutMs: 5000,
  }),
});

// Add content filter
executor.addPreStage({
  name: 'pii-filter',
  type: 'pre',
  step: createContentFilterStep({
    type: 'content_filter',
    blockPatterns: ['\\b\\d{3}-\\d{2}-\\d{4}\\b'], // SSN
    blockAction: 'deny',
  }),
});
```

## Shadow Mode

Compare responses from multiple providers:

```typescript
import { ShadowManager } from '@polyglot-llm-gateway/gateway-core';

const shadowManager = new ShadowManager({
  providerRegistry,
  storage,
  samplingRate: 0.1, // 10% of requests
  defaultConfig: {
    enabled: true,
    providers: [
      { name: 'anthropic', model: 'claude-3-sonnet-20240229' }
    ],
    timeout: '30s',
  },
});

// Execute shadow asynchronously
shadowManager.executeAsync(interactionId, request, primaryResponse, app, (results) => {
  // Results contain divergences
  for (const result of results) {
    if (result.divergences.length > 0) {
      console.log('Divergences detected:', result.divergences);
    }
  }
});
```

## Responses API

Full OpenAI Responses API support with threading:

```typescript
// Create response
const response = await handler.handle({
  model: 'gpt-4',
  input: [{ type: 'message', role: 'user', content: 'Hello!' }],
}, tenantId);

// Thread continuation
const followUp = await handler.handle({
  model: 'gpt-4',
  input: [{ type: 'message', role: 'user', content: 'Tell me more' }],
  previousResponseId: response.id,
}, tenantId);
```

## Port Interfaces

Implement these interfaces for custom runtimes:

```typescript
interface ConfigProvider {
  load(): Promise<GatewayConfig>;
  getApp(name: string): Promise<AppConfig | undefined>;
  getProvider(name: string): Promise<ProviderConfig | undefined>;
}

interface AuthProvider {
  authenticate(request: Request): Promise<AuthContext>;
}

interface StorageProvider {
  // Conversations, responses, events, shadows, threads
  saveConversation(c: Conversation): Promise<void>;
  getConversation(id: string): Promise<Conversation | null>;
  // ... other methods
}

interface EventPublisher {
  publish(event: LifecycleEvent): Promise<void>;
  flush(): Promise<void>;
}
```

## Testing

```bash
pnpm test
```

## License

MIT
