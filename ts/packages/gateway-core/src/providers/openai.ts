/**
 * OpenAI provider - makes requests to the OpenAI API.
 *
 * @module providers/openai
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
import { OpenAICodec } from '../codecs/openai.js';

// ============================================================================
// Constants
// ============================================================================

const DEFAULT_BASE_URL = 'https://api.openai.com';
const MODELS_PATH = '/v1/models';
const CHAT_PATH = '/v1/chat/completions';

// ============================================================================
// OpenAI Provider
// ============================================================================

/**
 * OpenAI provider implementation.
 */
export class OpenAIProvider implements Provider {
    readonly name: string;
    readonly apiType: APIType = 'openai';

    private readonly apiKey: string;
    private readonly baseUrl: string;
    private readonly codec: OpenAICodec;
    private readonly fetchFn: typeof fetch;

    constructor(config: ProviderFactoryConfig) {
        this.name = config.name;
        this.apiKey = config.apiKey;
        this.baseUrl = (config.baseUrl ?? DEFAULT_BASE_URL).replace(/\/$/, '');
        this.codec = new OpenAICodec();
        this.fetchFn = config.fetch ?? globalThis.fetch.bind(globalThis);
    }

    /**
     * Makes a non-streaming completion request.
     */
    async complete(request: CanonicalRequest): Promise<CanonicalResponse> {
        const body = this.codec.encodeRequest({ ...request, stream: false });

        const response = await this.fetchFn(`${this.baseUrl}${CHAT_PATH}`, {
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
        canonicalResponse.sourceAPIType = 'openai';

        // Extract rate limits from headers
        canonicalResponse.rateLimits = this.extractRateLimits(response.headers);

        return canonicalResponse;
    }

    /**
     * Makes a streaming completion request.
     */
    async *stream(request: CanonicalRequest): AsyncGenerator<CanonicalEvent, void, void> {
        const body = this.codec.encodeRequest({ ...request, stream: true });

        const response = await this.fetchFn(`${this.baseUrl}${CHAT_PATH}`, {
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

                    // Skip empty lines and comments
                    if (!trimmed || trimmed.startsWith(':')) continue;

                    // Parse SSE data
                    if (trimmed.startsWith('data: ')) {
                        const data = trimmed.slice(6);

                        // Check for [DONE] marker
                        if (data === '[DONE]') {
                            yield { type: 'done' };
                            return;
                        }

                        // Decode the chunk
                        const event = this.codec.decodeStreamChunk(data);
                        if (event) {
                            yield event;
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
     */
    async listModels(): Promise<ModelList> {
        const response = await this.fetchFn(`${this.baseUrl}${MODELS_PATH}`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${this.apiKey}`,
            },
        });

        if (!response.ok) {
            const responseBody = await response.arrayBuffer();
            throw this.codec.decodeError(new Uint8Array(responseBody), response.status);
        }

        const data = await response.json() as {
            object: string;
            data: Array<{
                id: string;
                object?: string;
                owned_by?: string;
                created?: number;
            }>;
        };

        return {
            object: data.object,
            data: data.data.map((m) => ({
                id: m.id,
                object: m.object,
                ownedBy: m.owned_by,
                created: m.created,
            })),
        };
    }

    /**
     * Gets request headers.
     */
    private getHeaders(request: CanonicalRequest): Record<string, string> {
        const headers: Record<string, string> = {
            'Authorization': `Bearer ${this.apiKey}`,
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
        const requestsLimit = headers.get('x-ratelimit-limit-requests');
        const requestsRemaining = headers.get('x-ratelimit-remaining-requests');
        const requestsReset = headers.get('x-ratelimit-reset-requests');
        const tokensLimit = headers.get('x-ratelimit-limit-tokens');
        const tokensRemaining = headers.get('x-ratelimit-remaining-tokens');
        const tokensReset = headers.get('x-ratelimit-reset-tokens');

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
 * Creates an OpenAI provider.
 */
export function createOpenAIProvider(config: ProviderFactoryConfig): Provider {
    return new OpenAIProvider(config);
}
