import { describe, it, expect } from 'vitest';
import { AnthropicCodec } from './anthropic';
import type { CanonicalRequest, CanonicalResponse } from '../domain/types';

describe('AnthropicCodec', () => {
    const codec = new AnthropicCodec();

    describe('decodeRequest', () => {
        it('should decode a basic messages request', () => {
            const body = JSON.stringify({
                model: 'claude-3-sonnet-20240229',
                max_tokens: 1024,
                messages: [
                    { role: 'user', content: [{ type: 'text', text: 'Hello' }] },
                ],
            });

            const request = codec.decodeRequest(body);

            expect(request.model).toBe('claude-3-sonnet-20240229');
            expect(request.maxTokens).toBe(1024);
            expect(request.messages).toHaveLength(1);
            expect(request.messages[0]?.role).toBe('user');
            expect(request.sourceAPIType).toBe('anthropic');
        });

        it('should decode request with system message', () => {
            const body = JSON.stringify({
                model: 'claude-3-sonnet-20240229',
                max_tokens: 1024,
                system: [{ type: 'text', text: 'You are helpful' }],
                messages: [
                    { role: 'user', content: [{ type: 'text', text: 'Hi' }] },
                ],
            });

            const request = codec.decodeRequest(body);

            expect(request.messages).toHaveLength(2);
            expect(request.messages[0]?.role).toBe('system');
            expect(request.messages[0]?.content).toBe('You are helpful');
        });

        it('should decode streaming request', () => {
            const body = JSON.stringify({
                model: 'claude-3-sonnet-20240229',
                max_tokens: 1024,
                messages: [{ role: 'user', content: [{ type: 'text', text: 'Hi' }] }],
                stream: true,
            });

            const request = codec.decodeRequest(body);
            expect(request.stream).toBe(true);
        });

        it('should throw on invalid JSON', () => {
            expect(() => codec.decodeRequest('not json')).toThrow('Invalid JSON');
        });
    });

    describe('encodeRequest', () => {
        it('should encode a canonical request to Anthropic format', () => {
            const request: CanonicalRequest = {
                tenantId: 'test',
                model: 'claude-3-sonnet-20240229',
                messages: [
                    { role: 'system', content: 'You are helpful' },
                    { role: 'user', content: 'Hello' },
                ],
                stream: false,
                maxTokens: 1024,
                sourceAPIType: 'anthropic',
            };

            const encoded = codec.encodeRequest(request);
            const parsed = JSON.parse(new TextDecoder().decode(encoded));

            expect(parsed.model).toBe('claude-3-sonnet-20240229');
            expect(parsed.max_tokens).toBe(1024);
            expect(parsed.system).toHaveLength(1);
            expect(parsed.system[0].text).toBe('You are helpful');
            expect(parsed.messages).toHaveLength(1);
        });

        it('should use default max_tokens if not specified', () => {
            const request: CanonicalRequest = {
                tenantId: 'test',
                model: 'claude-3-sonnet-20240229',
                messages: [{ role: 'user', content: 'Hi' }],
                stream: false,
                sourceAPIType: 'anthropic',
            };

            const encoded = codec.encodeRequest(request);
            const parsed = JSON.parse(new TextDecoder().decode(encoded));

            expect(parsed.max_tokens).toBe(4096);
        });
    });

    describe('decodeResponse', () => {
        it('should decode a messages response', () => {
            const body = JSON.stringify({
                id: 'msg_123',
                type: 'message',
                role: 'assistant',
                model: 'claude-3-sonnet-20240229',
                content: [{ type: 'text', text: 'Hello!' }],
                stop_reason: 'end_turn',
                stop_sequence: null,
                usage: {
                    input_tokens: 10,
                    output_tokens: 5,
                },
            });

            const response = codec.decodeResponse(body);

            expect(response.id).toBe('msg_123');
            expect(response.model).toBe('claude-3-sonnet-20240229');
            expect(response.choices).toHaveLength(1);
            expect(response.choices[0]?.message.content).toBe('Hello!');
            expect(response.choices[0]?.finishReason).toBe('stop');
            expect(response.usage.promptTokens).toBe(10);
            expect(response.usage.completionTokens).toBe(5);
            expect(response.usage.totalTokens).toBe(15);
        });

        it('should decode tool use response', () => {
            const body = JSON.stringify({
                id: 'msg_123',
                type: 'message',
                role: 'assistant',
                model: 'claude-3-sonnet-20240229',
                content: [
                    {
                        type: 'tool_use',
                        id: 'toolu_01',
                        name: 'get_weather',
                        input: { location: 'NYC' },
                    },
                ],
                stop_reason: 'tool_use',
                stop_sequence: null,
                usage: { input_tokens: 10, output_tokens: 20 },
            });

            const response = codec.decodeResponse(body);

            expect(response.choices[0]?.finishReason).toBe('tool_calls');
            expect(response.choices[0]?.message.toolCalls).toHaveLength(1);
            expect(response.choices[0]?.message.toolCalls?.[0]?.function.name).toBe('get_weather');
        });
    });

    describe('encodeResponse', () => {
        it('should encode a canonical response to Anthropic format', () => {
            const response: CanonicalResponse = {
                id: 'resp-123',
                object: 'chat.completion',
                created: 1699000000,
                model: 'claude-3-sonnet-20240229',
                choices: [
                    {
                        index: 0,
                        message: { role: 'assistant', content: 'Hello!' },
                        finishReason: 'stop',
                    },
                ],
                usage: {
                    promptTokens: 10,
                    completionTokens: 5,
                    totalTokens: 15,
                },
                sourceAPIType: 'anthropic',
            };

            const encoded = codec.encodeResponse(response);
            const parsed = JSON.parse(new TextDecoder().decode(encoded));

            expect(parsed.id).toBe('resp-123');
            expect(parsed.type).toBe('message');
            expect(parsed.role).toBe('assistant');
            expect(parsed.content[0].type).toBe('text');
            expect(parsed.stop_reason).toBe('end_turn');
            expect(parsed.usage.input_tokens).toBe(10);
            expect(parsed.usage.output_tokens).toBe(5);
        });
    });

    describe('encodeError / decodeError', () => {
        it('should encode error to Anthropic format', () => {
            const error = new Error('Test error');
            const result = codec.encodeError(error);

            expect(result.status).toBe(500);
            const parsed = JSON.parse(result.body);
            expect(parsed.type).toBe('error');
            expect(parsed.error.message).toBe('Test error');
        });

        it('should decode error from Anthropic format', () => {
            const body = JSON.stringify({
                type: 'error',
                error: {
                    type: 'authentication_error',
                    message: 'Invalid API key',
                },
            });

            const error = codec.decodeError(body, 401);

            expect(error.message).toBe('Invalid API key');
        });
    });
});
