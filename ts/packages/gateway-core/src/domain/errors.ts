/**
 * Canonical error types for the polyglot LLM gateway.
 *
 * @module domain/errors
 */

import type { APIType } from './types.js';

// ============================================================================
// Error Types
// ============================================================================

/** Category of an API error. */
export type ErrorType =
    | 'invalid_request'
    | 'authentication'
    | 'permission'
    | 'not_found'
    | 'rate_limit'
    | 'overloaded'
    | 'server'
    | 'context_length'
    | 'max_tokens';

/** Specific error codes for additional detail. */
export type ErrorCode =
    | 'context_length_exceeded'
    | 'rate_limit_exceeded'
    | 'invalid_api_key'
    | 'model_not_found'
    | 'max_tokens_exceeded'
    | 'output_truncated'
    | 'invalid_request_error'
    | 'server_error';

// ============================================================================
// APIError Class
// ============================================================================

/**
 * Canonical API error that can be returned by providers
 * and translated to the appropriate format by frontdoors.
 */
export class APIError extends Error {
    /** Error category. */
    readonly type: ErrorType;

    /** Specific error code. */
    readonly code?: ErrorCode | undefined;

    /** Parameter that caused the error. */
    readonly param?: string | undefined;

    /** HTTP status code. */
    readonly statusCode: number;

    /** Which API the error originated from. */
    readonly sourceAPI?: APIType | undefined;

    constructor(
        type: ErrorType,
        message: string,
        options?: {
            code?: ErrorCode | undefined;
            param?: string | undefined;
            statusCode?: number | undefined;
            sourceAPI?: APIType | undefined;
        },
    ) {
        super(message);
        this.name = 'APIError';
        this.type = type;
        this.code = options?.code;
        this.param = options?.param;
        this.statusCode = options?.statusCode ?? getDefaultStatusCode(type);
        this.sourceAPI = options?.sourceAPI;

        // Maintains proper stack trace for where error was thrown (V8 only)
        if (Error.captureStackTrace) {
            Error.captureStackTrace(this, APIError);
        }
    }

    /**
     * Creates a new error with an additional code.
     */
    withCode(code: ErrorCode): APIError {
        return new APIError(this.type, this.message, {
            code,
            param: this.param,
            statusCode: this.statusCode,
            sourceAPI: this.sourceAPI,
        });
    }

    /**
     * Creates a new error with a parameter name.
     */
    withParam(param: string): APIError {
        return new APIError(this.type, this.message, {
            code: this.code,
            param,
            statusCode: this.statusCode,
            sourceAPI: this.sourceAPI,
        });
    }

    /**
     * Creates a new error with a specific status code.
     */
    withStatusCode(statusCode: number): APIError {
        return new APIError(this.type, this.message, {
            code: this.code,
            param: this.param,
            statusCode,
            sourceAPI: this.sourceAPI,
        });
    }

    /**
     * Creates a new error with source API information.
     */
    withSourceAPI(sourceAPI: APIType): APIError {
        return new APIError(this.type, this.message, {
            code: this.code,
            param: this.param,
            statusCode: this.statusCode,
            sourceAPI,
        });
    }

    /**
     * Converts the error to a plain object for JSON serialization.
     */
    toJSON(): Record<string, unknown> {
        return {
            type: this.type,
            code: this.code,
            message: this.message,
            param: this.param,
        };
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Returns the default HTTP status code for an error type.
 */
function getDefaultStatusCode(type: ErrorType): number {
    switch (type) {
        case 'invalid_request':
        case 'context_length':
        case 'max_tokens':
            return 400;
        case 'authentication':
            return 401;
        case 'permission':
            return 403;
        case 'not_found':
            return 404;
        case 'rate_limit':
            return 429;
        case 'overloaded':
            return 503;
        case 'server':
        default:
            return 500;
    }
}

/**
 * Type guard to check if an error is an APIError.
 */
export function isAPIError(error: unknown): error is APIError {
    return error instanceof APIError;
}

// ============================================================================
// Convenience Constructors
// ============================================================================

/**
 * Creates an invalid request error.
 */
export function errInvalidRequest(message: string): APIError {
    return new APIError('invalid_request', message);
}

/**
 * Creates an authentication error.
 */
export function errAuthentication(message: string): APIError {
    return new APIError('authentication', message);
}

/**
 * Creates a permission error.
 */
export function errPermission(message: string): APIError {
    return new APIError('permission', message);
}

/**
 * Creates a not found error.
 */
export function errNotFound(message: string): APIError {
    return new APIError('not_found', message);
}

/**
 * Creates a rate limit error.
 */
export function errRateLimit(message: string): APIError {
    return new APIError('rate_limit', message, {
        code: 'rate_limit_exceeded',
    });
}

/**
 * Creates an overloaded error.
 */
export function errOverloaded(message: string): APIError {
    return new APIError('overloaded', message);
}

/**
 * Creates a server error.
 */
export function errServer(message: string): APIError {
    return new APIError('server', message);
}

/**
 * Creates a context length exceeded error.
 */
export function errContextLength(message: string): APIError {
    return new APIError('context_length', message, {
        code: 'context_length_exceeded',
    });
}

/**
 * Creates a max tokens error.
 */
export function errMaxTokens(message: string): APIError {
    return new APIError('max_tokens', message, {
        code: 'max_tokens_exceeded',
    });
}

/**
 * Creates an output truncated error.
 */
export function errOutputTruncated(message: string): APIError {
    return new APIError('max_tokens', message, {
        code: 'output_truncated',
    });
}

// ============================================================================
// Error Mapping
// ============================================================================

/** OpenAI error type mapping. */
export const OPENAI_ERROR_TYPE_MAP: Record<ErrorType, string> = {
    invalid_request: 'invalid_request_error',
    authentication: 'authentication_error',
    permission: 'permission_denied',
    not_found: 'not_found',
    rate_limit: 'rate_limit_error',
    overloaded: 'service_unavailable',
    server: 'server_error',
    context_length: 'invalid_request_error',
    max_tokens: 'invalid_request_error',
};

/** Anthropic error type mapping. */
export const ANTHROPIC_ERROR_TYPE_MAP: Record<ErrorType, string> = {
    invalid_request: 'invalid_request_error',
    authentication: 'authentication_error',
    permission: 'permission_error',
    not_found: 'not_found_error',
    rate_limit: 'rate_limit_error',
    overloaded: 'overloaded_error',
    server: 'api_error',
    context_length: 'invalid_request_error',
    max_tokens: 'invalid_request_error',
};

/**
 * Maps an APIError to the OpenAI error format.
 */
export function toOpenAIError(error: APIError): {
    error: {
        type: string;
        code: string | null;
        message: string;
        param: string | null;
    };
} {
    return {
        error: {
            type: OPENAI_ERROR_TYPE_MAP[error.type] ?? 'server_error',
            code: error.code ?? null,
            message: error.message,
            param: error.param ?? null,
        },
    };
}

/**
 * Maps an APIError to the Anthropic error format.
 */
export function toAnthropicError(error: APIError): {
    type: 'error';
    error: {
        type: string;
        message: string;
    };
} {
    return {
        type: 'error',
        error: {
            type: ANTHROPIC_ERROR_TYPE_MAP[error.type] ?? 'api_error',
            message: error.message,
        },
    };
}
