/**
 * OpenAI frontdoor - handles OpenAI API-compatible requests.
 *
 * @module frontdoors/openai
 */

import type { CanonicalRequest, CanonicalResponse, CanonicalEvent, ModelList } from '../domain/types.js';
import { APIError, isAPIError, errInvalidRequest, errNotFound } from '../domain/errors.js';
import type { Provider } from '../ports/provider.js';
import type { AuthContext } from '../ports/auth.js';
import type { AppConfig } from '../ports/config.js';
import { OpenAICodec, openaiCodec } from '../codecs/openai.js';
import { createSSEStream, sseResponse } from '../utils/streaming.js';
import type { Logger } from '../utils/logging.js';
import type { Frontdoor, FrontdoorContext, FrontdoorResponse } from './types.js';

// ============================================================================
// OpenAI Frontdoor
// ============================================================================

/**
 * OpenAI API-compatible frontdoor.
 */
export class OpenAIFrontdoor implements Frontdoor {
    readonly name = 'openai';
    private readonly codec: OpenAICodec;

    constructor() {
        this.codec = openaiCodec;
    }

    /**
     * Checks if this frontdoor handles the given path.
     */
    matches(path: string): boolean {
        return (
            path.startsWith('/v1/chat/completions') ||
            path.startsWith('/v1/models') ||
            path === '/chat/completions'
        );
    }

    /**
     * Handles a request.
     */
    async handle(ctx: FrontdoorContext): Promise<FrontdoorResponse> {
        const url = new URL(ctx.request.url);
        const path = url.pathname;

        // Route to appropriate handler
        if (path.endsWith('/chat/completions')) {
            return this.handleChatCompletions(ctx);
        }

        if (path.endsWith('/models') || path.includes('/models/')) {
            return this.handleModels(ctx, path);
        }

        return {
            response: new Response(
                JSON.stringify({ error: { message: 'Not found', type: 'not_found' } }),
                { status: 404, headers: { 'Content-Type': 'application/json' } },
            ),
        };
    }

    /**
     * Handles POST /v1/chat/completions
     */
    private async handleChatCompletions(ctx: FrontdoorContext): Promise<FrontdoorResponse> {
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
        logger?.info('chat_completion_request', {
            model: canonicalRequest.model,
            stream: canonicalRequest.stream,
            messageCount: canonicalRequest.messages.length,
        });

        try {
            if (canonicalRequest.stream) {
                // Streaming response
                const generator = provider.stream(canonicalRequest);
                const stream = createSSEStream(generator, this.codec, {
                    model: canonicalRequest.model,
                    created: Math.floor(Date.now() / 1000),
                });

                // Note: Cannot run post-middleware on streaming responses easily
                // The stream is returned directly to the client
                return {
                    response: sseResponse(stream),
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
            logger?.error('chat_completion_error', {
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
     * Handles GET /v1/models
     */
    private async handleModels(
        ctx: FrontdoorContext,
        path: string,
    ): Promise<FrontdoorResponse> {
        const { request, provider, app, logger } = ctx;

        if (request.method !== 'GET') {
            return this.errorResponse(errInvalidRequest('Method not allowed'), 405);
        }

        try {
            // Check for specific model
            const modelMatch = path.match(/\/models\/([^/]+)/);
            if (modelMatch) {
                const modelId = modelMatch[1];
                const models = app?.models
                    ? { object: 'list', data: app.models.map((m) => ({ id: m.id, object: m.object ?? 'model', ownedBy: m.ownedBy })) }
                    : await provider.listModels?.();

                const model = models?.data.find((m) => m.id === modelId);
                if (!model) {
                    return this.errorResponse(errNotFound(`Model '${modelId}' not found`), 404);
                }

                return {
                    response: new Response(JSON.stringify(model), {
                        status: 200,
                        headers: { 'Content-Type': 'application/json' },
                    }),
                };
            }

            // List all models
            let models: ModelList;

            // Use app-configured models if available
            if (app?.models && app.models.length > 0) {
                models = {
                    object: 'list',
                    data: app.models.map((m) => ({
                        id: m.id,
                        object: m.object ?? 'model',
                        ownedBy: m.ownedBy,
                        created: m.created,
                    })),
                };
            } else if (provider.listModels) {
                models = await provider.listModels();
            } else {
                models = { object: 'list', data: [] };
            }

            return {
                response: new Response(JSON.stringify(models), {
                    status: 200,
                    headers: { 'Content-Type': 'application/json' },
                }),
            };
        } catch (error) {
            logger?.error('list_models_error', {
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
export const openAIFrontdoor = new OpenAIFrontdoor();
