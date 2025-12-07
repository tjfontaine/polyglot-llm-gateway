/**
 * KV-based authentication provider for Cloudflare Workers.
 *
 * @module adapters/auth
 */

import type { AuthProvider, AuthContext, Tenant, ProviderConfig } from '@polyglot-llm-gateway/gateway-core';
import { sha256 } from '@polyglot-llm-gateway/gateway-core';
import { KV_KEYS } from '../bindings.js';

// ============================================================================
// KV Auth Provider
// ============================================================================

/**
 * Authentication provider backed by Cloudflare KV.
 */
export class KVAuthProvider implements AuthProvider {
    constructor(private readonly kv: KVNamespace) { }

    async authenticate(token: string): Promise<AuthContext | null> {
        // Hash the token to lookup
        const tokenHash = await sha256(token);
        const key = `${KV_KEYS.API_KEY_PREFIX}${tokenHash}`;

        // Lookup in KV
        const authData = await this.kv.get(key, 'json') as StoredAuthContext | null;
        if (!authData) {
            return null;
        }

        return {
            tenantId: authData.tenantId,
            userId: authData.userId,
            scopes: authData.scopes ?? [],
            metadata: authData.metadata ?? {},
        };
    }

    async getTenant(tenantId: string): Promise<Tenant | null> {
        const key = `${KV_KEYS.TENANT_PREFIX}${tenantId}`;
        const tenantData = await this.kv.get(key, 'json') as StoredTenant | null;

        if (!tenantData) {
            return null;
        }

        return {
            id: tenantData.id,
            name: tenantData.name,
            providers: tenantData.providers
                ? new Map(Object.entries(tenantData.providers))
                : undefined,
            routing: tenantData.routing,
        };
    }
}

// ============================================================================
// Static Auth Provider
// ============================================================================

/**
 * Simple authentication provider for development.
 * Uses a map of API keys to auth contexts.
 */
export class StaticAuthProvider implements AuthProvider {
    private readonly apiKeys: Map<string, AuthContext> = new Map();
    private readonly tenants: Map<string, Tenant> = new Map();

    /**
     * Adds an API key.
     */
    addApiKey(apiKey: string, context: AuthContext): void {
        this.apiKeys.set(apiKey, context);
    }

    /**
     * Adds a tenant.
     */
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

/**
 * Creates a development auth provider with a single passthrough key.
 */
export function createDevAuthProvider(
    apiKey: string = 'dev-api-key',
    tenantId: string = 'dev-tenant',
): StaticAuthProvider {
    const provider = new StaticAuthProvider();

    provider.addApiKey(apiKey, {
        tenantId,
        scopes: ['*'],
        metadata: {},
    });

    provider.addTenant({
        id: tenantId,
        name: 'Development',
    });

    return provider;
}

// ============================================================================
// Internal Types
// ============================================================================

interface StoredAuthContext {
    tenantId: string;
    userId?: string;
    scopes?: string[];
    metadata?: Record<string, string>;
}

interface StoredTenant {
    id: string;
    name: string;
    providers?: Record<string, ProviderConfig>;
    routing?: {
        rules?: Array<{ modelPrefix?: string; modelExact?: string; provider: string }>;
        defaultProvider?: string;
    };
}
