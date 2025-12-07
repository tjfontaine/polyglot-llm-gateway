/**
 * Lifecycle event types for the polyglot LLM gateway.
 * These events are published to track request/response lifecycle.
 *
 * @module domain/events
 */

import type { APIType, Usage } from './types.js';
import type { APIError } from './errors.js';

// ============================================================================
// Event Types
// ============================================================================

/** Type of lifecycle event. */
export type LifecycleEventType =
    | 'request_received'
    | 'request_validated'
    | 'provider_selected'
    | 'provider_request_sent'
    | 'provider_response_received'
    | 'response_sent'
    | 'error'
    | 'shadow_started'
    | 'shadow_completed';

/**
 * A lifecycle event in the request processing pipeline.
 */
export interface LifecycleEvent {
    /** Unique event ID. */
    id: string;

    /** Event type. */
    type: LifecycleEventType;

    /** Interaction ID this event belongs to. */
    interactionId: string;

    /** Tenant ID. */
    tenantId: string;

    /** Timestamp when event occurred. */
    timestamp: Date;

    /** Event-specific data. */
    data?: LifecycleEventData | undefined;
}

/** Union type for event-specific data. */
export type LifecycleEventData =
    | RequestReceivedData
    | ProviderSelectedData
    | ProviderResponseData
    | ResponseSentData
    | ErrorData
    | ShadowData;

/** Data for request_received events. */
export interface RequestReceivedData {
    /** Original API type of the request. */
    sourceAPIType: APIType;

    /** Requested model. */
    model: string;

    /** Whether streaming was requested. */
    stream: boolean;

    /** Number of messages in the request. */
    messageCount: number;
}

/** Data for provider_selected events. */
export interface ProviderSelectedData {
    /** Selected provider name. */
    providerName: string;

    /** Provider's API type. */
    providerAPIType: APIType;

    /** Actual model to use (may be rewritten). */
    targetModel: string;
}

/** Data for provider_response_received events. */
export interface ProviderResponseData {
    /** Response ID from provider. */
    responseId: string;

    /** Model that generated the response. */
    model: string;

    /** Token usage. */
    usage?: Usage | undefined;

    /** Duration in milliseconds. */
    durationMs: number;

    /** Finish reason. */
    finishReason?: string | undefined;
}

/** Data for response_sent events. */
export interface ResponseSentData {
    /** HTTP status code. */
    statusCode: number;

    /** Total duration in milliseconds. */
    totalDurationMs: number;
}

/** Data for error events. */
export interface ErrorData {
    /** Error type. */
    errorType: string;

    /** Error code. */
    errorCode?: string | undefined;

    /** Error message. */
    errorMessage: string;

    /** HTTP status code. */
    statusCode: number;
}

/** Data for shadow events. */
export interface ShadowData {
    /** Shadow provider name. */
    providerName: string;

    /** Whether divergence was detected. */
    hasDivergence?: boolean | undefined;

    /** Duration in milliseconds. */
    durationMs?: number | undefined;

    /** Error if shadow failed. */
    error?: APIError | undefined;
}

// ============================================================================
// Interaction Events (for storage)
// ============================================================================

/** Type of interaction event. */
export type InteractionEventType =
    | 'request'
    | 'response'
    | 'stream_start'
    | 'stream_chunk'
    | 'stream_end'
    | 'error'
    | 'pipeline_pre'
    | 'pipeline_post';

/**
 * An event in an interaction's lifecycle (for storage/audit).
 */
export interface InteractionEvent {
    /** Unique event ID. */
    id: string;

    /** Interaction ID. */
    interactionId: string;

    /** Event type. */
    type: InteractionEventType;

    /** Timestamp. */
    timestamp: Date;

    /** Event payload. */
    payload?: unknown;
}

// ============================================================================
// Factory Functions
// ============================================================================

/**
 * Creates a lifecycle event.
 */
export function createLifecycleEvent(
    type: LifecycleEventType,
    interactionId: string,
    tenantId: string,
    data?: LifecycleEventData,
): LifecycleEvent {
    return {
        id: crypto.randomUUID(),
        type,
        interactionId,
        tenantId,
        timestamp: new Date(),
        data,
    };
}

/**
 * Creates an interaction event.
 */
export function createInteractionEvent(
    type: InteractionEventType,
    interactionId: string,
    payload?: unknown,
): InteractionEvent {
    return {
        id: crypto.randomUUID(),
        interactionId,
        type,
        timestamp: new Date(),
        payload,
    };
}
