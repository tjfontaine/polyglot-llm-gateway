import { describe, it, expect } from 'vitest';
import { OpenAICodec } from './openai';
import type { CanonicalRequest, CanonicalResponse, CanonicalEvent } from '../domain/types';

describe('OpenAICodec', () => {
    const codec = new OpenAICodec();

    describe('decodeRequest', () => {
        it('should decode a basic chat completion request', () => {
            const body = JSON.stringify({
                model: 'gpt-4',
                messages: [
                    { role: 'user', content: 'Hello' },
                ],
            });

            const request = codec.decodeRequest(body);

            expect(request.model).toBe('gpt-4');
            expect(request.messages).toHaveLength(1);
            expect(request.messages[0]?.role).toBe('user');
            expect(request.messages[0]?.content).toBe('Hello');
            expect(request.stream).toBe(false);
            expect(request.sourceAPIType).toBe('openai');
        });

        it('should decode streaming request', () => {
            const body = JSON.stringify({
                model: 'gpt-4',
                messages: [{ role: 'user', content: 'Hi' }],
                stream: true,
            });

            const request = codec.decodeRequest(body);
            expect(request.stream).toBe(true);
        });

        it('should decode request with tools', () => {
            const body = JSON.stringify({
                model: 'gpt-4',
                messages: [{ role: 'user', content: 'Get weather' }],
                tools: [
                    {
                        type: 'function',
                        function: {
                            name: 'get_weather',
                            description: 'Get the weather',
                            parameters: {
                                type: 'object',
                                properties: { location: { type: 'string' } },
                            },
                        },
                    },
                ],
            });

            const request = codec.decodeRequest(body);
            expect(request.tools).toHaveLength(1);
            expect(request.tools?.[0]?.function.name).toBe('get_weather');
        });

        it('should decode max_completion_tokens', () => {
            const body = JSON.stringify({
                model: 'gpt-4',
                messages: [{ role: 'user', content: 'Hi' }],
                max_completion_tokens: 1000,
            });

            const request = codec.decodeRequest(body);
            expect(request.maxTokens).toBe(1000);
        });

        it('should throw on invalid JSON', () => {
            expect(() => codec.decodeRequest('not json')).toThrow('Invalid JSON');
        });
    });

    describe('encodeRequest', () => {
        it('should encode a canonical request to OpenAI format', () => {
            const request: CanonicalRequest = {
                tenantId: 'test',
                model: 'gpt-4',
                messages: [
                    { role: 'user', content: 'Hello' },
                ],
                stream: false,
                maxTokens: 100,
                temperature: 0.7,
                sourceAPIType: 'openai',
            };

            const encoded = codec.encodeRequest(request);
            const parsed = JSON.parse(new TextDecoder().decode(encoded));

            expect(parsed.model).toBe('gpt-4');
            expect(parsed.messages).toHaveLength(1);
            expect(parsed.max_completion_tokens).toBe(100);
            expect(parsed.temperature).toBe(0.7);
        });
    });

    describe('decodeResponse', () => {
        it('should decode a chat completion response', () => {
            const body = JSON.stringify({
                id: 'chatcmpl-123',
                object: 'chat.completion',
                created: 1699000000,
                model: 'gpt-4',
                choices: [
                    {
                        index: 0,
                        message: { role: 'assistant', content: 'Hello!' },
                        finish_reason: 'stop',
                    },
                ],
                usage: {
                    prompt_tokens: 10,
                    completion_tokens: 5,
                    total_tokens: 15,
                },
            });

            const response = codec.decodeResponse(body);

            expect(response.id).toBe('chatcmpl-123');
            expect(response.model).toBe('gpt-4');
            expect(response.choices).toHaveLength(1);
            expect(response.choices[0]?.message.content).toBe('Hello!');
            expect(response.choices[0]?.finishReason).toBe('stop');
            expect(response.usage.promptTokens).toBe(10);
            expect(response.usage.completionTokens).toBe(5);
        });
    });

    describe('encodeResponse', () => {
        it('should encode a canonical response to OpenAI format', () => {
            const response: CanonicalResponse = {
                id: 'resp-123',
                object: 'chat.completion',
                created: 1699000000,
                model: 'gpt-4',
                choices: [
                    {
                        index: 0,
                        message: { role: 'assistant', content: 'Hi!' },
                        finishReason: 'stop',
                    },
                ],
                usage: {
                    promptTokens: 10,
                    completionTokens: 5,
                    totalTokens: 15,
                },
                sourceAPIType: 'openai',
            };

            const encoded = codec.encodeResponse(response);
            const parsed = JSON.parse(new TextDecoder().decode(encoded));

            expect(parsed.id).toBe('resp-123');
            expect(parsed.choices[0].finish_reason).toBe('stop');
            expect(parsed.usage.prompt_tokens).toBe(10);
        });
    });

    describe('decodeStreamChunk', () => {
        it('should decode a content delta chunk', () => {
            const chunk = JSON.stringify({
                id: 'chatcmpl-123',
                object: 'chat.completion.chunk',
                created: 1699000000,
                model: 'gpt-4',
                choices: [
                    {
                        index: 0,
                        delta: { content: 'Hello' },
                        finish_reason: null,
                    },
                ],
            });

            const event = codec.decodeStreamChunk(chunk);

            expect(event).not.toBeNull();
            expect(event?.contentDelta).toBe('Hello');
        });

        it('should decode [DONE] marker', () => {
            const event = codec.decodeStreamChunk('[DONE]');
            expect(event?.type).toBe('done');
        });

        it('should return null for invalid JSON', () => {
            const event = codec.decodeStreamChunk('not json');
            expect(event).toBeNull();
        });
    });

    describe('encodeError / decodeError', () => {
        it('should encode error to OpenAI format', () => {
            const error = new Error('Test error');
            const result = codec.encodeError(error);

            expect(result.status).toBe(500);
            const parsed = JSON.parse(result.body);
            expect(parsed.error.message).toBe('Test error');
        });

        it('should decode error from OpenAI format', () => {
            const body = JSON.stringify({
                error: {
                    type: 'invalid_request_error',
                    code: 'model_not_found',
                    message: 'Model not found',
                    param: 'model',
                },
            });

            const error = codec.decodeError(body, 404);

            expect(error.message).toBe('Model not found');
        });
    });
});
