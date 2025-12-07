/**
 * Frontdoor types and interfaces.
 *
 * @module frontdoors/types
 */

import type { CanonicalRequest, CanonicalResponse, CanonicalEvent } from '../domain/types.js';
import type { Provider } from '../ports/provider.js';
import type { AuthContext } from '../ports/auth.js';
import type { AppConfig } from '../ports/config.js';
import type { StorageProvider } from '../ports/storage.js';
import type { PipelineExecutor } from '../middleware/executor.js';
import type { Logger } from '../utils/logging.js';

// ============================================================================
// Frontdoor Interface
// ============================================================================

/**
 * Context passed to frontdoor handlers.
 */
export interface FrontdoorContext {
    /** The incoming request. */
    request: Request;

    /** The selected provider to use. */
    provider: Provider;

    /** Authenticated context. */
    auth: AuthContext;

    /** App configuration (if matched to an app). */
    app?: AppConfig | undefined;

    /** Storage provider (optional, needed for Responses API). */
    storage?: StorageProvider | undefined;

    /** Logger. */
    logger?: Logger | undefined;

    /** Interaction ID for tracing. */
    interactionId: string;

    /** Pipeline executor for middleware (optional). */
    pipeline?: PipelineExecutor | undefined;
}

/**
 * Response from a frontdoor handler.
 */
export interface FrontdoorResponse {
    /** The HTTP response to send. */
    response: Response;

    /** The canonical request (for logging/storage). */
    canonicalRequest?: CanonicalRequest | undefined;

    /** The canonical response (for non-streaming). */
    canonicalResponse?: CanonicalResponse | undefined;
}

/**
 * A frontdoor handles requests in a specific API format.
 */
export interface Frontdoor {
    /** Frontdoor name. */
    readonly name: string;

    /**
     * Checks if this frontdoor handles the given path.
     */
    matches(path: string): boolean;

    /**
     * Handles a request.
     */
    handle(ctx: FrontdoorContext): Promise<FrontdoorResponse>;
}

// ============================================================================
// Frontdoor Registry
// ============================================================================

/**
 * Registry of frontdoors.
 */
export interface FrontdoorRegistry {
    /**
     * Registers a frontdoor.
     */
    register(frontdoor: Frontdoor): void;

    /**
     * Gets a frontdoor by name.
     */
    get(name: string): Frontdoor | undefined;

    /**
     * Finds a frontdoor that matches the given path.
     */
    match(path: string): Frontdoor | undefined;

    /**
     * Lists registered frontdoor names.
     */
    list(): string[];
}

/**
 * Creates a new frontdoor registry.
 */
export function createFrontdoorRegistry(): FrontdoorRegistry {
    const frontdoors = new Map<string, Frontdoor>();

    return {
        register(frontdoor: Frontdoor): void {
            frontdoors.set(frontdoor.name, frontdoor);
        },

        get(name: string): Frontdoor | undefined {
            return frontdoors.get(name);
        },

        match(path: string): Frontdoor | undefined {
            for (const frontdoor of frontdoors.values()) {
                if (frontdoor.matches(path)) {
                    return frontdoor;
                }
            }
            return undefined;
        },

        list(): string[] {
            return Array.from(frontdoors.keys());
        },
    };
}
