/**
 * Cloudflare Workers environment bindings.
 *
 * @module bindings
 */

// ============================================================================
// Environment Bindings
// ============================================================================

/**
 * Cloudflare Workers environment bindings.
 */
export interface Env {
    // KV Namespaces
    CONFIG_KV: KVNamespace;
    AUTH_KV: KVNamespace;

    // D1 Database
    DB: D1Database;

    // Queues
    USAGE_QUEUE?: Queue;

    // R2 Storage
    R2_STORAGE?: R2Bucket;

    // Secrets/Variables
    SIGNING_KEY?: string;
    ENVIRONMENT?: string;
}

// ============================================================================
// KV Key Prefixes
// ============================================================================

export const KV_KEYS = {
    // Config keys
    CONFIG: 'config',
    PROVIDERS: 'providers',
    APPS: 'apps',
    ROUTING: 'routing',

    // Auth keys
    API_KEY_PREFIX: 'api_key:',
    TENANT_PREFIX: 'tenant:',
} as const;

// ============================================================================
// D1 Tables
// ============================================================================

export const D1_TABLES = {
    CONVERSATIONS: 'conversations',
    MESSAGES: 'messages',
    RESPONSES: 'responses',
    INTERACTIONS: 'interactions',
    INTERACTION_EVENTS: 'interaction_events',
    SHADOW_RESULTS: 'shadow_results',
    THREAD_STATE: 'thread_state',
} as const;
