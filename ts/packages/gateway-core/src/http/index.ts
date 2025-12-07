/**
 * HTTP utilities module exports.
 *
 * @module http
 */

export {
    // Context types
    type RequestContext,
    type HttpRateLimitInfo,

    // Context helpers
    getRequestContext,
    ensureRequestContext,
    getRequestId,
    addLogField,
    addError,
    setRateLimits,

    // Handler types
    type HttpHandler,
    type HttpMiddleware,

    // Middleware
    requestIdMiddleware,
    loggingMiddleware,
    type LoggingMiddlewareOptions,
    timeoutMiddleware,
    rateLimitMiddleware,

    // Composition
    composeMiddleware,
    createStandardMiddleware,
} from './middleware.js';
