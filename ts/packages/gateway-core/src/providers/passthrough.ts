/**
 * Passthrough provider - bypasses canonical conversion when source/target API types match.
 *
 * This is an optimization that sends raw request bytes directly to the provider
 * when the incoming API format matches the provider's API format, avoiding
 * unnecessary encode/decode cycles.
 *
 * @module providers/passthrough
 */

import type {
    CanonicalRequest,
    CanonicalResponse,
    CanonicalEvent,
    APIType,
    Choice,
    Usage,
    Message,
    ModelList,
} from '../domain/types.js';
import type { Provider } from '../ports/provider.js';

// ============================================================================
// Types
// ============================================================================

/**
 * Options for the passthrough provider.
 */
export interface PassthroughOptions {
    /** API key for the provider. */
    apiKey: string;

    /** Base URL for the provider API. */
    baseURL?: string | undefined;

    /** Default headers to include in requests. */
    headers?: Record<string, string> | undefined;
}

/**
 * Extended provider interface with passthrough support.
 */
export interface PassthroughableProvider extends Provider {
    /**
     * Returns true if this provider can handle raw requests from the given API type.
     */
    supportsPassthrough(sourceType: APIType): boolean;

    /**
     * Handles a raw request (when passthrough is supported).
     */
    completeRaw?(request: CanonicalRequest): Promise<[Uint8Array, CanonicalResponse]>;

    /**
     * Handles a raw streaming request.
     */
    streamRaw?(request: CanonicalRequest): AsyncGenerator<CanonicalEvent>;
}

// ============================================================================
// Passthrough Provider
// ============================================================================

/**
 * Wraps a provider to add passthrough support.
 *
 * When the source API type matches the provider's API type and a raw request
 * is available, this provider sends the request directly without canonical
 * conversion, improving latency.
 */
export class PassthroughProvider implements Provider {
    readonly name: string;
    readonly apiType: APIType;
    private readonly inner: Provider;
    private readonly apiKey: string;
    private readonly baseURL: string;
    private readonly headers?: Record<string, string>;

    constructor(inner: Provider, options: PassthroughOptions) {
        this.inner = inner;
        this.name = inner.name;
        this.apiType = inner.apiType;
        this.apiKey = options.apiKey;
        this.baseURL = options.baseURL ?? this.getDefaultBaseURL(inner.apiType);
        this.headers = options.headers;
    }

    /**
     * Returns true if this provider can handle raw requests from the given API type.
     */
    supportsPassthrough(sourceType: APIType): boolean {
        return sourceType === this.apiType && !!this.apiKey;
    }

    /**
     * Completes a request, using passthrough when possible.
     */
    async complete(request: CanonicalRequest): Promise<CanonicalResponse> {
        // Check if we can use passthrough
        if (this.supportsPassthrough(request.sourceAPIType) && request.rawRequest?.length) {
            const [rawResponse, parsedResponse] = await this.completeRaw(request);
            parsedResponse.rawResponse = rawResponse;
            return parsedResponse;
        }

        // Fall back to canonical conversion
        return this.inner.complete(request);
    }

    /**
     * Streams a request, using passthrough when possible.
     */
    stream(request: CanonicalRequest): AsyncGenerator<CanonicalEvent> {
        // Check if we can use passthrough
        if (this.supportsPassthrough(request.sourceAPIType) && request.rawRequest?.length) {
            return this.streamRaw(request);
        }

        // Fall back to canonical conversion
        return this.inner.stream(request);
    }

    /**
     * Lists available models (passes through to inner provider).
     */
    async listModels(): Promise<ModelList> {
        if (this.inner.listModels) {
            return this.inner.listModels();
        }
        return { object: 'list', data: [] };
    }

    // ---- Private Methods ----

    /**
     * Completes a raw request.
     */
    private async completeRaw(request: CanonicalRequest): Promise<[Uint8Array, CanonicalResponse]> {
        const { endpoint, headers } = this.getRequestConfig(this.apiType);

        const response = await fetch(endpoint, {
            method: 'POST',
            headers,
            body: request.rawRequest,
        });

        const rawResponse = new Uint8Array(await response.arrayBuffer());

        if (!response.ok) {
            throw new Error(
                `Provider API error (${response.status}): ${new TextDecoder().decode(rawResponse)}`,
            );
        }

        // Parse the response for canonical format (for recording/logging)
        const parsedResponse = this.parseRawResponse(rawResponse);
        parsedResponse.sourceAPIType = this.apiType;

        return [rawResponse, parsedResponse];
    }

