/**
 * HTTP middleware for gateway requests.
 *
 * Provides request ID, logging, timeout, and rate limit normalization
 * middleware that work with Web standard Request/Response.
 *
 * @module http/middleware
 */

import type { Logger } from '../utils/logging.js';
import { randomUUID } from '../utils/crypto.js';

// ============================================================================
// Context Types
// ============================================================================

/**
 * Request context with middleware-injected values.
 */
export interface RequestContext {
    /** Unique request ID. */
    requestId: string;

    /** Request start time. */
    startTime: number;

    /** Custom log fields added during request handling. */
    logFields: Map<string, string>;

    /** Rate limit info to write as response headers. */
    rateLimits?: HttpRateLimitInfo | undefined;
}

/**
 * Rate limit information for response headers.
 * Named HttpRateLimitInfo to avoid conflict with domain RateLimitInfo.
 */
export interface HttpRateLimitInfo {
    requestsLimit?: number | undefined;
    requestsRemaining?: number | undefined;
    requestsReset?: string | undefined;
    tokensLimit?: number | undefined;
    tokensRemaining?: number | undefined;
    tokensReset?: string | undefined;
}

// ============================================================================
// Context Storage
// ============================================================================

/**
 * WeakMap to store request context without modifying Request object.
 */
const requestContexts = new WeakMap<Request, RequestContext>();

/**
 * Gets the request context for a request.
 */
export function getRequestContext(request: Request): RequestContext | undefined {
    return requestContexts.get(request);
}

/**
 * Gets or creates a request context.
 */
export function ensureRequestContext(request: Request): RequestContext {
    let ctx = requestContexts.get(request);
    if (!ctx) {
        ctx = {
            requestId: randomUUID(),
            startTime: Date.now(),
            logFields: new Map(),
        };
        requestContexts.set(request, ctx);
    }
    return ctx;
}

/**
 * Gets the request ID from a request.
 */
export function getRequestId(request: Request): string {
    return getRequestContext(request)?.requestId ?? '';
}

/**
 * Adds a log field to the request context.
 */
export function addLogField(request: Request, key: string, value: string): void {
    const ctx = getRequestContext(request);
    if (ctx && value) {
        ctx.logFields.set(key, value);
    }
}

/**
 * Adds an error to the request context log fields.
 */
export function addError(request: Request, error: Error): void {
    addLogField(request, 'error', error.message);
}

/**
 * Sets rate limit info in the request context.
 */
export function setRateLimits(request: Request, rateLimits: HttpRateLimitInfo): void {
    const ctx = getRequestContext(request);
    if (ctx) {
        ctx.rateLimits = rateLimits;
    }
}

// ============================================================================
// Middleware Types
// ============================================================================

/**
 * HTTP handler function type.
 */
export type HttpHandler = (request: Request) => Promise<Response>;

/**
 * HTTP middleware function type.
 */
export type HttpMiddleware = (handler: HttpHandler) => HttpHandler;

// ============================================================================
// Request ID Middleware
// ============================================================================

/**
 * Adds a unique request ID to each request.
 * Sets X-Request-ID response header.
 */
export function requestIdMiddleware(): HttpMiddleware {
    return (handler) => async (request) => {
        const ctx = ensureRequestContext(request);

        // Check for existing request ID in header
        const existingId = request.headers.get('x-request-id');
        if (existingId) {
            ctx.requestId = existingId;
        }

        const response = await handler(request);

        // Add request ID to response headers
        const newHeaders = new Headers(response.headers);
        newHeaders.set('X-Request-ID', ctx.requestId);

        return new Response(response.body, {
            status: response.status,
            statusText: response.statusText,
            headers: newHeaders,
        });
    };
}

// ============================================================================
// Logging Middleware
// ============================================================================

/**
 * Options for logging middleware.
 */
export interface LoggingMiddlewareOptions {
    /** Logger instance. */
    logger: Logger;

    /** Whether to log request bodies (default: false). */
    logRequestBody?: boolean;

    /** Paths to skip logging for (e.g., health checks). */
    skipPaths?: string[];
}

/**
 * Logs HTTP requests with structured logging.
 */
