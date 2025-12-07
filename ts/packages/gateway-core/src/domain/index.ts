/**
 * Domain module exports.
 *
 * @module domain
 */

// Types
export * from './types.js';

// Errors
export {
    type ErrorType,
    type ErrorCode,
    APIError,
    isAPIError,
    errInvalidRequest,
    errAuthentication,
    errPermission,
    errNotFound,
    errRateLimit,
    errOverloaded,
    errServer,
    errContextLength,
    errMaxTokens,
    errOutputTruncated,
    toOpenAIError,
    toAnthropicError,
    OPENAI_ERROR_TYPE_MAP,
    ANTHROPIC_ERROR_TYPE_MAP,
} from './errors.js';

// Events
export * from './events.js';

// Responses API
export * from './responses.js';

// Shadow mode
export * from './shadow.js';
