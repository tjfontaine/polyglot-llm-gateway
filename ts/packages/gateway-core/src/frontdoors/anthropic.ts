/**
 * Anthropic frontdoor - handles Anthropic API-compatible requests.
 *
 * @module frontdoors/anthropic
 */

import type { CanonicalRequest, CanonicalResponse } from '../domain/types.js';
import { APIError, isAPIError, errInvalidRequest } from '../domain/errors.js';
import { AnthropicCodec, anthropicCodec } from '../codecs/anthropic.js';
import { createAnthropicSSEStream, sseResponse, sseHeaders } from '../utils/streaming.js';
import type { Frontdoor, FrontdoorContext, FrontdoorResponse } from './types.js';

// ============================================================================
// Anthropic Frontdoor
// ============================================================================

/**
 * Anthropic API-compatible frontdoor.
 */
export class AnthropicFrontdoor implements Frontdoor {
    readonly name = 'anthropic';
    private readonly codec: AnthropicCodec;

    constructor() {
        this.codec = anthropicCodec;
    }

    /**
     * Checks if this frontdoor handles the given path.
     */
    matches(path: string): boolean {
        return path.startsWith('/v1/messages') || path === '/messages';
    }

    /**
     * Handles a request.
     */
    async handle(ctx: FrontdoorContext): Promise<FrontdoorResponse> {
        const url = new URL(ctx.request.url);
        const path = url.pathname;

        // Route to messages handler
        if (path.endsWith('/messages')) {
            return this.handleMessages(ctx);
        }

        return {
            response: new Response(
                JSON.stringify({
                    type: 'error',
                    error: { type: 'not_found_error', message: 'Not found' },
                }),
                { status: 404, headers: { 'Content-Type': 'application/json' } },
            ),
        };
    }

    /**
     * Handles POST /v1/messages
     */
    private async handleMessages(ctx: FrontdoorContext): Promise<FrontdoorResponse> {
        const { request, provider, auth, app, logger, pipeline } = ctx;

        // Validate request method
        if (request.method !== 'POST') {
            return this.errorResponse(errInvalidRequest('Method not allowed'), 405);
        }

        // Decode request
        let canonicalRequest: CanonicalRequest;
        try {
            const body = new Uint8Array(await request.arrayBuffer());
            canonicalRequest = this.codec.decodeRequest(body);
            canonicalRequest.tenantId = auth.tenantId;
            canonicalRequest.userAgent = request.headers.get('user-agent') ?? undefined;

            // Apply default model if configured
            if (!canonicalRequest.model && app?.defaultModel) {
                canonicalRequest.model = app.defaultModel;
            }

            // Validate model
            if (!canonicalRequest.model) {
                return this.errorResponse(errInvalidRequest('model is required'), 400);
            }
        } catch (error) {
            if (isAPIError(error)) {
                return this.errorResponse(error, error.statusCode);
            }
            return this.errorResponse(
                errInvalidRequest('Failed to parse request body'),
                400,
            );
        }

        // Run pre-request middleware pipeline
        const pipelineMetadata = new Map<string, unknown>();
        if (pipeline) {
            const preResult = await pipeline.runPre({
                request: canonicalRequest,
                tenantId: auth.tenantId,
                appName: app?.name,
                interactionId: ctx.interactionId,
                metadata: pipelineMetadata,
            });

            if (!preResult.continue) {
                // Pipeline denied the request or responded early
                if (preResult.response) {
                    // Early response from middleware
                    const responseBody = this.codec.encodeResponse(preResult.response);
                    return {
                        response: new Response(responseBody, {
                            status: 200,
                            headers: { 'Content-Type': 'application/json' },
                        }),
                        canonicalRequest,
                        canonicalResponse: preResult.response,
                    };
                }
                // Denied
                return this.errorResponse(
                    new APIError('permission', preResult.denyReason ?? 'Request denied by middleware'),
                    preResult.denyStatusCode ?? 403,
                );
            }

            // Use potentially modified request
            if (preResult.request) {
                canonicalRequest = preResult.request;
            }
        }

        // Log request
        logger?.info('messages_request', {
            model: canonicalRequest.model,
            stream: canonicalRequest.stream,
            messageCount: canonicalRequest.messages.length,
        });

        try {
            if (canonicalRequest.stream) {
                // Streaming response
                const generator = provider.stream(canonicalRequest);
                const stream = createAnthropicSSEStream(generator, this.codec, {
                    model: canonicalRequest.model,
                    created: Math.floor(Date.now() / 1000),
                });

                // Note: Cannot run post-middleware on streaming responses easily
                return {
                    response: new Response(stream, {
                        status: 200,
                        headers: {
                            ...sseHeaders(),
                            // Anthropic-specific headers
                            'anthropic-version': '2023-06-01',
                        },
                    }),
                    canonicalRequest,
                };
            } else {
                // Non-streaming response
                let canonicalResponse = await provider.complete(canonicalRequest);

                // Run post-request middleware pipeline
                if (pipeline) {
                    const postResult = await pipeline.runPost({
                        request: canonicalRequest,
                        response: canonicalResponse,
                        tenantId: auth.tenantId,
                        appName: app?.name,
                        interactionId: ctx.interactionId,
                        metadata: pipelineMetadata,
                    });

                    if (!postResult.continue) {
                        if (postResult.response) {
                            canonicalResponse = postResult.response;
                        } else if (postResult.denyReason) {
                            return this.errorResponse(
                                new APIError('permission', postResult.denyReason ?? 'Response denied by middleware'),
                                postResult.denyStatusCode ?? 403,
                            );
                        }
                    } else if (postResult.response) {
                        // Modified response
                        canonicalResponse = postResult.response;
                    }
                }

                const responseBody = this.codec.encodeResponse(canonicalResponse);

                return {
                    response: new Response(responseBody, {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                    canonicalRequest,
                    canonicalResponse,
                };
            }
        } catch (error) {
            logger?.error('messages_error', {
                error: error instanceof Error ? error.message : String(error),
            });

            if (isAPIError(error)) {
                return this.errorResponse(error, error.statusCode);
            }
            return this.errorResponse(
                new APIError('server', error instanceof Error ? error.message : 'Internal error'),
                500,
            );
        }
    }

    /**
     * Creates an error response.
     */
    private errorResponse(error: APIError, status?: number): FrontdoorResponse {
        const { body } = this.codec.encodeError(error);
        return {
            response: new Response(body, {
                status: status ?? error.statusCode,
                headers: { 'Content-Type': 'application/json' },
            }),
        };
    }
}

// Export singleton
export const anthropicFrontdoor = new AnthropicFrontdoor();
