/**
 * Interaction recorder for full request/response observability.
 *
 * Records interactions with:
 * - Raw request (as received from client)
 * - Canonical request (after decode)
 * - Provider request (what was sent upstream)
 * - Raw response (from provider)
 * - Canonical response (after decode)
 * - Client response (after encode)
 *
 * @module recorder/interaction
 */

import type { CanonicalRequest, CanonicalResponse, APIType } from '../domain/types.js';
import type { StorageProvider } from '../ports/storage.js';
import type { Logger } from '../utils/logging.js';
import { randomUUID } from '../utils/crypto.js';

// ============================================================================
// Types
// ============================================================================

/**
 * Status of an interaction.
 */
export type InteractionStatus = 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled';

/**
 * Parameters for recording an interaction.
 */
export interface RecordInteractionParams {
    /** Raw request as received from client. */
    rawRequest?: Uint8Array | undefined;

    /** Canonical request after decode. */
    canonicalRequest?: CanonicalRequest | undefined;

    /** Fields not mapped during request decode. */
    unmappedRequest?: string[] | undefined;

    /** Raw response from provider. */
    rawResponse?: Uint8Array | undefined;

    /** Canonical response after decode. */
    canonicalResponse?: CanonicalResponse | undefined;

    /** Fields not mapped during response decode. */
    unmappedResponse?: string[] | undefined;

    /** Response sent to client. */
    clientResponse?: Uint8Array | undefined;

    /** Request sent to provider. */
    providerRequestBody?: Uint8Array | undefined;

    /** Request headers (filtered for safety). */
    requestHeaders?: Record<string, string> | undefined;

    /** Frontdoor API type. */
    frontdoor: APIType;

    /** Provider name. */
    provider: string;

    /** App name. */
    appName?: string | undefined;

    /** Whether streaming was used. */
    streaming?: boolean | undefined;

    /** Error if request failed. */
    error?: Error | undefined;

    /** Request duration in milliseconds. */
    durationMs?: number | undefined;

    /** Response finish reason. */
    finishReason?: string | undefined;

    /** Previous interaction ID (for threading). */
    previousInteractionId?: string | undefined;

    /** Thread key. */
    threadKey?: string | undefined;

    /** Tenant ID. */
    tenantId: string;

    /** Request ID. */
    requestId?: string | undefined;
}

/**
 * Stored interaction record.
 */
export interface Interaction {
    /** Unique interaction ID. */
    id: string;

    /** Tenant ID. */
    tenantId: string;

    /** Current status. */
    status: InteractionStatus;

    /** Frontdoor type. */
    frontdoor: APIType;

    /** Provider name. */
    provider: string;

    /** App name. */
    appName?: string | undefined;

    /** Whether streaming was used. */
    streaming: boolean;

    /** Model requested by client. */
    requestedModel?: string | undefined;

    /** Model that served the request. */
    servedModel?: string | undefined;

    /** Duration in milliseconds. */
    durationMs?: number | undefined;

    /** Previous interaction ID. */
    previousInteractionId?: string | undefined;

    /** Thread key. */
    threadKey?: string | undefined;

    /** Request headers (filtered). */
    requestHeaders?: Record<string, string> | undefined;

    /** Request information. */
    request?: InteractionRequest | undefined;

    /** Response information. */
    response?: InteractionResponse | undefined;

    /** Error information. */
    error?: InteractionError | undefined;

    /** Transformation steps for debugging. */
    transformationSteps?: TransformationStep[] | undefined;

    /** Metadata. */
    metadata: Record<string, string>;

    /** Creation timestamp. */
    createdAt: Date;

    /** Update timestamp. */
    updatedAt: Date;
}

/**
 * Request portion of an interaction.
 */
export interface InteractionRequest {
    /** Raw request bytes. */
    raw?: Uint8Array | undefined;

    /** Canonical JSON. */
    canonicalJson?: string | undefined;

    /** Unmapped fields. */
    unmappedFields?: string[] | undefined;

    /** Request sent to provider. */
    providerRequest?: Uint8Array | undefined;
}

/**
 * Response portion of an interaction.
 */
export interface InteractionResponse {
    /** Raw response bytes. */
    raw?: Uint8Array | undefined;

    /** Canonical JSON. */
    canonicalJson?: string | undefined;

    /** Unmapped fields. */
    unmappedFields?: string[] | undefined;

    /** Response sent to client. */
    clientResponse?: Uint8Array | undefined;

    /** Token usage. */
    usage?: { promptTokens: number; completionTokens: number; totalTokens: number } | undefined;

