/**
 * Integration tests for the Gateway.
 *
 * These tests verify end-to-end request handling.
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { Gateway } from '../gateway.js';
import type { ConfigProvider, GatewayConfig } from '../ports/config.js';
import type { AuthProvider, AuthContext } from '../ports/auth.js';

// ============================================================================
// Test Fixtures
// ============================================================================

const mockConfig: GatewayConfig = {
    apps: [
        {
            name: 'test-app',
            frontdoor: 'openai',
            path: '/v1',
            provider: 'openai',
        },
    ],
    providers: [
        {
            name: 'openai',
            type: 'openai',
            apiKey: 'test-key',
        },
    ],
    routing: {
        defaultProvider: 'openai',
        rules: [{ modelPrefix: 'gpt', provider: 'openai' }],
    },
};

class MockConfigProvider implements ConfigProvider {
    private config: GatewayConfig;

    constructor(config: GatewayConfig = mockConfig) {
        this.config = config;
    }

    async load(): Promise<GatewayConfig> {
        return this.config;
    }

    setConfig(config: GatewayConfig): void {
        this.config = config;
    }
}

class MockAuthProvider implements AuthProvider {
    private validTokens: Map<string, AuthContext> = new Map();

    constructor() {
        // Default valid token
        this.validTokens.set('test-token', {
            tenantId: 'test-tenant',
            userId: 'test-user',
            scopes: ['*'],
        });
    }

    async authenticate(token: string): Promise<AuthContext | null> {
        return this.validTokens.get(token) ?? null;
    }

    addToken(token: string, context: AuthContext): void {
        this.validTokens.set(token, context);
    }
}

// ============================================================================
// Health Check Tests
// ============================================================================

describe('Gateway Integration', () => {
    let configProvider: MockConfigProvider;
    let authProvider: MockAuthProvider;
    let gateway: Gateway;

    beforeEach(() => {
        configProvider = new MockConfigProvider();
        authProvider = new MockAuthProvider();
        gateway = new Gateway({
            config: configProvider,
            auth: authProvider,
        });
    });

    describe('health check', () => {
        it('should respond to /health without authentication', async () => {
            const request = new Request('http://localhost/health', {
                method: 'GET',
            });

            const response = await gateway.fetch(request);
            expect(response.status).toBe(200);

            const body = await response.json();
            expect(body).toEqual({ status: 'ok' });
        });

        it('should respond to /healthz without authentication', async () => {
            const request = new Request('http://localhost/healthz', {
                method: 'GET',
            });

            const response = await gateway.fetch(request);
            expect(response.status).toBe(200);

            const body = await response.json();
            expect(body).toEqual({ status: 'ok' });
        });
    });

    describe('authentication', () => {
        it('should reject requests without Authorization header', async () => {
            const request = new Request('http://localhost/v1/chat/completions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model: 'gpt-4', messages: [] }),
            });

            const response = await gateway.fetch(request);
            expect(response.status).toBe(401);
        });

        it('should reject requests with invalid token', async () => {
            const request = new Request('http://localhost/v1/chat/completions', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer invalid-token',
                },
                body: JSON.stringify({ model: 'gpt-4', messages: [] }),
            });

            const response = await gateway.fetch(request);
            expect(response.status).toBe(401);
        });
    });

    describe('routing', () => {
        it('should match app by path', async () => {
            // Force reload to load config
            await gateway.reload();

            // Access app at configured path
            const request = new Request('http://localhost/v1/chat/completions', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer test-token',
                },
                body: JSON.stringify({ model: 'gpt-4', messages: [] }),
            });

            const response = await gateway.fetch(request);
            // Will fail because provider doesn't actually exist, but routing works
            // We're testing that we don't get a 404
            expect(response.status).not.toBe(404);
        });
    });

    describe('reload', () => {
        it('should reload configuration', async () => {
            await gateway.reload();

            // Update config
            configProvider.setConfig({
                ...mockConfig,
                apps: [
                    ...mockConfig.apps,
                    {
                        name: 'new-app',
                        frontdoor: 'anthropic',
                        path: '/anthropic',
                    },
                ],
            });

            await gateway.reload();

            // New app should be available
            const request = new Request('http://localhost/anthropic/messages', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer test-token',
                },
                body: JSON.stringify({ model: 'claude-3', messages: [] }),
            });

            const response = await gateway.fetch(request);
            // Not 404 means the route was found
            expect(response.status).not.toBe(404);
        });
    });
});
