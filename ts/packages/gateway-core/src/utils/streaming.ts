/**
 * Streaming utilities using Web Streams API.
 *
 * @module utils/streaming
 */

import type { CanonicalEvent } from '../domain/types.js';
import type { Codec, StreamMetadata } from '../codecs/types.js';

// ============================================================================
// SSE Stream Creation
// ============================================================================

/**
 * Creates an SSE ReadableStream from a CanonicalEvent generator.
 */
export function createSSEStream(
    generator: AsyncGenerator<CanonicalEvent, void, void>,
    codec: Codec,
    metadata?: StreamMetadata,
): ReadableStream<Uint8Array> {
    const encoder = new TextEncoder();

    return new ReadableStream({
        async start(controller) {
            try {
                for await (const event of generator) {
                    // Skip empty events
                    if (event.type === 'done') {
                        controller.enqueue(encoder.encode('data: [DONE]\n\n'));
                        break;
                    }

                    const sseData = codec.encodeStreamEvent(event, metadata);
                    controller.enqueue(encoder.encode(`data: ${sseData}\n\n`));
                }
            } catch (error) {
                // Encode error as SSE event
                const errorEvent: CanonicalEvent = {
                    type: 'error',
                    error: error instanceof Error ? error : new Error(String(error)),
                };
                const sseData = codec.encodeStreamEvent(errorEvent, metadata);
                controller.enqueue(encoder.encode(`data: ${sseData}\n\n`));
            } finally {
                controller.close();
            }
        },
    });
}

/**
 * Creates an Anthropic-style SSE ReadableStream.
 * Anthropic uses event: type prefixes.
 */
export function createAnthropicSSEStream(
    generator: AsyncGenerator<CanonicalEvent, void, void>,
    codec: Codec,
    metadata?: StreamMetadata,
): ReadableStream<Uint8Array> {
    const encoder = new TextEncoder();

    return new ReadableStream({
        async start(controller) {
            try {
                for await (const event of generator) {
                    // Skip empty events
                    if (event.type === 'done') {
                        controller.enqueue(encoder.encode('event: message_stop\ndata: {}\n\n'));
                        break;
                    }

                    const eventType = mapCanonicalToAnthropicEventType(event.type);
                    const sseData = codec.encodeStreamEvent(event, metadata);
                    controller.enqueue(encoder.encode(`event: ${eventType}\ndata: ${sseData}\n\n`));
                }
            } catch (error) {
                const errorEvent: CanonicalEvent = {
                    type: 'error',
                    error: error instanceof Error ? error : new Error(String(error)),
                };
                const sseData = codec.encodeStreamEvent(errorEvent, metadata);
                controller.enqueue(encoder.encode(`event: error\ndata: ${sseData}\n\n`));
            } finally {
                controller.close();
            }
        },
    });
}

/**
 * Maps canonical event type to Anthropic event type.
 */
function mapCanonicalToAnthropicEventType(type: CanonicalEvent['type']): string {
    switch (type) {
        case 'message_start':
            return 'message_start';
        case 'content_delta':
        case 'content_block_delta':
            return 'content_block_delta';
        case 'content_block_start':
            return 'content_block_start';
        case 'content_block_stop':
            return 'content_block_stop';
        case 'message_delta':
            return 'message_delta';
        case 'message_stop':
        case 'done':
            return 'message_stop';
        case 'error':
            return 'error';
        default:
            return 'ping';
    }
}

// ============================================================================
// SSE Response Helpers
// ============================================================================

/**
 * Creates SSE response headers.
 */
export function sseHeaders(): Record<string, string> {
    return {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
    };
}

/**
 * Creates an SSE Response from a stream.
 */
export function sseResponse(
    stream: ReadableStream<Uint8Array>,
    additionalHeaders?: Record<string, string>,
): Response {
    return new Response(stream, {
        status: 200,
        headers: {
            ...sseHeaders(),
            ...additionalHeaders,
        },
    });
}

// ============================================================================
// Stream Utilities
// ============================================================================

/**
 * Collects all events from a CanonicalEvent generator.
 * Useful for testing.
 */
export async function collectEvents(
    generator: AsyncGenerator<CanonicalEvent, void, void>,
): Promise<CanonicalEvent[]> {
    const events: CanonicalEvent[] = [];
    for await (const event of generator) {
        events.push(event);
    }
    return events;
}

/**
 * Creates a generator that emits events from an array.
 * Useful for testing.
 */
export async function* arrayToGenerator(
    events: CanonicalEvent[],
): AsyncGenerator<CanonicalEvent, void, void> {
    for (const event of events) {
        yield event;
    }
}

/**
 * Pipes events through a transform function.
 */
export async function* transformEvents(
    source: AsyncGenerator<CanonicalEvent, void, void>,
    transform: (event: CanonicalEvent) => CanonicalEvent | null,
): AsyncGenerator<CanonicalEvent, void, void> {
    for await (const event of source) {
        const transformed = transform(event);
        if (transformed) {
            yield transformed;
        }
    }
}

/**
 * Accumulates streaming content into a complete response.
 */
export interface StreamAccumulator {
    content: string;
    toolCalls: Map<number, { id: string; name: string; arguments: string }>;
    finishReason?: string;
    usage?: {
        promptTokens: number;
        completionTokens: number;
        totalTokens: number;
    };
    model?: string;
    responseId?: string;
}

/**
 * Creates a new stream accumulator.
 */
export function createStreamAccumulator(): StreamAccumulator {
    return {
        content: '',
        toolCalls: new Map(),
    };
}

/**
 * Accumulates a streaming event.
 */
export function accumulateEvent(
    acc: StreamAccumulator,
    event: CanonicalEvent,
): void {
    if (event.contentDelta) {
        acc.content += event.contentDelta;
    }

    if (event.toolCall) {
        const existing = acc.toolCalls.get(event.toolCall.index);
        if (existing) {
            if (event.toolCall.function?.arguments) {
                existing.arguments += event.toolCall.function.arguments;
            }
        } else {
            acc.toolCalls.set(event.toolCall.index, {
                id: event.toolCall.id ?? '',
                name: event.toolCall.function?.name ?? '',
                arguments: event.toolCall.function?.arguments ?? '',
            });
        }
    }

    if (event.finishReason) {
        acc.finishReason = event.finishReason;
    }

    if (event.usage) {
        acc.usage = event.usage;
    }

    if (event.model) {
        acc.model = event.model;
    }

    if (event.responseId) {
        acc.responseId = event.responseId;
    }
}
