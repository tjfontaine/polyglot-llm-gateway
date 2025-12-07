/**
 * Responses API frontdoor - handles /v1/responses endpoints.
 *
 * @module frontdoors/responses
 */

import type { Frontdoor, FrontdoorContext, FrontdoorResponse } from './types.js';
import type { ResponsesAPIRequest } from '../domain/responses.js';
import { ResponsesHandler } from '../responses/handler.js';
import { APIError, errServer, errInvalidRequest, errNotFound, toOpenAIError } from '../domain/errors.js';

// ============================================================================
// Responses Frontdoor
// ============================================================================

/**
 * Frontdoor for OpenAI Responses API.
 */
class ResponsesFrontdoor implements Frontdoor {
    readonly name = 'responses';

    matches(path: string): boolean {
        return path.startsWith('/v1/responses') || path.startsWith('/v1/threads');
    }

    async handle(ctx: FrontdoorContext): Promise<FrontdoorResponse> {
        const { request, provider, auth, logger, app, storage } = ctx;
        const url = new URL(request.url);
        const path = url.pathname;
        const method = request.method;

        // Need storage for Responses API
        if (!storage) {
            return this.errorResponse(errServer('Storage not configured for Responses API'));
        }

        const handler = new ResponsesHandler({
            storage,
            provider,
            logger,
        });

        try {
            // ============================================
            // Responses Routes
            // ============================================

            // POST /v1/responses - Create response
            if (method === 'POST' && path === '/v1/responses') {
                const body = await request.json() as ResponsesAPIRequest;

                // Validate required fields
                if (!body.model) {
                    return this.errorResponse(errInvalidRequest('model is required'));
                }
                if (!body.input) {
                    return this.errorResponse(errInvalidRequest('input is required'));
                }

                // Check if streaming is requested
                if (body.stream) {
                    // Streaming response
                    const sseStream = this.createSSEStream(
                        handler.handleStream(body, auth.tenantId, app?.name),
                    );

                    return {
                        response: new Response(sseStream, {
                            status: 200,
                            headers: {
                                'Content-Type': 'text/event-stream',
                                'Cache-Control': 'no-cache',
                                'Connection': 'keep-alive',
                            },
                        }),
                    };
                }

                // Non-streaming response
                const response = await handler.handle(body, auth.tenantId, app?.name);

                return {
                    response: new Response(JSON.stringify(response), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // POST /v1/responses/:id/cancel - Cancel response
            if (method === 'POST' && path.match(/^\/v1\/responses\/resp_[a-zA-Z0-9]+\/cancel$/)) {
                const responseId = path.split('/')[3]!;
                const response = await handler.cancel(responseId, auth.tenantId);

                if (!response) {
                    return this.errorResponse(errNotFound(`Response '${responseId}' not found`));
                }

                return {
                    response: new Response(JSON.stringify(response), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // GET /v1/responses/:id - Get response
            if (method === 'GET' && path.match(/^\/v1\/responses\/resp_[a-zA-Z0-9]+$/)) {
                const responseId = path.split('/').pop()!;
                const response = await handler.get(responseId);

                if (!response) {
                    return this.errorResponse(errNotFound(`Response '${responseId}' not found`));
                }

                return {
                    response: new Response(JSON.stringify(response), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // GET /v1/responses - List responses
            if (method === 'GET' && path === '/v1/responses') {
                const limit = parseInt(url.searchParams.get('limit') ?? '20', 10);
                const responses = await handler.list(auth.tenantId, { limit });

                return {
                    response: new Response(
                        JSON.stringify({
                            object: 'list',
                            data: responses,
                            has_more: responses.length >= limit,
                        }),
                        {
                            status: 200,
                            headers: { 'Content-Type': 'application/json' },
                        },
                    ),
                };
            }

            // ============================================
            // Thread Routes
            // ============================================

            // POST /v1/threads - Create thread
            if (method === 'POST' && path === '/v1/threads') {
                const body = await request.json().catch(() => ({})) as { metadata?: Record<string, string> };
                const thread = await handler.createThread(auth.tenantId, body.metadata);

                return {
                    response: new Response(JSON.stringify(thread), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // GET /v1/threads/:id - Get thread
            if (method === 'GET' && path.match(/^\/v1\/threads\/thread_[a-zA-Z0-9]+$/)) {
                const threadId = path.split('/').pop()!;
                const thread = await handler.getThread(threadId, auth.tenantId);

                if (!thread) {
                    return this.errorResponse(errNotFound(`Thread '${threadId}' not found`));
                }

                return {
                    response: new Response(JSON.stringify(thread), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // POST /v1/threads/:id/messages - Create message
            if (method === 'POST' && path.match(/^\/v1\/threads\/thread_[a-zA-Z0-9]+\/messages$/)) {
                const threadId = path.split('/')[3]!;
                const body = await request.json() as { role?: 'user' | 'assistant'; content: string };

                if (!body.content) {
                    return this.errorResponse(errInvalidRequest('content is required'));
                }

                const message = await handler.createMessage(
                    threadId,
                    auth.tenantId,
                    body.role ?? 'user',
                    body.content,
                );

                if (!message) {
                    return this.errorResponse(errNotFound(`Thread '${threadId}' not found`));
                }

                return {
                    response: new Response(JSON.stringify(message), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // GET /v1/threads/:id/messages - List messages
            if (method === 'GET' && path.match(/^\/v1\/threads\/thread_[a-zA-Z0-9]+\/messages$/)) {
                const threadId = path.split('/')[3]!;
                const messages = await handler.listMessages(threadId, auth.tenantId);

                return {
                    response: new Response(
                        JSON.stringify({
                            object: 'list',
                            data: messages,
                        }),
                        {
                            status: 200,
                            headers: { 'Content-Type': 'application/json' },
                        },
                    ),
                };
            }

            // POST /v1/threads/:id/runs - Create run
            if (method === 'POST' && path.match(/^\/v1\/threads\/thread_[a-zA-Z0-9]+\/runs$/)) {
                const threadId = path.split('/')[3]!;
                const body = await request.json().catch(() => ({})) as {
                    model?: string;
                    instructions?: string;
                };

                const response = await handler.createRun(threadId, auth.tenantId, body);

                if (!response) {
                    return this.errorResponse(errNotFound(`Thread '${threadId}' not found`));
                }

                return {
                    response: new Response(JSON.stringify(response), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // Not found
            return this.errorResponse(errNotFound('Endpoint not found'), 404);
        } catch (error) {
            logger?.error('Responses API error', {
                error: error instanceof Error ? error.message : String(error),
            });

            if (error instanceof APIError) {
                return this.errorResponse(error);
            }

            return this.errorResponse(errServer(error instanceof Error ? error.message : 'Internal error'));
        }
    }

    private errorResponse(error: APIError, status?: number): FrontdoorResponse {
        return {
            response: new Response(JSON.stringify(toOpenAIError(error)), {
                status: status ?? error.statusCode,
                headers: { 'Content-Type': 'application/json' },
            }),
        };
    }

    /**
     * Creates a ReadableStream from an async generator of SSE strings.
     */
    private createSSEStream(generator: AsyncGenerator<string>): ReadableStream<Uint8Array> {
        const encoder = new TextEncoder();

        return new ReadableStream({
            async pull(controller) {
                try {
                    const { value, done } = await generator.next();
                    if (done) {
                        controller.close();
                    } else {
                        controller.enqueue(encoder.encode(value));
                    }
                } catch (error) {
                    controller.error(error);
                }
            },
            cancel() {
                generator.return(undefined);
            },
        });
    }
}

// Export singleton
export const responsesFrontdoor = new ResponsesFrontdoor();