    /** Finish reason. */
    finishReason?: string | undefined;

    /** Provider's response ID. */
    providerResponseId?: string | undefined;
}

/**
 * Error information in an interaction.
 */
export interface InteractionError {
    /** Error type. */
    type: string;

    /** Error code. */
    code?: string | undefined;

    /** Error message. */
    message: string;
}

/**
 * Transformation step for debugging.
 */
export interface TransformationStep {
    /** Stage name. */
    stage: string;

    /** Timestamp. */
    timestamp: Date;

    /** Codec used. */
    codec?: string | undefined;

    /** Description. */
    description: string;

    /** Details. */
    details?: Record<string, unknown> | undefined;

    /** Warnings. */
    warnings?: string[] | undefined;
}

// ============================================================================
// Interaction Recorder
// ============================================================================

/**
 * Options for interaction recorder.
 */
export interface InteractionRecorderOptions {
    /** Storage provider. */
    storage: StorageProvider;

    /** Logger. */
    logger?: Logger | undefined;

    /** Timeout for persistence operations (ms). */
    persistenceTimeoutMs?: number | undefined;
}

/**
 * Records interactions for observability.
 */
export class InteractionRecorder {
    private readonly storage: StorageProvider;
    private readonly logger?: Logger;
    private readonly persistenceTimeoutMs: number;

    constructor(options: InteractionRecorderOptions) {
        this.storage = options.storage;
        this.logger = options.logger;
        this.persistenceTimeoutMs = options.persistenceTimeoutMs ?? 5000;
    }

    /**
     * Records a complete interaction.
     */
    async record(params: RecordInteractionParams): Promise<string> {
        const interactionId = `int_${randomUUID().replace(/-/g, '')}`;
        const now = new Date();

        const interaction = this.buildInteraction(interactionId, params, now);

        // Build transformation steps
        interaction.transformationSteps = this.buildTransformationSteps(params, now);

        // Set status based on outcome
        if (params.error) {
            interaction.status = 'failed';
            interaction.error = {
                type: 'error',
                message: params.error.message,
            };
        } else {
            interaction.status = 'completed';
        }

        // Persist asynchronously without blocking
        this.persistInteraction(interaction).catch((err) => {
            this.logger?.error('Failed to save interaction', {
                interactionId,
                error: err instanceof Error ? err.message : String(err),
            });
        });

        return interactionId;
    }

    /**
     * Starts an in-progress interaction.
     */
    async start(params: RecordInteractionParams): Promise<Interaction> {
        const interactionId = `int_${randomUUID().replace(/-/g, '')}`;
        const now = new Date();

        const interaction = this.buildInteraction(interactionId, params, now);
        interaction.status = 'pending';

        await this.persistInteraction(interaction);

        return interaction;
    }

    /**
     * Completes an in-progress interaction.
     */
    async complete(interaction: Interaction, params: RecordInteractionParams): Promise<void> {
        const now = new Date();

        // Update response data
        interaction.durationMs = params.durationMs;
        interaction.updatedAt = now;

        if (params.canonicalResponse) {
            interaction.servedModel = params.canonicalResponse.model;

            if (params.canonicalResponse.id) {
                interaction.metadata['provider_response_id'] = params.canonicalResponse.id;
            }
        }

        // Build response
        interaction.response = this.buildResponse(params);

        // Build transformation steps
        interaction.transformationSteps = this.buildTransformationSteps(params, interaction.createdAt);

        // Set status
        if (params.error) {
            interaction.status = 'failed';
            interaction.error = {
                type: 'error',
                message: params.error.message,
            };
        } else {
            interaction.status = 'completed';
        }

        await this.persistInteraction(interaction);
    }

    // ---- Private Methods ----

    private buildInteraction(
        id: string,
        params: RecordInteractionParams,
        now: Date,
    ): Interaction {
        const interaction: Interaction = {
            id,
            tenantId: params.tenantId,
            status: 'pending',
            frontdoor: params.frontdoor,
            provider: params.provider,
            appName: params.appName,
            streaming: params.streaming ?? false,
            requestedModel: params.canonicalRequest?.model,
            servedModel: params.canonicalResponse?.model,
            durationMs: params.durationMs,
            previousInteractionId: params.previousInteractionId,
            threadKey: params.threadKey,
            requestHeaders: params.requestHeaders,
            metadata: {},
            createdAt: now,
            updatedAt: now,
        };

        // Add request ID to metadata
        if (params.requestId) {
            interaction.metadata['request_id'] = params.requestId;
        }

        // Build request
        if (params.canonicalRequest || params.rawRequest) {
            interaction.request = {
                raw: params.rawRequest,
                canonicalJson: params.canonicalRequest
                    ? JSON.stringify(params.canonicalRequest)
                    : undefined,
                unmappedFields: params.unmappedRequest,
                providerRequest: params.providerRequestBody,
            };
        }

        // Build response
        interaction.response = this.buildResponse(params);

        return interaction;
    }

