/**
 * Shadow mode executor - executes requests against shadow providers.
 *
 * @module shadow/executor
 */

import type { CanonicalRequest, CanonicalResponse } from '../domain/types.js';
import type { Provider, ProviderRegistry } from '../ports/provider.js';
import type { StorageProvider } from '../ports/storage.js';
import type {
    ShadowResult,
    ShadowConfig,
    ShadowProviderConfig,
    ShadowRequest,
    ShadowResponse,
    ShadowError,
} from '../domain/shadow.js';
import { detectDivergences, createShadowResult } from '../domain/shadow.js';
import type { Logger } from '../utils/logging.js';
import { randomUUID } from '../utils/crypto.js';

// ============================================================================
// Shadow Executor Options
// ============================================================================

/**
 * Options for the shadow executor.
 */
export interface ShadowExecutorOptions {
    /** Provider registry for looking up shadow providers. */
    providerRegistry: ProviderRegistry;

    /** Storage for persisting shadow results. */
    storage?: StorageProvider | undefined;

    /** Logger. */
    logger?: Logger | undefined;

    /** Default timeout in ms. */
    defaultTimeoutMs?: number | undefined;
}

// ============================================================================
// Shadow Executor
// ============================================================================

/**
 * Executes requests against shadow providers and compares responses.
 */
export class ShadowExecutor {
    private readonly providerRegistry: ProviderRegistry;
    private readonly storage?: StorageProvider;
    private readonly logger?: Logger;
    private readonly defaultTimeoutMs: number;

    constructor(options: ShadowExecutorOptions) {
        this.providerRegistry = options.providerRegistry;
        this.storage = options.storage;
        this.logger = options.logger;
        this.defaultTimeoutMs = options.defaultTimeoutMs ?? 30000;
    }

    /**
     * Executes a shadow request for a single provider.
     */
    async executeOne(
        interactionId: string,
        request: CanonicalRequest,
        primaryResponse: CanonicalResponse,
        providerConfig: ShadowProviderConfig,
        timeoutMs: number,
    ): Promise<ShadowResult> {
        const startTime = Date.now();
        const providerName = providerConfig.name;

        // Get shadow provider
        const shadowProvider = this.providerRegistry.create(providerName, {
            name: providerName,
            apiKey: '', // Will need to be injected elsewhere
        });

        // Build shadow request
        const shadowRequest: ShadowRequest = {
            model: providerConfig.model ?? request.model,
            maxTokens: request.maxTokens
                ? Math.floor(request.maxTokens * (providerConfig.maxTokensMultiplier ?? 1))
                : undefined,
        };

        const canonicalShadowRequest: CanonicalRequest = {
            ...request,
            model: shadowRequest.model,
            maxTokens: shadowRequest.maxTokens ?? request.maxTokens,
        };

        try {
            // Execute with timeout
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

            // Make shadow request
            const shadowCanonicalResponse = await shadowProvider.complete(canonicalShadowRequest);
            clearTimeout(timeoutId);

            // Convert to shadow response format
            const shadowResponse = this.toShadowResponse(shadowCanonicalResponse);

            // Create result with divergence detection
            const result = createShadowResult(interactionId, providerName, Date.now() - startTime, {
                request: shadowRequest,
                response: shadowResponse,
                primaryResponse,
            });

            // Persist if storage available
            if (this.storage) {
                await this.storage.saveShadowResult(result);
            }

            // Log divergences
            if (result.divergences.length > 0) {
                this.logger?.warn('Shadow divergences detected', {
                    interactionId,
                    shadowProvider: providerName,
                    divergenceCount: result.divergences.length,
                    hasStructural: result.hasStructuralDivergence,
                });
            }

            return result;
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            const shadowError: ShadowError = {
                type: 'execution_error',
                message: errorMessage,
            };

            this.logger?.error('Shadow request failed', {
                interactionId,
                shadowProvider: providerName,
                error: errorMessage,
            });

            const result = createShadowResult(interactionId, providerName, Date.now() - startTime, {
                request: shadowRequest,
                error: shadowError,
            });

            // Persist error result
            if (this.storage) {
                await this.storage.saveShadowResult(result);
            }

            return result;
        }
    }

    /**
     * Executes shadow requests for all providers in config.
     */
    async executeAll(
        interactionId: string,
        request: CanonicalRequest,
        primaryResponse: CanonicalResponse,
        config: ShadowConfig,
    ): Promise<ShadowResult[]> {
        if (!config.enabled || !config.providers?.length) {
            return [];
        }

        const timeoutMs = this.parseTimeout(config.timeout);

        // Execute all shadow requests in parallel
        const results = await Promise.all(
            config.providers.map((providerConfig) =>
                this.executeOne(interactionId, request, primaryResponse, providerConfig, timeoutMs),
            ),
        );

        return results;
    }

    /**
     * Converts a canonical response to shadow response format.
     */
    private toShadowResponse(response: CanonicalResponse): ShadowResponse {
        const choice = response.choices[0];
        return {
            id: response.id,
            model: response.model,
            content: choice?.message.content ?? '',
            usage: {
                promptTokens: response.usage.promptTokens,
                completionTokens: response.usage.completionTokens,
                totalTokens: response.usage.totalTokens,
            },
            finishReason: choice?.finishReason ?? undefined,
            toolCalls: choice?.message.toolCalls?.map((tc) => ({
                id: tc.id,
                name: tc.function.name,
                arguments: tc.function.arguments,
            })),
        };
    }

    /**
     * Parses a timeout string (e.g., "30s", "1m") to milliseconds.
     */
    private parseTimeout(timeout?: string): number {
        if (!timeout) return this.defaultTimeoutMs;

        const match = timeout.match(/^(\d+)(ms|s|m)$/);
        if (!match) return this.defaultTimeoutMs;

        const value = parseInt(match[1] ?? '0', 10);
        const unit = match[2];

        switch (unit) {
            case 'ms':
                return value;
            case 's':
                return value * 1000;
            case 'm':
                return value * 60 * 1000;
            default:
                return this.defaultTimeoutMs;
        }
    }
}
