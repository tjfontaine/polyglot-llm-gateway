/**
 * Provider port interface.
 *
 * @module ports/provider
 */

import type {
    APIType,
    CanonicalRequest,
    CanonicalResponse,
    CanonicalEvent,
    ModelList,
} from '../domain/types.js';

// ============================================================================
// Provider Interface
// ============================================================================

/**
 * An LLM provider that can complete requests.
 */
export interface Provider {
    /** Provider name. */
    readonly name: string;

    /** API type this provider uses. */
    readonly apiType: APIType;

    /**
     * Completes a non-streaming request.
     */
    complete(request: CanonicalRequest): Promise<CanonicalResponse>;

    /**
     * Streams a request, yielding events.
     */
    stream(request: CanonicalRequest): AsyncGenerator<CanonicalEvent, void, void>;

    /**
     * Lists available models.
     */
    listModels?(): Promise<ModelList>;
}

// ============================================================================
// Provider Factory Types
// ============================================================================

/**
 * Creates a provider from configuration.
 */
export type ProviderFactory = (config: ProviderFactoryConfig) => Provider;

/**
 * Configuration passed to a provider factory.
 */
export interface ProviderFactoryConfig {
    /** Provider name. */
    name: string;

    /** API key. */
    apiKey: string;

    /** Custom base URL. */
    baseUrl?: string | undefined;

    /** HTTP client override (for testing). */
    fetch?: typeof globalThis.fetch | undefined;

    /** Additional options. */
    options?: Record<string, unknown> | undefined;
}

// ============================================================================
// Provider Registry
// ============================================================================

/**
 * Registry of provider factories.
 */
export interface ProviderRegistry {
    /**
     * Registers a provider factory.
     */
    register(type: string, factory: ProviderFactory): void;

    /**
     * Creates a provider from configuration.
     */
    create(type: string, config: ProviderFactoryConfig): Provider;

    /**
     * Checks if a provider type is registered.
     */
    has(type: string): boolean;

    /**
     * Lists registered provider types.
     */
    list(): string[];
}

/**
 * Creates a new provider registry.
 */
export function createProviderRegistry(): ProviderRegistry {
    const factories = new Map<string, ProviderFactory>();

    return {
        register(type: string, factory: ProviderFactory): void {
            factories.set(type, factory);
        },

        create(type: string, config: ProviderFactoryConfig): Provider {
            const factory = factories.get(type);
            if (!factory) {
                throw new Error(`Unknown provider type: ${type}`);
            }
            return factory(config);
        },

        has(type: string): boolean {
            return factories.has(type);
        },

        list(): string[] {
            return Array.from(factories.keys());
        },
    };
}
