/**
 * Node.js gateway entrypoint.
 */

import { existsSync } from 'node:fs';
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { Gateway, type ConfigProvider } from '@polyglot-llm-gateway/gateway-core';
import {
    FileConfigProvider,
    EnvConfigProvider,
    StaticAuthProvider,
    MemoryStorageProvider,
    NullEventPublisher,
} from '@polyglot-llm-gateway/gateway-adapter-node';

const PORT = parseInt(process.env['PORT'] ?? '8080', 10);
const CONFIG_PATH = process.env['CONFIG_PATH'] ?? 'config.yaml';

// Choose config provider based on what's available
function createConfigProvider(): ConfigProvider {
    if (existsSync(CONFIG_PATH)) {
        console.log(`Using file config: ${CONFIG_PATH}`);
        return new FileConfigProvider({ path: CONFIG_PATH });
    }
    console.log('Using environment config');
    return new EnvConfigProvider(process.env as Record<string, string>);
}

// Create gateway
const gateway = new Gateway({
    config: createConfigProvider(),
    auth: createAuthProvider(),
    storage: new MemoryStorageProvider(),
    events: new NullEventPublisher(),
});

// Load configuration
await gateway.reload();

// Start watching for config changes (if supported)
await gateway.startWatching();

// Create HTTP server
const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
    try {
        // Convert Node request to Web Request
        const url = `http://${req.headers.host}${req.url}`;
        const headers = new Headers();
        for (const [key, value] of Object.entries(req.headers)) {
            if (value) {
                headers.set(key, Array.isArray(value) ? value.join(', ') : value);
            }
        }

        let body: ArrayBuffer | undefined;
        if (req.method !== 'GET' && req.method !== 'HEAD') {
            body = await readBody(req);
        }

        const webRequest = new Request(url, {
            method: req.method,
            headers,
            body,
        });

        // Handle request
        const webResponse = await gateway.fetch(webRequest);

        // Convert Web Response to Node response
        res.statusCode = webResponse.status;
        webResponse.headers.forEach((value, key) => {
            res.setHeader(key, value);
        });

        if (webResponse.body) {
            const reader = webResponse.body.getReader();
            try {
                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;
                    res.write(value);
                }
            } finally {
                reader.releaseLock();
            }
        }

        res.end();
    } catch (error) {
        console.error('Request error:', error);
        res.statusCode = 500;
        res.setHeader('Content-Type', 'application/json');
        res.end(JSON.stringify({ error: { message: 'Internal server error' } }));
    }
});

server.listen(PORT, () => {
    console.log(`Gateway listening on http://localhost:${PORT}`);
});

// Utility functions

async function readBody(req: IncomingMessage): Promise<ArrayBuffer> {
    const chunks: Buffer[] = [];
    for await (const chunk of req) {
        chunks.push(chunk);
    }
    const buffer = Buffer.concat(chunks);
    return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
}

function createAuthProvider(): StaticAuthProvider {
    const provider = new StaticAuthProvider();

    // Add development API key
    const devKey = process.env['DEV_API_KEY'] ?? 'dev-api-key';
    provider.addApiKey(devKey, {
        tenantId: 'dev-tenant',
        scopes: ['*'],
        metadata: {},
    });

    provider.addTenant({
        id: 'dev-tenant',
        name: 'Development',
    });

    return provider;
}
