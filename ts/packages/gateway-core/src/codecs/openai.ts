/**
 * OpenAI codec - translates between OpenAI API format and canonical format.
 *
 * @module codecs/openai
 */

import type {
    APIType,
    CanonicalRequest,
    CanonicalResponse,
    CanonicalEvent,
    Message,
    ToolCall,
    ToolDefinition,
    Choice,
    Usage,
    ToolCallChunk,
} from '../domain/types.js';
import {
    APIError,
    toOpenAIError,
    isAPIError,
    errServer,
} from '../domain/errors.js';
import type { Codec, StreamMetadata } from './types.js';
import { toText, toBytes, safeParseJSON } from './types.js';

// ============================================================================
// OpenAI API Types
// ============================================================================

/** OpenAI chat completion request. */
interface OpenAIRequest {
    model: string;
    messages: OpenAIMessage[];
    stream?: boolean;
    max_tokens?: number;
    max_completion_tokens?: number;
    temperature?: number;
    top_p?: number;
    stop?: string | string[];
    tools?: OpenAITool[];
    tool_choice?: unknown;
    response_format?: { type: string; json_schema?: unknown };
    user?: string;
}

/** OpenAI message. */
interface OpenAIMessage {
    role: string;
    content: string | null;
    name?: string;
    tool_calls?: OpenAIToolCall[];
    tool_call_id?: string;
}

/** OpenAI tool call. */
interface OpenAIToolCall {
    id: string;
    type: 'function';
    function: {
        name: string;
        arguments: string;
    };
}

/** OpenAI tool definition. */
interface OpenAITool {
    type: 'function';
    function: {
        name: string;
        description?: string;
        parameters?: Record<string, unknown>;
    };
}

/** OpenAI chat completion response. */
interface OpenAIResponse {
    id: string;
    object: string;
    created: number;
    model: string;
    choices: OpenAIChoice[];
    usage: OpenAIUsage;
    system_fingerprint?: string;
}

/** OpenAI choice. */
interface OpenAIChoice {
    index: number;
    message: OpenAIMessage;
    finish_reason: string | null;
    logprobs?: unknown;
}

/** OpenAI usage. */
interface OpenAIUsage {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
}

/** OpenAI streaming chunk. */
interface OpenAIChunk {
    id: string;
    object: string;
    created: number;
    model: string;
    choices: OpenAIChunkChoice[];
    usage?: OpenAIUsage;
    system_fingerprint?: string;
}

/** OpenAI chunk choice. */
interface OpenAIChunkChoice {
    index: number;
    delta: {
        role?: string;
        content?: string | null;
        tool_calls?: OpenAIToolCallChunk[];
    };
    finish_reason: string | null;
}

/** OpenAI tool call chunk. */
interface OpenAIToolCallChunk {
    index: number;
    id?: string;
    type?: string;
    function?: {
        name?: string;
        arguments?: string;
    };
}

/** OpenAI error response. */
interface OpenAIErrorResponse {
    error: {
        type: string;
        code: string | null;
        message: string;
        param: string | null;
    };
}

// ============================================================================
// OpenAI Codec
// ============================================================================

/**
 * OpenAI codec implementation.
 */
export class OpenAICodec implements Codec {
    readonly name = 'openai';
    readonly apiType: APIType = 'openai';

    // ---- Request handling ----

    decodeRequest(body: Uint8Array | string): CanonicalRequest {
        const json = safeParseJSON<OpenAIRequest>(toText(body));
        if (!json) {
            throw new APIError('invalid_request', 'Invalid JSON in request body');
        }
        return apiRequestToCanonical(json);
    }

    encodeRequest(request: CanonicalRequest): Uint8Array {
        const apiReq = canonicalToApiRequest(request);
        return toBytes(JSON.stringify(apiReq));
    }

    // ---- Response handling ----

    decodeResponse(body: Uint8Array | string): CanonicalResponse {
        const json = safeParseJSON<OpenAIResponse>(toText(body));
        if (!json) {
            throw new APIError('server', 'Invalid JSON in response body');
        }
        return apiResponseToCanonical(json);
    }

    encodeResponse(response: CanonicalResponse): Uint8Array {
        const apiResp = canonicalToApiResponse(response);
        return toBytes(JSON.stringify(apiResp));
    }

    // ---- Streaming ----

    decodeStreamChunk(chunk: string): CanonicalEvent | null {
        // Handle [DONE] marker
        if (chunk.trim() === '[DONE]') {
            return { type: 'done' };
        }

        const json = safeParseJSON<OpenAIChunk>(chunk);
        if (!json) return null;

        return apiChunkToCanonical(json);
    }

    encodeStreamEvent(event: CanonicalEvent, metadata?: StreamMetadata): string {
        const chunk = canonicalToApiChunk(event, metadata);
        return JSON.stringify(chunk);
    }

    // ---- Errors ----

