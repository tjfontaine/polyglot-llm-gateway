/**
 * Shadow mode types for the polyglot LLM gateway.
 *
 * @module domain/shadow
 */

import type { APIType, CanonicalRequest, CanonicalResponse } from './types.js';
import type { APIError } from './errors.js';

// ============================================================================
// Shadow Result Types
// ============================================================================

/**
 * Result of a shadow execution.
 */
export interface ShadowResult {
    /** Unique shadow result ID. */
    id: string;

    /** ID of the primary interaction this shadow belongs to. */
    interactionId: string;

    /** Name of the shadow provider. */
    providerName: string;

    /** Shadow request details. */
    request?: ShadowRequest | undefined;

    /** Shadow response (if successful). */
    response?: ShadowResponse | undefined;

    /** Error (if failed). */
    error?: ShadowError | undefined;

    /** Execution duration in milliseconds. */
    durationMs: number;

    /** Detected divergences from primary. */
    divergences: Divergence[];

    /** Quick filter flag for queries. */
    hasStructuralDivergence: boolean;

    /** Timestamp. */
    createdAt: Date;
}

/** Shadow request details. */
export interface ShadowRequest {
    /** Model used for shadow request. */
    model: string;

    /** Max tokens (may differ from primary). */
    maxTokens?: number | undefined;
}

/** Shadow response details. */
export interface ShadowResponse {
    /** Response ID. */
    id: string;

    /** Model that generated the response. */
    model: string;

    /** Generated content. */
    content: string;

    /** Token usage. */
    usage: {
        promptTokens: number;
        completionTokens: number;
        totalTokens: number;
    };

    /** Finish reason. */
    finishReason?: string | undefined;

    /** Tool calls made. */
    toolCalls?: ShadowToolCall[] | undefined;
}

/** Tool call in shadow response. */
export interface ShadowToolCall {
    /** Tool call ID. */
    id: string;

    /** Function name. */
    name: string;

    /** Function arguments. */
    arguments: string;
}

/** Shadow execution error. */
export interface ShadowError {
    /** Error type. */
    type: string;

    /** Error code. */
    code?: string | undefined;

    /** Error message. */
    message: string;

    /** HTTP status code. */
    statusCode?: number | undefined;
}

// ============================================================================
// Divergence Types
// ============================================================================

/** Type of divergence detected. */
export type DivergenceType =
    | 'tool_call_count'
    | 'tool_call_name'
    | 'tool_call_arguments'
    | 'content_length'
    | 'finish_reason'
    | 'error_presence'
    | 'model_mismatch';

/** Severity of a divergence. */
export type DivergenceSeverity = 'info' | 'warning' | 'critical';

/**
 * A detected divergence between primary and shadow responses.
 */
export interface Divergence {
    /** Type of divergence. */
    type: DivergenceType;

    /** Human-readable description. */
    description: string;

    /** Severity level. */
    severity: DivergenceSeverity;

    /** Primary value (for comparison). */
    primaryValue?: string | undefined;

    /** Shadow value (for comparison). */
    shadowValue?: string | undefined;
}

// ============================================================================
// Shadow Configuration
// ============================================================================

/** Configuration for a shadow provider. */
export interface ShadowProviderConfig {
    /** Provider name (must match a configured provider). */
    name: string;

    /** Optional model override. */
    model?: string | undefined;

    /** Max tokens multiplier (0=unlimited, 1=same, >1=increase). */
    maxTokensMultiplier?: number | undefined;
}

/** Shadow mode configuration. */
export interface ShadowConfig {
    /** Whether shadow mode is enabled. */
    enabled: boolean;

    /** Shadow providers to use. */
    providers: ShadowProviderConfig[];

    /** Timeout for shadow execution. */
    timeout?: string | undefined;

    /** Whether to store individual stream chunks. */
    storeStreamChunks?: boolean | undefined;
}

// ============================================================================
// Divergence Detection
// ============================================================================

/**
 * Compares primary and shadow responses and returns divergences.
 */
export function detectDivergences(
    primary: CanonicalResponse,
    shadow: ShadowResponse,
): Divergence[] {
    const divergences: Divergence[] = [];

    // Compare tool call count
    const primaryToolCalls = primary.choices[0]?.message.toolCalls ?? [];
    const shadowToolCalls = shadow.toolCalls ?? [];

    if (primaryToolCalls.length !== shadowToolCalls.length) {
        divergences.push({
            type: 'tool_call_count',
            description: `Primary made ${primaryToolCalls.length} tool calls, shadow made ${shadowToolCalls.length}`,
            severity: 'critical',
            primaryValue: String(primaryToolCalls.length),
            shadowValue: String(shadowToolCalls.length),
        });
    }

    // Compare tool call names
    const primaryNames = primaryToolCalls.map((tc) => tc.function.name).sort();
    const shadowNames = shadowToolCalls.map((tc) => tc.name).sort();

    if (JSON.stringify(primaryNames) !== JSON.stringify(shadowNames)) {
        divergences.push({
            type: 'tool_call_name',
            description: 'Different tool functions called',
            severity: 'critical',
            primaryValue: primaryNames.join(', '),
            shadowValue: shadowNames.join(', '),
        });
    }

    // Compare content length (significant difference threshold: 50%)
    const primaryContent = primary.choices[0]?.message.content ?? '';
    const shadowContent = shadow.content;
    const lengthDiff = Math.abs(primaryContent.length - shadowContent.length);
    const avgLength = (primaryContent.length + shadowContent.length) / 2;

    if (avgLength > 0 && lengthDiff / avgLength > 0.5) {
        divergences.push({
            type: 'content_length',
            description: `Significant content length difference: ${primaryContent.length} vs ${shadowContent.length}`,
            severity: 'warning',
            primaryValue: String(primaryContent.length),
            shadowValue: String(shadowContent.length),
        });
    }

    // Compare finish reason
    const primaryFinish = primary.choices[0]?.finishReason;
    const shadowFinish = shadow.finishReason;

    if (primaryFinish !== shadowFinish) {
        divergences.push({
            type: 'finish_reason',
            description: `Different finish reasons: ${primaryFinish} vs ${shadowFinish}`,
            severity: primaryFinish === 'tool_calls' || shadowFinish === 'tool_calls'
                ? 'critical'
                : 'info',
            primaryValue: primaryFinish ?? 'null',
            shadowValue: shadowFinish ?? 'null',
        });
    }

    return divergences;
}

/**
 * Creates a ShadowResult from execution data.
 */
export function createShadowResult(
    interactionId: string,
    providerName: string,
    durationMs: number,
    options: {
        request?: ShadowRequest;
        response?: ShadowResponse;
        error?: ShadowError;
        primaryResponse?: CanonicalResponse;
    },
): ShadowResult {
    let divergences: Divergence[] = [];
    let hasStructuralDivergence = false;

    if (options.response && options.primaryResponse) {
        divergences = detectDivergences(options.primaryResponse, options.response);
        hasStructuralDivergence = divergences.some(
            (d) => d.severity === 'critical',
        );
    } else if (options.error && options.primaryResponse) {
        divergences = [
            {
                type: 'error_presence',
                description: `Shadow failed with error: ${options.error.message}`,
                severity: 'critical',
                primaryValue: 'success',
                shadowValue: options.error.type,
            },
        ];
        hasStructuralDivergence = true;
    }

    return {
        id: crypto.randomUUID(),
        interactionId,
        providerName,
        request: options.request,
        response: options.response,
        error: options.error,
        durationMs,
        divergences,
        hasStructuralDivergence,
        createdAt: new Date(),
    };
}
