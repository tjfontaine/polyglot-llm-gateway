import { describe, it, expect } from 'vitest';
import { Gateway } from './gateway';
import type { ConfigProvider, GatewayConfig, AppConfig, ProviderConfig } from './ports/config';
import type { AuthProvider, AuthContext } from './ports/auth';
import type { EventPublisher } from './ports/events';

// Mock implementations
class MockConfigProvider implements ConfigProvider {
    private config: GatewayConfig;

    constructor(config: GatewayConfig) {
        this.config = {
            ...config,
            apps: config.apps ?? [],
        };
    }

    async load(): Promise<GatewayConfig> {
        return this.config;
    }

    async getConfig(): Promise<GatewayConfig> {
        return this.config;
    }

    async getApp(name: string): Promise<AppConfig | undefined> {
        return this.config.apps?.find((a) => a.name === name);
    }

    async getProvider(name: string): Promise<ProviderConfig | undefined> {
        return this.config.providers?.find((p) => p.name === name);
    }

    async reload(): Promise<void> { }
}

class MockAuthProvider implements AuthProvider {
    private tenantId: string;

    constructor(tenantId: string = 'test-tenant') {
        this.tenantId = tenantId;
    }

    async authenticate(request: Request): Promise<AuthContext> {
        return {
            tenantId: this.tenantId,
            authenticated: true,
            permissions: ['*'],
        };
    }
}

class MockEventPublisher implements EventPublisher {
    events: unknown[] = [];

    async publish(event: unknown): Promise<void> {
        this.events.push(event);
    }

    async flush(): Promise<void> { }
}

describe('Gateway', () => {
    describe('constructor', () => {
        it('should create gateway with required options', () => {
            const gateway = new Gateway({
                config: new MockConfigProvider({
                    version: '1.0',
                    providers: [{ name: 'openai', apiType: 'openai', apiKey: 'test' }],
                    apps: [],
                }),
                auth: new MockAuthProvider(),
            });

            expect(gateway).toBeDefined();
        });

        it('should create gateway with all options', () => {
            const gateway = new Gateway({
                config: new MockConfigProvider({
                    version: '1.0',
                    providers: [],
                    apps: [],
                }),
                auth: new MockAuthProvider(),
                events: new MockEventPublisher(),
            });

            expect(gateway).toBeDefined();
        });
    });

    describe('health check', () => {
        it('should respond to health check', async () => {
            const gateway = new Gateway({
                config: new MockConfigProvider({
                    version: '1.0',
                    providers: [{ name: 'openai', apiType: 'openai', apiKey: 'test' }],
                    apps: [],
                }),
                auth: new MockAuthProvider(),
            });

            const request = new Request('http://localhost/health');
            const response = await gateway.fetch(request);

            expect(response.status).toBe(200);
            const body = await response.json();
            expect(body.status).toBe('ok');
        });
    });

    describe('authentication', () => {
        it('should reject unauthenticated requests', async () => {
            class FailingAuthProvider implements AuthProvider {
                async authenticate(): Promise<AuthContext> {
                    return {
                        tenantId: '',
                        authenticated: false,
                        error: 'Invalid API key',
                    };
                }
            }

            const gateway = new Gateway({
                config: new MockConfigProvider({
                    version: '1.0',
                    providers: [{ name: 'openai', apiType: 'openai', apiKey: 'test' }],
                    apps: [],
                }),
                auth: new FailingAuthProvider(),
            });

            const request = new Request('http://localhost/v1/chat/completions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model: 'gpt-4', messages: [] }),
            });

            const response = await gateway.fetch(request);

            expect(response.status).toBe(401);
        });
    });
});
