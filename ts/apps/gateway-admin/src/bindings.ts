/**
 * Cloudflare Workers environment bindings for gateway-admin.
 */

export interface Env {
    /** KV namespace for configuration. */
    CONFIG_KV: KVNamespace;

    /** KV namespace for auth. */
    AUTH_KV: KVNamespace;

    /** D1 database for storage. */
    DB: D1Database;

    /** Environment name. */
    ENVIRONMENT: string;
}