    /**
     * Streams a raw request.
     */
    private async *streamRaw(request: CanonicalRequest): AsyncGenerator<CanonicalEvent> {
        const { endpoint, headers } = this.getRequestConfig(this.apiType);

        // Ensure streaming is enabled in the raw request
        const rawRequest = this.ensureStreamingEnabled(request.rawRequest!);

        const response = await fetch(endpoint, {
            method: 'POST',
            headers,
            body: rawRequest,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Provider API error (${response.status}): ${errorText}`);
        }

        if (!response.body) {
            throw new Error('No response body for streaming');
        }

        // Parse SSE stream
        yield* this.parseSSEStream(response.body, this.apiType);
    }

    /**
     * Gets the default base URL for an API type.
     */
    private getDefaultBaseURL(apiType: APIType): string {
        switch (apiType) {
            case 'openai':
                return 'https://api.openai.com/v1';
            case 'anthropic':
                return 'https://api.anthropic.com/v1';
            default:
                return '';
        }
    }

    /**
     * Gets request configuration for an API type.
     */
    private getRequestConfig(apiType: APIType): { endpoint: string; headers: Record<string, string> } {
        const commonHeaders = {
            'Content-Type': 'application/json',
            ...this.headers,
        };

        switch (apiType) {
            case 'openai':
                return {
                    endpoint: `${this.baseURL}/chat/completions`,
                    headers: {
                        ...commonHeaders,
                        Authorization: `Bearer ${this.apiKey}`,
                    },
                };
            case 'anthropic':
                return {
                    endpoint: `${this.baseURL}/messages`,
                    headers: {
                        ...commonHeaders,
                        'x-api-key': this.apiKey,
                        'anthropic-version': '2023-06-01',
                    },
                };
            default:
                throw new Error(`Unsupported API type for passthrough: ${apiType}`);
        }
    }

    /**
     * Ensures the raw request has streaming enabled.
     */
    private ensureStreamingEnabled(rawRequest: Uint8Array): Uint8Array {
        try {
            const text = new TextDecoder().decode(rawRequest);
            const obj = JSON.parse(text);
            obj.stream = true;
            return new TextEncoder().encode(JSON.stringify(obj));
        } catch {
            return rawRequest;
        }
    }

    /**
     * Parses a raw response into canonical format.
     */
    private parseRawResponse(rawResponse: Uint8Array): CanonicalResponse {
        const text = new TextDecoder().decode(rawResponse);
        const obj = JSON.parse(text);

        switch (this.apiType) {
            case 'openai':
                return this.parseOpenAIResponse(obj);
            case 'anthropic':
                return this.parseAnthropicResponse(obj);
            default:
                throw new Error(`Unsupported API type: ${this.apiType}`);
        }
    }

    /**
     * Parses an OpenAI API response.
     */
    private parseOpenAIResponse(obj: Record<string, unknown>): CanonicalResponse {
        const choices = (obj.choices as Array<{
            index: number;
            message: { role: string; content: string };
            finish_reason: string;
        }>).map((c): Choice => ({
            index: c.index,
            message: {
                role: c.message.role as 'assistant',
                content: c.message.content || '',
            },
            finishReason: c.finish_reason as 'stop' | 'length' | null,
        }));

        const usage = obj.usage as { prompt_tokens: number; completion_tokens: number; total_tokens: number };

        return {
            id: obj.id as string,
            object: obj.object as string,
            created: obj.created as number,
            model: obj.model as string,
            choices,
            usage: {
                promptTokens: usage.prompt_tokens,
                completionTokens: usage.completion_tokens,
                totalTokens: usage.total_tokens,
            },
            sourceAPIType: 'openai',
        };
    }

    /**
     * Parses an Anthropic API response.
     */
    private parseAnthropicResponse(obj: Record<string, unknown>): CanonicalResponse {
        const content = obj.content as Array<{ type: string; text?: string }>;
        const textContent = content
            .filter((c) => c.type === 'text')
            .map((c) => c.text || '')
            .join('');

        const usage = obj.usage as { input_tokens: number; output_tokens: number };

        const message: Message = {
            role: obj.role as 'assistant',
            content: textContent,
        };

        return {
            id: obj.id as string,
            object: 'chat.completion',
            created: Date.now(),
            model: obj.model as string,
            choices: [{
                index: 0,
                message,
                finishReason: this.mapAnthropicStopReason(obj.stop_reason as string),
            }],
            usage: {
                promptTokens: usage.input_tokens,
                completionTokens: usage.output_tokens,
                totalTokens: usage.input_tokens + usage.output_tokens,
            },
            sourceAPIType: 'anthropic',
        };
    }

    /**
     * Maps Anthropic stop reason to canonical finish reason.
     */
    private mapAnthropicStopReason(reason: string): 'stop' | 'length' | 'tool_calls' | null {
        switch (reason) {
            case 'end_turn':
                return 'stop';
            case 'max_tokens':
                return 'length';
            case 'tool_use':
                return 'tool_calls';
            default:
                return null;
        }
    }

    /**
     * Parses an SSE stream into canonical events.
     */
    private async *parseSSEStream(
        body: ReadableStream<Uint8Array>,
        apiType: APIType,
    ): AsyncGenerator<CanonicalEvent> {
        const reader = body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let currentEventType = '';

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                for (const line of lines) {
                    const trimmed = line.trim();
                    if (!trimmed) {
                        // Empty line means end of event
                        currentEventType = '';
                        continue;
                    }

                    if (trimmed.startsWith('event:')) {
                        currentEventType = trimmed.slice(6).trim();
                        continue;
                    }

                    if (trimmed.startsWith('data:')) {
                        const data = trimmed.slice(5).trim();
                        if (data === '[DONE]') {
                            yield { type: 'done' };
                            return;
                        }

                        try {
                            const parsed = JSON.parse(data);
                            const event = this.parseStreamEvent(parsed, apiType, currentEventType);
                            if (event) {
                                event.rawEvent = new TextEncoder().encode(data);
                                yield event;
                            }
                        } catch {
                            // Skip unparseable events
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    }

    /**
     * Parses a single stream event.
     */
    private parseStreamEvent(
        obj: Record<string, unknown>,
        apiType: APIType,
        eventType: string,
    ): CanonicalEvent | null {
        switch (apiType) {
            case 'openai':
                return this.parseOpenAIStreamEvent(obj);
            case 'anthropic':
                return this.parseAnthropicStreamEvent(obj, eventType);
            default:
                return null;
        }
    }

    /**
     * Parses an OpenAI streaming event.
     */
    private parseOpenAIStreamEvent(obj: Record<string, unknown>): CanonicalEvent | null {
        const choices = obj.choices as Array<{
            delta?: { content?: string; role?: string };
        }> | undefined;

        if (choices?.[0]?.delta?.content) {
            return {
                type: 'content_delta',
                contentDelta: choices[0].delta.content,
            };
        }

        if (choices?.[0]?.delta?.role) {
            return {
                type: 'message_start',
                role: choices[0].delta.role,
            };
        }

        const usage = obj.usage as { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number } | undefined;
        if (usage) {
            return {
                type: 'message_delta',
                usage: {
                    promptTokens: usage.prompt_tokens || 0,
                    completionTokens: usage.completion_tokens || 0,
                    totalTokens: usage.total_tokens || 0,
                },
            };
        }

        return null;
    }

    /**
     * Parses an Anthropic streaming event.
     */
    private parseAnthropicStreamEvent(
        obj: Record<string, unknown>,
        eventType: string,
    ): CanonicalEvent | null {
        switch (eventType) {
            case 'content_block_delta': {
                const delta = obj.delta as { type: string; text?: string } | undefined;
                if (delta?.type === 'text_delta' && delta.text) {
                    return {
                        type: 'content_delta',
                        contentDelta: delta.text,
                    };
                }
                break;
            }
            case 'message_start': {
                const message = obj.message as { role?: string; usage?: { input_tokens?: number } } | undefined;
                if (message) {
                    return {
                        type: 'message_start',
                        role: message.role,
                        usage: message.usage?.input_tokens
                            ? { promptTokens: message.usage.input_tokens, completionTokens: 0, totalTokens: message.usage.input_tokens }
                            : undefined,
                    };
                }
                break;
            }
            case 'message_delta': {
                const usage = obj.usage as { output_tokens?: number } | undefined;
                if (usage?.output_tokens) {
                    return {
                        type: 'message_delta',
                        usage: {
                            promptTokens: 0,
                            completionTokens: usage.output_tokens,
                            totalTokens: usage.output_tokens,
                        },
                    };
                }
                break;
            }
            case 'message_stop':
                return { type: 'message_stop' };
        }

        return null;
    }
}

// ============================================================================
// Factory Function
// ============================================================================

/**
 * Wraps a provider with passthrough support.
 */
export function withPassthrough(
    provider: Provider,
    options: PassthroughOptions,
): PassthroughProvider {
    return new PassthroughProvider(provider, options);
}
