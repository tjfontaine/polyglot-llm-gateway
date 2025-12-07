/**
 * Node.js adapter for polyglot-llm-gateway.
 *
 * @module index
 */

// File-based config provider
export { FileConfigProvider, type FileConfigProviderOptions } from './config.js';

import type {
    ConfigProvider,
    GatewayConfig,
    AuthProvider,
    AuthContext,
    Tenant,
    StorageProvider,
    EventPublisher,
    LifecycleEvent,
    ProviderConfig,
    Conversation,
    ResponseRecord,
    InteractionSummary,
    InteractionEvent,
    ShadowResult,
    ListOptions,
    InteractionListOptions,
    DivergenceListOptions,
} from '@polyglot-llm-gateway/gateway-core';

// ============================================================================
// Environment Config Provider
// ============================================================================

/**
 * Configuration provider that reads from environment variables.
 */
export class EnvConfigProvider implements ConfigProvider {
    constructor(private readonly env: Record<string, string | undefined>) { }

    async load(): Promise<GatewayConfig> {
        const configJson = this.env['GATEWAY_CONFIG'];
        if (configJson) {
            return JSON.parse(configJson) as GatewayConfig;
        }

        // Build minimal config from individual env vars
        const providers: GatewayConfig['providers'] = [];

        // OpenAI
        if (this.env['OPENAI_API_KEY']) {
            providers.push({
                name: 'openai',
                type: 'openai',
                apiKey: this.env['OPENAI_API_KEY'],
                baseUrl: this.env['OPENAI_BASE_URL'],
            });
        }

        // Anthropic
        if (this.env['ANTHROPIC_API_KEY']) {
            providers.push({
                name: 'anthropic',
                type: 'anthropic',
                apiKey: this.env['ANTHROPIC_API_KEY'],
                baseUrl: this.env['ANTHROPIC_BASE_URL'],
            });
        }

        return {
            providers,
            apps: [
                {
                    name: 'default',
                    frontdoor: 'openai',
                    path: '/v1',
                },
            ],
            routing: {
                defaultProvider: 'openai',
            },
        };
    }
}

// ============================================================================
// Static Auth Provider
// ============================================================================

/**
 * Simple authentication provider for development.
 */
export class StaticAuthProvider implements AuthProvider {
    private readonly apiKeys: Map<string, AuthContext> = new Map();
    private readonly tenants: Map<string, Tenant> = new Map();

    addApiKey(apiKey: string, context: AuthContext): void {
        this.apiKeys.set(apiKey, context);
    }

    addTenant(tenant: Tenant): void {
        this.tenants.set(tenant.id, tenant);
    }

    async authenticate(token: string): Promise<AuthContext | null> {
        return this.apiKeys.get(token) ?? null;
    }

    async getTenant(tenantId: string): Promise<Tenant | null> {
        return this.tenants.get(tenantId) ?? null;
    }
}

// ============================================================================
// Memory Storage Provider
// ============================================================================

/**
 * In-memory storage provider for development.
 */
export class MemoryStorageProvider implements StorageProvider {
    private readonly conversations = new Map<string, Conversation>();
    private readonly responses = new Map<string, ResponseRecord>();
    private readonly events = new Map<string, InteractionEvent[]>();
    private readonly shadowResults = new Map<string, ShadowResult[]>();
    private readonly threadState = new Map<string, string>();

    // Conversations
    async saveConversation(conversation: Conversation): Promise<void> {
        this.conversations.set(conversation.id, conversation);
    }

    async getConversation(id: string): Promise<Conversation | null> {
        return this.conversations.get(id) ?? null;
    }

    async listConversations(tenantId: string, options?: ListOptions): Promise<Conversation[]> {
        const limit = options?.limit ?? 50;
        return Array.from(this.conversations.values())
            .filter((c) => c.tenantId === tenantId)
            .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime())
            .slice(0, limit);
    }

    // Responses
    async saveResponse(response: ResponseRecord): Promise<void> {
        this.responses.set(response.id, response);
    }

    async getResponse(id: string): Promise<ResponseRecord | null> {
        return this.responses.get(id) ?? null;
    }

    async listResponses(tenantId: string, options?: ListOptions): Promise<ResponseRecord[]> {
        const limit = options?.limit ?? 50;
        return Array.from(this.responses.values())
            .filter((r) => r.tenantId === tenantId)
            .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime())
            .slice(0, limit);
    }

    // Interactions
    async listInteractions(options?: InteractionListOptions): Promise<InteractionSummary[]> {
        const limit = options?.limit ?? 50;
        const all: InteractionSummary[] = [];

        for (const c of this.conversations.values()) {
            if (options?.tenantId && c.tenantId !== options.tenantId) continue;
            all.push({
                id: c.id,
                type: 'conversation',
                tenantId: c.tenantId,
                appName: c.appName,
                model: c.model,
                messageCount: c.messages.length,
                createdAt: c.createdAt,
                updatedAt: c.updatedAt,
            });
        }

        for (const r of this.responses.values()) {
            if (options?.tenantId && r.tenantId !== options.tenantId) continue;
            all.push({
                id: r.id,
                type: 'response',
                tenantId: r.tenantId,
                appName: r.appName,
                model: r.model,
                status: r.status,
                createdAt: r.createdAt,
                updatedAt: r.updatedAt,
            });
        }

        return all
            .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime())
            .slice(0, limit);
    }

    async getInteractionCount(options?: InteractionListOptions): Promise<number> {
        let count = 0;
        for (const c of this.conversations.values()) {
            if (!options?.tenantId || c.tenantId === options.tenantId) count++;
        }
        for (const r of this.responses.values()) {
            if (!options?.tenantId || r.tenantId === options.tenantId) count++;
        }
        return count;
    }

    async saveEvent(event: InteractionEvent): Promise<void> {
        const existing = this.events.get(event.interactionId) ?? [];
        existing.push(event);
        this.events.set(event.interactionId, existing);
    }

    async getEvents(interactionId: string): Promise<InteractionEvent[]> {
        return this.events.get(interactionId) ?? [];
    }

    // Shadow Results
    async saveShadowResult(result: ShadowResult): Promise<void> {
        const existing = this.shadowResults.get(result.interactionId) ?? [];
        existing.push(result);
        this.shadowResults.set(result.interactionId, existing);
    }

    async getShadowResults(interactionId: string): Promise<ShadowResult[]> {
        return this.shadowResults.get(interactionId) ?? [];
    }

    async getShadowResult(id: string): Promise<ShadowResult | null> {
        for (const results of this.shadowResults.values()) {
            const result = results.find((r) => r.id === id);
            if (result) return result;
        }
        return null;
    }

    async listDivergentShadowResults(options?: DivergenceListOptions): Promise<ShadowResult[]> {
        const limit = options?.limit ?? 50;
        const all: ShadowResult[] = [];

        for (const results of this.shadowResults.values()) {
            for (const result of results) {
                if (options?.structuralOnly && !result.hasStructuralDivergence) continue;
                all.push(result);
            }
        }

        return all
            .sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime())
            .slice(0, limit);
    }

    // Thread State
    async setThreadState(threadKey: string, responseId: string): Promise<void> {
        this.threadState.set(threadKey, responseId);
    }

    async getThreadState(threadKey: string): Promise<string | null> {
        return this.threadState.get(threadKey) ?? null;
    }
}

// ============================================================================
// Null Event Publisher
// ============================================================================

/**
 * No-op event publisher.
 */
export class NullEventPublisher implements EventPublisher {
    async publish(_event: LifecycleEvent): Promise<void> {
        // No-op
    }
}
