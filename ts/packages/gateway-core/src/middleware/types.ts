/**
 * Middleware types - Intermediate Representation for request/response pipeline.
 *
 * @module middleware/types
 */

import type { CanonicalRequest, CanonicalResponse } from '../domain/types.js';

// ============================================================================
// Pipeline Context
// ============================================================================

/**
 * Context passed through the middleware pipeline.
 */
export interface PipelineContext {
    /** The canonical request. */
    request: CanonicalRequest;

    /** The canonical response (set after provider call). */
    response?: CanonicalResponse | undefined;

    /** Tenant ID. */
    tenantId: string;

    /** App name. */
    appName?: string | undefined;

    /** Interaction ID for tracing. */
    interactionId: string;

    /** Metadata storage for middleware. */
    metadata: Map<string, unknown>;

    /** Abort signal. */
    signal?: AbortSignal | undefined;
}

// ============================================================================
// Middleware Step Types
// ============================================================================

/**
 * Result of a middleware step.
 */
export type StepResult =
    | { action: 'continue' }
    | { action: 'modify'; request?: CanonicalRequest; response?: CanonicalResponse }
    | { action: 'deny'; reason: string; statusCode?: number }
    | { action: 'respond'; response: CanonicalResponse };

/**
 * A middleware step function.
 */
export type MiddlewareStep = (ctx: PipelineContext) => Promise<StepResult>;

/**
 * Configuration for a pipeline stage.
 */
export interface StageConfig {
    /** Stage name. */
    name: string;

    /** Stage type (pre = before provider, post = after provider). */
    type: 'pre' | 'post';

    /** Step to execute. */
    step: MiddlewareStep;

    /** Timeout in milliseconds. */
    timeoutMs?: number | undefined;

    /** Error handling mode. */
    onError?: 'allow' | 'deny' | undefined;

    /** Execution order (lower = earlier). */
    order?: number | undefined;
}

// ============================================================================
// Built-in Step Types
// ============================================================================

/**
 * Transform step configuration.
 */
export interface TransformStepConfig {
    type: 'transform';
    /** JSONPath or field name to modify. */
    path: string;
    /** Value to set (string template with {{field}} replacements). */
    value?: string | undefined;
    /** Delete the field. */
    delete?: boolean | undefined;
}

/**
 * Rate limit step configuration.
 */
export interface RateLimitStepConfig {
    type: 'rate_limit';
    /** Requests per window. */
    limit: number;
    /** Window size in seconds. */
    windowSeconds: number;
    /** Key function (tenant, user, ip, custom). */
    keyBy: 'tenant' | 'user' | 'ip' | string;
}

/**
 * Content filter step configuration.
 */
export interface ContentFilterStepConfig {
    type: 'content_filter';
    /** Patterns to block (regex strings). */
    blockPatterns?: string[] | undefined;
    /** Patterns to allow (regex strings). */
    allowPatterns?: string[] | undefined;
    /** Action when blocked. */
    blockAction?: 'deny' | 'remove' | undefined;
}

/**
 * Webhook step configuration.
 */
export interface WebhookStepConfig {
    type: 'webhook';
    /** Webhook URL. */
    url: string;
    /** HTTP method. */
    method?: 'POST' | 'PUT' | undefined;
    /** Additional headers. */
    headers?: Record<string, string> | undefined;
    /** Timeout in milliseconds. */
    timeoutMs?: number | undefined;
    /** Retry count. */
    retries?: number | undefined;
}

/**
 * Log step configuration.
 */
export interface LogStepConfig {
    type: 'log';
    /** Log level. */
    level?: 'debug' | 'info' | 'warn' | undefined;
    /** Log message template. */
    message?: string | undefined;
    /** Fields to include. */
    fields?: string[] | undefined;
}

/**
 * Union of all step configurations.
 */
export type StepConfig =
    | TransformStepConfig
    | RateLimitStepConfig
    | ContentFilterStepConfig
    | WebhookStepConfig
    | LogStepConfig;

// ============================================================================
// Pipeline Configuration
// ============================================================================

/**
 * Full middleware pipeline configuration.
 */
export interface MiddlewarePipelineConfig {
    /** Pre-request stages. */
    pre?: StageConfig[] | undefined;

    /** Post-request stages. */
    post?: StageConfig[] | undefined;
}

// ============================================================================
// Step Result Helpers
// ============================================================================

/**
 * Creates a continue result.
 */
export function continueResult(): StepResult {
    return { action: 'continue' };
}

/**
 * Creates a modify result.
 */
export function modifyResult(
    updates: { request?: CanonicalRequest; response?: CanonicalResponse },
): StepResult {
    return { action: 'modify', ...updates };
}

/**
 * Creates a deny result.
 */
export function denyResult(reason: string, statusCode?: number): StepResult {
    return { action: 'deny', reason, statusCode };
}

/**
 * Creates a respond result.
 */
export function respondResult(response: CanonicalResponse): StepResult {
    return { action: 'respond', response };
}
