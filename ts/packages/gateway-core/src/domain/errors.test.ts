import { describe, it, expect } from 'vitest';
import {
    APIError,
    errInvalidRequest,
    errAuthentication,
    errRateLimit,
    errServer,
    errNotFound,
    isAPIError,
    toOpenAIError,
    toAnthropicError,
} from './errors';

describe('APIError', () => {
    describe('constructor', () => {
        it('should create error with message', () => {
            const error = new APIError('invalid_request', 'Test error');
            expect(error.message).toBe('Test error');
            expect(error.type).toBe('invalid_request');
            expect(error.name).toBe('APIError');
        });

        it('should create error with options', () => {
            const error = new APIError('server', 'Server error', {
                code: 'model_not_found',
                param: 'model',
                statusCode: 503,
                sourceAPI: 'openai',
            });

            expect(error.type).toBe('server');
            expect(error.code).toBe('model_not_found');
            expect(error.param).toBe('model');
            expect(error.statusCode).toBe(503);
            expect(error.sourceAPI).toBe('openai');
        });

        it('should use default status code based on type', () => {
            expect(new APIError('invalid_request', 'test').statusCode).toBe(400);
            expect(new APIError('authentication', 'test').statusCode).toBe(401);
            expect(new APIError('permission', 'test').statusCode).toBe(403);
            expect(new APIError('not_found', 'test').statusCode).toBe(404);
            expect(new APIError('rate_limit', 'test').statusCode).toBe(429);
            expect(new APIError('server', 'test').statusCode).toBe(500);
            expect(new APIError('overloaded', 'test').statusCode).toBe(503);
        });
    });

    describe('builder methods', () => {
        it('should chain builder methods', () => {
            const error = new APIError('invalid_request', 'Test')
                .withCode('model_not_found')
                .withParam('model')
                .withStatusCode(404)
                .withSourceAPI('openai');

            expect(error.code).toBe('model_not_found');
            expect(error.param).toBe('model');
            expect(error.statusCode).toBe(404);
            expect(error.sourceAPI).toBe('openai');
        });
    });

    describe('toJSON', () => {
        it('should serialize to JSON format', () => {
            const error = new APIError('invalid_request', 'Invalid model', {
                code: 'model_not_found',
                param: 'model',
            });

            const json = error.toJSON();
            expect(json).toEqual({
                type: 'invalid_request',
                message: 'Invalid model',
                code: 'model_not_found',
                param: 'model',
            });
        });
    });
});

describe('convenience constructors', () => {
    it('errInvalidRequest should create invalid_request error', () => {
        const error = errInvalidRequest('Bad input');
        expect(error.type).toBe('invalid_request');
        expect(error.message).toBe('Bad input');
        expect(error.statusCode).toBe(400);
    });

    it('errAuthentication should create authentication error', () => {
        const error = errAuthentication('Invalid key');
        expect(error.type).toBe('authentication');
        expect(error.statusCode).toBe(401);
    });

    it('errRateLimit should create rate_limit error', () => {
        const error = errRateLimit('Too many requests');
        expect(error.type).toBe('rate_limit');
        expect(error.statusCode).toBe(429);
    });

    it('errServer should create server error', () => {
        const error = errServer('Internal error');
        expect(error.type).toBe('server');
        expect(error.statusCode).toBe(500);
    });

    it('errNotFound should create not_found error', () => {
        const error = errNotFound('Model not found');
        expect(error.type).toBe('not_found');
        expect(error.statusCode).toBe(404);
    });
});

describe('isAPIError', () => {
    it('should return true for APIError instances', () => {
        const error = new APIError('server', 'Test');
        expect(isAPIError(error)).toBe(true);
    });

    it('should return false for regular Error instances', () => {
        const error = new Error('Test');
        expect(isAPIError(error)).toBe(false);
    });

    it('should return false for non-error values', () => {
        expect(isAPIError(null)).toBe(false);
        expect(isAPIError(undefined)).toBe(false);
        expect(isAPIError('error')).toBe(false);
        expect(isAPIError({ message: 'test' })).toBe(false);
    });
});

describe('toOpenAIError', () => {
    it('should convert to OpenAI error format', () => {
        const error = new APIError('invalid_request', 'Invalid model', {
            code: 'model_not_found',
            param: 'model',
        });

        const openaiError = toOpenAIError(error);
        expect(openaiError).toEqual({
            error: {
                type: 'invalid_request_error',
                code: 'model_not_found',
                message: 'Invalid model',
                param: 'model',
            },
        });
    });
});

describe('toAnthropicError', () => {
    it('should convert to Anthropic error format', () => {
        const error = new APIError('authentication', 'Invalid API key');

        const anthropicError = toAnthropicError(error);
        expect(anthropicError).toEqual({
            type: 'error',
            error: {
                type: 'authentication_error',
                message: 'Invalid API key',
            },
        });
    });
});
