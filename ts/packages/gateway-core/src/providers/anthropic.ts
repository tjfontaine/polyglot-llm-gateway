/**
 * Anthropic provider - makes requests to the Anthropic API.
 *
 * @module providers/anthropic
 */

import type {
    CanonicalRequest,
    CanonicalResponse,
    CanonicalEvent,
    ModelList,
    APIType,
} from '../domain/types.js';
import { APIError, errServer } from '../domain/errors.js';
import type { Provider, ProviderFactoryConfig } from '../ports/provider.js';
import { AnthropicCodec } from '../codecs/anthropic.js';

// ============================================================================
// Constants
// ============================================================================

const DEFAULT_BASE_URL = 'https://api.anthropic.com';
const MESSAGES_PATH = '/v1/messages';
const API_VERSION = '2023-06-01';

// ============================================================================
// Anthropic Provider
// ============================================================================

/**
 * Anthropic provider implementation.
 */
export class AnthropicProvider implements Provider {
    readonly name: string;
    readonly apiType: APIType = 'anthropic';

    private readonly apiKey: string;
    private readonly baseUrl: string;
    private readonly codec: AnthropicCodec;
    private readonly fetchFn: typeof fetch;

    constructor(config: ProviderFactoryConfig) {
        this.name = config.name;
        this.apiKey = config.apiKey;
        this.baseUrl = (config.baseUrl ?? DEFAULT_BASE_URL).replace(/\/$/, '');
        this.codec = new AnthropicCodec();
        this.fetchFn = config.fetch ?? globalThis.fetch.bind(globalThis);
    }

    /**
     * Makes a non-streaming completion request.
     */
    async complete(request: CanonicalRequest): Promise<CanonicalResponse> {
        const body = this.codec.encodeRequest({ ...request, stream: false });

        const response = await this.fetchFn(`${this.baseUrl}${MESSAGES_PATH}`, {
            method: 'POST',
            headers: this.getHeaders(request),
            body,
        });

        const responseBody = await response.arrayBuffer();
        const responseBytes = new Uint8Array(responseBody);

        if (!response.ok) {
            throw this.codec.decodeError(responseBytes, response.status);
        }

        const canonicalResponse = this.codec.decodeResponse(responseBytes);
        canonicalResponse.sourceAPIType = 'anthropic';

        // Extract rate limits from headers
        canonicalResponse.rateLimits = this.extractRateLimits(response.headers);

        return canonicalResponse;
    }

    /**
     * Makes a streaming completion request.
     */
    async *stream(request: CanonicalRequest): AsyncGenerator<CanonicalEvent, void, void> {
        const body = this.codec.encodeRequest({ ...request, stream: true });

        const response = await this.fetchFn(`${this.baseUrl}${MESSAGES_PATH}`, {
            method: 'POST',
            headers: this.getHeaders(request),
            body,
        });

        if (!response.ok) {
            const responseBody = await response.arrayBuffer();
            throw this.codec.decodeError(new Uint8Array(responseBody), response.status);
        }

        if (!response.body) {
            throw errServer('No response body for streaming request');
        }

        // Parse SSE stream
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let currentEventType = '';

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });

                // Process complete lines
                const lines = buffer.split('\n');
                buffer = lines.pop() ?? ''; // Keep incomplete line in buffer

                for (const line of lines) {
                    const trimmed = line.trim();

                    // Skip empty lines
                    if (!trimmed) {
                        currentEventType = '';
                        continue;
                    }

                    // Parse event type
                    if (trimmed.startsWith('event: ')) {
                        currentEventType = trimmed.slice(7);
                        continue;
                    }

                    // Parse data
                    if (trimmed.startsWith('data: ')) {
                        const data = trimmed.slice(6);

                        // Decode the chunk
                        const event = this.codec.decodeStreamChunk(data);
                        if (event) {
                            yield event;

                            // Check for message_stop
                            if (event.type === 'message_stop') {
                                yield { type: 'done' };
                                return;
                            }
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    }

    /**
     * Lists available models.
     * Note: Anthropic doesn't have a public models endpoint, so we return known models.
     */
    async listModels(): Promise<ModelList> {
        // Anthropic doesn't expose a models API, return known models
        return {
            object: 'list',
            data: [
                { id: 'claude-3-5-sonnet-20241022', ownedBy: 'anthropic' },
                { id: 'claude-3-5-haiku-20241022', ownedBy: 'anthropic' },
                { id: 'claude-3-opus-20240229', ownedBy: 'anthropic' },
                { id: 'claude-3-sonnet-20240229', ownedBy: 'anthropic' },
                { id: 'claude-3-haiku-20240307', ownedBy: 'anthropic' },
            ],
        };
    }

    /**
     * Gets request headers.
     */
    private getHeaders(request: CanonicalRequest): Record<string, string> {
        const headers: Record<string, string> = {
            'x-api-key': this.apiKey,
            'anthropic-version': API_VERSION,
            'Content-Type': 'application/json',
        };

        if (request.userAgent) {
            headers['User-Agent'] = request.userAgent;
        }

        return headers;
    }

    /**
     * Extracts rate limit info from response headers.
     */
    private extractRateLimits(headers: Headers): CanonicalResponse['rateLimits'] {
        const requestsLimit = headers.get('anthropic-ratelimit-requests-limit');
        const requestsRemaining = headers.get('anthropic-ratelimit-requests-remaining');
        const requestsReset = headers.get('anthropic-ratelimit-requests-reset');
        const tokensLimit = headers.get('anthropic-ratelimit-tokens-limit');
        const tokensRemaining = headers.get('anthropic-ratelimit-tokens-remaining');
        const tokensReset = headers.get('anthropic-ratelimit-tokens-reset');

        if (!requestsLimit && !tokensLimit) return undefined;

        return {
            requestsLimit: requestsLimit ? parseInt(requestsLimit, 10) : undefined,
            requestsRemaining: requestsRemaining ? parseInt(requestsRemaining, 10) : undefined,
            requestsReset: requestsReset ?? undefined,
            tokensLimit: tokensLimit ? parseInt(tokensLimit, 10) : undefined,
            tokensRemaining: tokensRemaining ? parseInt(tokensRemaining, 10) : undefined,
            tokensReset: tokensReset ?? undefined,
        };
    }
}

/**
 * Creates an Anthropic provider.
 */
export function createAnthropicProvider(config: ProviderFactoryConfig): Provider {
    return new AnthropicProvider(config);
}
