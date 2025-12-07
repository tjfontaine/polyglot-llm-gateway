/**
 * Authentication provider port.
 *
 * @module ports/auth
 */

import type { ProviderConfig, RoutingConfig } from './config.js';

// ============================================================================
// Auth Types
// ============================================================================

/**
 * Authenticated request context.
 */
export interface AuthContext {
    /** Tenant ID. */
    tenantId: string;

    /** User ID (if available). */
    userId?: string | undefined;

    /** Granted scopes. */
    scopes: string[];

    /** Additional metadata. */
    metadata: Record<string, string>;
}

/**
 * Tenant information.
 */
export interface Tenant {
    /** Tenant ID. */
    id: string;

    /** Tenant name. */
    name: string;

    /** Tenant-specific providers. */
    providers?: Map<string, ProviderConfig> | undefined;

    /** Tenant-specific routing. */
    routing?: RoutingConfig | undefined;
}

// ============================================================================
// AuthProvider Interface
// ============================================================================

/**
 * Provides authentication and tenant lookup.
 * Implementations: KV-based (CF), database-based (Node), etc.
 */
export interface AuthProvider {
    /**
     * Authenticates a request and returns the auth context.
     * Returns null if authentication fails.
     *
     * @param token - The bearer token from the Authorization header
     */
    authenticate(token: string): Promise<AuthContext | null>;

    /**
     * Gets tenant information by ID.
     *
     * @param tenantId - The tenant ID
     */
    getTenant(tenantId: string): Promise<Tenant | null>;
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Extracts a bearer token from an Authorization header.
 * Returns null if the header is missing or malformed.
 */
export function extractBearerToken(
    authorizationHeader: string | null,
): string | null {
    if (!authorizationHeader) return null;

    const parts = authorizationHeader.split(' ');
    if (parts.length !== 2) return null;

    const [scheme, token] = parts;
    if (scheme?.toLowerCase() !== 'bearer') return null;

    return token ?? null;
}

/**
 * Hashes an API key using SHA-256.
 * Uses Web Crypto API for portability.
 */
export async function hashAPIKey(apiKey: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(apiKey);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = new Uint8Array(hashBuffer);
    return Array.from(hashArray)
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('');
}