    decodeError(body: Uint8Array | string, status: number): Error {
        const json = safeParseJSON<OpenAIErrorResponse>(toText(body));
        if (!json?.error) {
            return errServer('Unknown error').withStatusCode(status);
        }

        const errorType = mapOpenAIErrorType(json.error.type);
        return new APIError(errorType, json.error.message, {
            code: json.error.code as any,
            param: json.error.param ?? undefined,
            statusCode: status,
            sourceAPI: 'openai',
        });
    }

    encodeError(error: Error): { body: string; status: number } {
        if (isAPIError(error)) {
            return {
                body: JSON.stringify(toOpenAIError(error)),
                status: error.statusCode,
            };
        }
        const apiError = errServer(error.message);
        return {
            body: JSON.stringify(toOpenAIError(apiError)),
            status: 500,
        };
    }
}

// ============================================================================
// Conversion Functions
// ============================================================================

/**
 * Converts OpenAI API request to canonical format.
 */
function apiRequestToCanonical(req: OpenAIRequest): CanonicalRequest {
    const messages: Message[] = req.messages.map((m) => ({
        role: m.role as Message['role'],
        content: m.content ?? '',
        name: m.name,
        toolCallId: m.tool_call_id,
        toolCalls: m.tool_calls?.map((tc): ToolCall => ({
            id: tc.id,
            type: 'function',
            function: {
                name: tc.function.name,
                arguments: tc.function.arguments,
            },
        })),
    }));

    // Prefer max_completion_tokens over max_tokens
    const maxTokens = req.max_completion_tokens ?? req.max_tokens;

    // Convert stop to array
    const stop = req.stop
        ? Array.isArray(req.stop)
            ? req.stop
            : [req.stop]
        : undefined;

    // Convert tools
    const tools = req.tools?.map((t): ToolDefinition => ({
        name: t.function.name,
        type: 'function',
        function: {
            name: t.function.name,
            description: t.function.description,
            parameters: t.function.parameters ?? {},
        },
    }));

    // Convert response format
    const responseFormat = req.response_format
        ? {
            type: req.response_format.type as 'text' | 'json_object' | 'json_schema',
            jsonSchema: req.response_format.json_schema as Record<string, unknown>,
        }
        : undefined;

    return {
        tenantId: '', // Set by gateway
        model: req.model,
        messages,
        stream: req.stream ?? false,
        maxTokens,
        temperature: req.temperature,
        topP: req.top_p,
        stop,
        tools,
        toolChoice: req.tool_choice as CanonicalRequest['toolChoice'],
        responseFormat,
        sourceAPIType: 'openai',
    };
}

/**
 * Converts canonical request to OpenAI API format.
 */
function canonicalToApiRequest(req: CanonicalRequest): OpenAIRequest {
    const messages: OpenAIMessage[] = [];

    // Add system prompt if set separately
    if (req.systemPrompt) {
        messages.push({ role: 'system', content: req.systemPrompt });
    }

    // Add instructions as system message (Responses API)
    if (req.instructions && !req.systemPrompt) {
        messages.push({ role: 'system', content: req.instructions });
    }

    // Add conversation messages
    for (const m of req.messages) {
        const msg: OpenAIMessage = {
            role: m.role,
            content: m.content,
            name: m.name,
            tool_call_id: m.toolCallId,
        };

        if (m.toolCalls?.length) {
            msg.tool_calls = m.toolCalls.map((tc): OpenAIToolCall => ({
                id: tc.id,
                type: 'function',
                function: {
                    name: tc.function.name,
                    arguments: tc.function.arguments,
                },
            }));
        }

        messages.push(msg);
    }

    const apiReq: OpenAIRequest = {
        model: req.model,
        messages,
        stream: req.stream,
    };

    if (req.maxTokens) {
        apiReq.max_completion_tokens = req.maxTokens;
    }

    if (req.temperature !== undefined) {
        apiReq.temperature = req.temperature;
    }

    if (req.topP !== undefined) {
        apiReq.top_p = req.topP;
    }

    if (req.stop?.length) {
        apiReq.stop = req.stop;
    }

    if (req.toolChoice !== undefined) {
        apiReq.tool_choice = req.toolChoice;
    }

    if (req.responseFormat) {
        apiReq.response_format = {
            type: req.responseFormat.type,
            json_schema: req.responseFormat.jsonSchema,
        };
    }

    if (req.tools?.length) {
        apiReq.tools = req.tools.map((t): OpenAITool => ({
            type: 'function',
            function: {
                name: t.function.name,
                description: t.function.description,
                parameters: t.function.parameters,
            },
        }));
    }

    return apiReq;
}

/**
 * Converts OpenAI API response to canonical format.
 */