export function loggingMiddleware(options: LoggingMiddlewareOptions): HttpMiddleware {
    const { logger, skipPaths = ['/health', '/healthz', '/ready'] } = options;

    return (handler) => async (request) => {
        const url = new URL(request.url);

        // Skip logging for certain paths
        if (skipPaths.includes(url.pathname)) {
            return handler(request);
        }

        const ctx = ensureRequestContext(request);

        // Log request start
        logger.info('request started', {
            requestId: ctx.requestId,
            method: request.method,
            path: url.pathname,
        });

        let response: Response | undefined;
        let error: Error | undefined;

        try {
            response = await handler(request);
            return response;
        } catch (e) {
            error = e instanceof Error ? e : new Error(String(e));
            throw e;
        } finally {
            const duration = Date.now() - ctx.startTime;

            // Build log attributes
            const attrs: Record<string, unknown> = {
                requestId: ctx.requestId,
                method: request.method,
                path: url.pathname,
                status: response?.status ?? 500,
                durationMs: duration,
            };

            // Add custom log fields
            for (const [key, value] of ctx.logFields) {
                attrs[key] = value;
            }

            if (error) {
                attrs.error = error.message;
            }

            logger.info('request completed', attrs);
        }
    };
}

// ============================================================================
// Timeout Middleware
// ============================================================================

/**
 * Enforces request timeouts.
 * Uses AbortController to cancel long-running requests.
 */
export function timeoutMiddleware(timeoutMs: number): HttpMiddleware {
    return (handler) => async (request) => {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

        try {
            // Create a new request with the abort signal
            const newRequest = new Request(request.url, {
                method: request.method,
                headers: request.headers,
                body: request.body,
                signal: controller.signal,
            });

            // Copy context to new request
            const ctx = getRequestContext(request);
            if (ctx) {
                requestContexts.set(newRequest, ctx);
            }

            return await handler(newRequest);
        } catch (error) {
            if (error instanceof Error && error.name === 'AbortError') {
                return new Response(
                    JSON.stringify({
                        error: {
                            type: 'timeout',
                            message: `Request timed out after ${timeoutMs}ms`,
                        },
                    }),
                    {
                        status: 504,
                        headers: { 'Content-Type': 'application/json' },
                    },
                );
            }
            throw error;
        } finally {
            clearTimeout(timeoutId);
        }
    };
}

// ============================================================================
// Rate Limit Header Middleware
// ============================================================================

/**
 * Writes normalized rate limit headers to responses.
 */
export function rateLimitMiddleware(): HttpMiddleware {
    return (handler) => async (request) => {
        const response = await handler(request);
        const ctx = getRequestContext(request);

        if (!ctx?.rateLimits) {
            return response;
        }

        const rl = ctx.rateLimits;
        const newHeaders = new Headers(response.headers);

        // Write normalized rate limit headers
        if (rl.requestsLimit !== undefined) {
            newHeaders.set('x-ratelimit-limit-requests', String(rl.requestsLimit));
        }
        if (rl.requestsRemaining !== undefined) {
            newHeaders.set('x-ratelimit-remaining-requests', String(rl.requestsRemaining));
        }
        if (rl.requestsReset) {
            newHeaders.set('x-ratelimit-reset-requests', rl.requestsReset);
        }
        if (rl.tokensLimit !== undefined) {
            newHeaders.set('x-ratelimit-limit-tokens', String(rl.tokensLimit));
        }
        if (rl.tokensRemaining !== undefined) {
            newHeaders.set('x-ratelimit-remaining-tokens', String(rl.tokensRemaining));
        }
        if (rl.tokensReset) {
            newHeaders.set('x-ratelimit-reset-tokens', rl.tokensReset);
        }

        return new Response(response.body, {
            status: response.status,
            statusText: response.statusText,
            headers: newHeaders,
        });
    };
}

// ============================================================================
// Middleware Composition
// ============================================================================

/**
 * Composes multiple middleware into a single middleware.
 * Middleware are applied in order (first middleware wraps outermost).
 */
export function composeMiddleware(...middlewares: HttpMiddleware[]): HttpMiddleware {
    return (handler) => {
        return middlewares.reduceRight((h, mw) => mw(h), handler);
    };
}

/**
 * Creates a standard middleware stack.
 */
export function createStandardMiddleware(options: {
    logger: Logger;
    timeoutMs?: number;
}): HttpMiddleware {
    const middlewares: HttpMiddleware[] = [
        requestIdMiddleware(),
        loggingMiddleware({ logger: options.logger }),
        rateLimitMiddleware(),
    ];

    if (options.timeoutMs) {
        middlewares.push(timeoutMiddleware(options.timeoutMs));
    }

    return composeMiddleware(...middlewares);
}