    private buildResponse(params: RecordInteractionParams): InteractionResponse | undefined {
        if (!params.canonicalResponse && !params.rawResponse) {
            return undefined;
        }

        return {
            raw: params.rawResponse,
            canonicalJson: params.canonicalResponse
                ? JSON.stringify(params.canonicalResponse)
                : undefined,
            unmappedFields: params.unmappedResponse,
            clientResponse: params.clientResponse,
            usage: params.canonicalResponse?.usage,
            finishReason:
                params.finishReason ??
                params.canonicalResponse?.choices?.[0]?.finishReason ??
                undefined,
            providerResponseId: params.canonicalResponse?.id,
        };
    }

    private buildTransformationSteps(
        params: RecordInteractionParams,
        timestamp: Date,
    ): TransformationStep[] {
        const steps: TransformationStep[] = [];

        // Step 1: Decode request
        if (params.canonicalRequest) {
            steps.push({
                stage: 'decode_request',
                timestamp,
                codec: params.frontdoor,
                description: `Decoded ${params.frontdoor} request to canonical format`,
                details: {
                    unmappedFieldsCount: params.unmappedRequest?.length ?? 0,
                },
                warnings: params.unmappedRequest?.map(
                    (f) => `Field '${f}' could not be mapped`,
                ),
            });
        }

        // Step 2: Model mapping
        if (
            params.canonicalResponse?.model &&
            params.canonicalRequest?.model &&
            params.canonicalResponse.model !== params.canonicalRequest.model
        ) {
            steps.push({
                stage: 'model_mapping',
                timestamp,
                description: 'Mapped model and selected provider',
                details: {
                    originalModel: params.canonicalRequest.model,
                    mappedModel: params.canonicalResponse.model,
                    providerChosen: params.provider,
                },
            });
        }

        // Step 3: Encode for provider
        if (params.providerRequestBody) {
            steps.push({
                stage: 'encode_provider_request',
                timestamp,
                description: `Encoded canonical request to ${params.provider} format`,
                details: { provider: params.provider },
            });
        }

        // Step 4: Decode provider response
        if (params.canonicalResponse) {
            steps.push({
                stage: 'decode_provider_response',
                timestamp,
                description: `Decoded ${params.provider} response to canonical format`,
                details: {
                    unmappedFieldsCount: params.unmappedResponse?.length ?? 0,
                },
                warnings: params.unmappedResponse?.map(
                    (f) => `Field '${f}' could not be mapped`,
                ),
            });
        }

        // Step 5: Encode for client
        if (params.clientResponse) {
            steps.push({
                stage: 'encode_client_response',
                timestamp,
                codec: params.frontdoor,
                description: `Encoded canonical response to ${params.frontdoor} format`,
            });
        }

        return steps;
    }

    private async persistInteraction(interaction: Interaction): Promise<void> {
        // For now, log the interaction
        // TODO: Add dedicated interaction storage method
        this.logger?.info('Interaction recorded', {
            interactionId: interaction.id,
            tenantId: interaction.tenantId,
            status: interaction.status,
            provider: interaction.provider,
            model: interaction.servedModel ?? interaction.requestedModel,
            durationMs: interaction.durationMs,
        });
    }
}

// ============================================================================
// Header Extraction
// ============================================================================

/**
 * Headers to capture for observability.
 */
const RELEVANT_HEADERS = [
    'user-agent',
    'content-type',
    'accept',
    'x-request-id',
    'x-correlation-id',
    'x-trace-id',
    'anthropic-version',
    'anthropic-beta',
    'openai-organization',
    'x-forwarded-for',
    'x-real-ip',
];

/**
 * Extracts relevant headers for observability.
 * Filters out sensitive headers like API keys.
 */
export function extractRelevantHeaders(headers: Headers): Record<string, string> {
    const result: Record<string, string> = {};

    for (const key of RELEVANT_HEADERS) {
        const value = headers.get(key);
        if (value) {
            result[key] = value;
        }
    }

    return result;
}