function apiResponseToCanonical(resp: OpenAIResponse): CanonicalResponse {
    const choices: Choice[] = resp.choices.map((c): Choice => {
        const msg: Message = {
            role: c.message.role as Message['role'],
            content: c.message.content ?? '',
            name: c.message.name,
        };

        if (c.message.tool_calls?.length) {
            msg.toolCalls = c.message.tool_calls.map((tc): ToolCall => ({
                id: tc.id,
                type: 'function',
                function: {
                    name: tc.function.name,
                    arguments: tc.function.arguments,
                },
            }));
        }

        return {
            index: c.index,
            message: msg,
            finishReason: c.finish_reason as Choice['finishReason'],
            logprobs: c.logprobs,
        };
    });

    return {
        id: resp.id,
        object: resp.object,
        created: resp.created,
        model: resp.model,
        choices,
        usage: {
            promptTokens: resp.usage.prompt_tokens,
            completionTokens: resp.usage.completion_tokens,
            totalTokens: resp.usage.total_tokens,
        },
        sourceAPIType: 'openai',
        systemFingerprint: resp.system_fingerprint,
    };
}

/**
 * Converts canonical response to OpenAI API format.
 */
function canonicalToApiResponse(resp: CanonicalResponse): OpenAIResponse {
    const choices: OpenAIChoice[] = resp.choices.map((c): OpenAIChoice => {
        const msg: OpenAIMessage = {
            role: c.message.role,
            content: c.message.content,
            name: c.message.name,
        };

        if (c.message.toolCalls?.length) {
            msg.tool_calls = c.message.toolCalls.map((tc): OpenAIToolCall => ({
                id: tc.id,
                type: 'function',
                function: {
                    name: tc.function.name,
                    arguments: tc.function.arguments,
                },
            }));
        }

        return {
            index: c.index,
            message: msg,
            finish_reason: c.finishReason,
        };
    });

    return {
        id: resp.id,
        object: resp.object || 'chat.completion',
        created: resp.created,
        model: resp.model,
        choices,
        usage: {
            prompt_tokens: resp.usage.promptTokens,
            completion_tokens: resp.usage.completionTokens,
            total_tokens: resp.usage.totalTokens,
        },
        system_fingerprint: resp.systemFingerprint,
    };
}

/**
 * Converts OpenAI streaming chunk to canonical event.
 */
function apiChunkToCanonical(chunk: OpenAIChunk): CanonicalEvent {
    const event: CanonicalEvent = {
        type: 'content_delta',
        responseId: chunk.id,
        model: chunk.model,
    };

    const choice = chunk.choices[0];
    if (choice) {
        event.role = choice.delta.role;
        event.contentDelta = choice.delta.content ?? undefined;

        if (choice.finish_reason) {
            event.finishReason = choice.finish_reason;
            event.type = 'message_stop';
        }

        // Handle tool calls
        const tc = choice.delta.tool_calls?.[0];
        if (tc) {
            event.toolCall = {
                index: tc.index,
                id: tc.id,
                type: tc.type,
                function: tc.function
                    ? {
                        name: tc.function.name,
                        arguments: tc.function.arguments,
                    }
                    : undefined,
            };
        }
    }

    // Handle usage in final chunk
    if (chunk.usage) {
        event.usage = {
            promptTokens: chunk.usage.prompt_tokens,
            completionTokens: chunk.usage.completion_tokens,
            totalTokens: chunk.usage.total_tokens,
        };
    }

    return event;
}

/**
 * Converts canonical event to OpenAI streaming chunk.
 */
function canonicalToApiChunk(
    event: CanonicalEvent,
    metadata?: StreamMetadata,
): OpenAIChunk {
    const chunk: OpenAIChunk = {
        id: metadata?.id ?? '',
        object: 'chat.completion.chunk',
        created: metadata?.created ?? Math.floor(Date.now() / 1000),
        model: metadata?.model ?? event.model ?? '',
        choices: [
            {
                index: 0,
                delta: {
                    role: event.role,
                    content: event.contentDelta,
                },
                finish_reason: event.finishReason ?? null,
            },
        ],
        system_fingerprint: metadata?.systemFingerprint,
    };

    // Handle tool calls
    if (event.toolCall) {
        chunk.choices[0]!.delta.tool_calls = [
            {
                index: event.toolCall.index,
                id: event.toolCall.id,
                type: event.toolCall.type,
                function: event.toolCall.function,
            },
        ];
    }

    // Handle usage
    if (event.usage) {
        chunk.usage = {
            prompt_tokens: event.usage.promptTokens,
            completion_tokens: event.usage.completionTokens,
            total_tokens: event.usage.totalTokens,
        };
    }

    return chunk;
}

/**
 * Maps OpenAI error type to canonical error type.
 */
function mapOpenAIErrorType(
    type: string,
): APIError['type'] {
    switch (type) {
        case 'invalid_request_error':
            return 'invalid_request';
        case 'authentication_error':
            return 'authentication';
        case 'permission_denied':
            return 'permission';
        case 'not_found':
            return 'not_found';
        case 'rate_limit_error':
            return 'rate_limit';
        case 'service_unavailable':
            return 'overloaded';
        case 'server_error':
        default:
            return 'server';
    }
}

// Export singleton instance
export const openaiCodec = new OpenAICodec();
