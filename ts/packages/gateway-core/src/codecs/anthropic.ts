/**
 * Anthropic codec - translates between Anthropic API format and canonical format.
 *
 * @module codecs/anthropic
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
} from '../domain/types.js';
import {
    APIError,
    toAnthropicError,
    isAPIError,
    errServer,
} from '../domain/errors.js';
import type { Codec, StreamMetadata } from './types.js';
import { toText, toBytes, safeParseJSON } from './types.js';

// ============================================================================
// Anthropic API Types
// ============================================================================

/** Anthropic messages request. */
interface AnthropicRequest {
    model: string;
    messages: AnthropicMessage[];
    max_tokens: number;
    system?: AnthropicSystemBlock[];
    stream?: boolean;
    temperature?: number;
    top_p?: number;
    stop_sequences?: string[];
    tools?: AnthropicTool[];
    tool_choice?: AnthropicToolChoice;
    metadata?: { user_id?: string };
}

/** Anthropic system block. */
interface AnthropicSystemBlock {
    type: 'text';
    text: string;
}

/** Anthropic message. */
interface AnthropicMessage {
    role: 'user' | 'assistant';
    content: AnthropicContentBlock[];
}

/** Anthropic content block. */
interface AnthropicContentBlock {
    type: 'text' | 'image' | 'tool_use' | 'tool_result';
    text?: string;
    id?: string;
    name?: string;
    input?: unknown;
    tool_use_id?: string;
    content?: string;
    is_error?: boolean;
    source?: {
        type: 'base64';
        media_type: string;
        data: string;
    };
}

/** Anthropic tool definition. */
interface AnthropicTool {
    name: string;
    description?: string;
    input_schema: Record<string, unknown>;
}

/** Anthropic tool choice. */
interface AnthropicToolChoice {
    type: 'auto' | 'any' | 'tool';
    name?: string;
}

/** Anthropic messages response. */
interface AnthropicResponse {
    id: string;
    type: 'message';
    role: 'assistant';
    model: string;
    content: AnthropicResponseContent[];
    stop_reason: string | null;
    stop_sequence: string | null;
    usage: AnthropicUsage;
}

/** Anthropic response content. */
interface AnthropicResponseContent {
    type: 'text' | 'tool_use';
    text?: string;
    id?: string;
    name?: string;
    input?: unknown;
}

/** Anthropic usage. */
interface AnthropicUsage {
    input_tokens: number;
    output_tokens: number;
}

/** Anthropic error response. */
interface AnthropicErrorResponse {
    type: 'error';
    error: {
        type: string;
        message: string;
    };
}

// ============================================================================
// Anthropic Streaming Types
// ============================================================================

/** Anthropic streaming event types. */
type AnthropicStreamEvent =
    | { type: 'message_start'; message: AnthropicResponse }
    | { type: 'message_delta'; delta: { stop_reason?: string }; usage?: AnthropicUsage }
    | { type: 'message_stop' }
    | { type: 'content_block_start'; index: number; content_block: AnthropicResponseContent }
    | { type: 'content_block_delta'; index: number; delta: { type: string; text?: string } }
    | { type: 'content_block_stop'; index: number }
    | { type: 'ping' };

// ============================================================================
// Anthropic Codec
// ============================================================================

/**
 * Anthropic codec implementation.
 */
export class AnthropicCodec implements Codec {
    readonly name = 'anthropic';
    readonly apiType: APIType = 'anthropic';

    // ---- Request handling ----

    decodeRequest(body: Uint8Array | string): CanonicalRequest {
        const json = safeParseJSON<AnthropicRequest>(toText(body));
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
        const json = safeParseJSON<AnthropicResponse>(toText(body));
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
        const json = safeParseJSON<AnthropicStreamEvent>(chunk);
        if (!json) return null;

        return streamEventToCanonical(json);
    }

    encodeStreamEvent(event: CanonicalEvent, _metadata?: StreamMetadata): string {
        const apiEvent = canonicalToStreamEvent(event);
        return JSON.stringify(apiEvent);
    }

    // ---- Errors ----

    decodeError(body: Uint8Array | string, status: number): Error {
        const json = safeParseJSON<AnthropicErrorResponse>(toText(body));
        if (!json?.error) {
            return errServer('Unknown error').withStatusCode(status);
        }

        const errorType = mapAnthropicErrorType(json.error.type);
        return new APIError(errorType, json.error.message, {
            statusCode: status,
            sourceAPI: 'anthropic',
        });
    }

    encodeError(error: Error): { body: string; status: number } {
        if (isAPIError(error)) {
            return {
                body: JSON.stringify(toAnthropicError(error)),
                status: error.statusCode,
            };
        }
        const apiError = errServer(error.message);
        return {
            body: JSON.stringify(toAnthropicError(apiError)),
            status: 500,
        };
    }
}

// ============================================================================
// Conversion Functions
// ============================================================================

/**
 * Converts Anthropic API request to canonical format.
 */
function apiRequestToCanonical(req: AnthropicRequest): CanonicalRequest {
    const messages: Message[] = [];

    // Add system messages first
    if (req.system) {
        for (const sys of req.system) {
            messages.push({
                role: 'system',
                content: sys.text,
            });
        }
    }

    // Add conversation messages
    for (const msg of req.messages) {
        const content = collapseContentBlocks(msg.content);
        messages.push({
            role: msg.role as Message['role'],
            content,
        });
    }

    // Convert tools
    const tools = req.tools?.map((t): ToolDefinition => ({
        name: t.name,
        type: 'function',
        function: {
            name: t.name,
            description: t.description,
            parameters: t.input_schema,
        },
    }));

    // Convert tool choice
    let toolChoice: CanonicalRequest['toolChoice'];
    if (req.tool_choice) {
        switch (req.tool_choice.type) {
            case 'auto':
                toolChoice = 'auto';
                break;
            case 'any':
                toolChoice = 'required';
                break;
            case 'tool':
                if (req.tool_choice.name) {
                    toolChoice = { type: 'function', function: { name: req.tool_choice.name } };
                }
                break;
        }
    }

    return {
        tenantId: '', // Set by gateway
        model: req.model,
        messages,
        stream: req.stream ?? false,
        maxTokens: req.max_tokens,
        temperature: req.temperature,
        topP: req.top_p,
        stop: req.stop_sequences,
        tools,
        toolChoice,
        sourceAPIType: 'anthropic',
    };
}

/**
 * Collapses Anthropic content blocks to a single string.
 */
function collapseContentBlocks(blocks: AnthropicContentBlock[]): string {
    return blocks
        .filter((b) => b.type === 'text' || b.type === 'input_text' as any || b.type === 'output_text' as any)
        .map((b) => b.text ?? '')
        .join('');
}

/**
 * Converts canonical request to Anthropic API format.
 */
function canonicalToApiRequest(req: CanonicalRequest): AnthropicRequest {
    const system: AnthropicSystemBlock[] = [];
    const messages: AnthropicMessage[] = [];

    // Handle system prompt
    if (req.systemPrompt) {
        system.push({ type: 'text', text: req.systemPrompt });
    }

    // Handle instructions (Responses API) as system message
    if (req.instructions && !req.systemPrompt) {
        system.push({ type: 'text', text: req.instructions });
    }

    // Convert messages
    for (const m of req.messages) {
        if (m.role === 'system') {
            system.push({ type: 'text', text: m.content });
            continue;
        }

        let content: AnthropicContentBlock[];

        if (m.role === 'tool') {
            // Tool results go in user messages
            content = [
                {
                    type: 'tool_result',
                    tool_use_id: m.toolCallId,
                    content: m.content,
                },
            ];
            messages.push({ role: 'user', content });
        } else if (m.role === 'user' && m.toolCallId) {
            // User message with tool call ID is a tool result
            content = [
                {
                    type: 'tool_result',
                    tool_use_id: m.toolCallId,
                    content: m.content,
                },
            ];
            messages.push({ role: 'user', content });
        } else if (m.role === 'assistant' && m.toolCalls?.length) {
            // Assistant with tool calls
            content = [];
            if (m.content) {
                content.push({ type: 'text', text: m.content });
            }
            for (const tc of m.toolCalls) {
                // Parse arguments JSON
                let input: unknown;
                try {
                    input = JSON.parse(tc.function.arguments);
                } catch {
                    input = tc.function.arguments;
                }
                content.push({
                    type: 'tool_use',
                    id: tc.id,
                    name: tc.function.name,
                    input,
                });
            }
            messages.push({ role: 'assistant', content });
        } else {
            // Regular message
            content = [{ type: 'text', text: m.content }];
            messages.push({
                role: m.role as 'user' | 'assistant',
                content,
            });
        }
    }

    const apiReq: AnthropicRequest = {
        model: req.model,
        messages,
        max_tokens: req.maxTokens ?? 4096, // Anthropic requires max_tokens
        stream: req.stream,
    };

    if (system.length > 0) {
        apiReq.system = system;
    }

    if (req.temperature !== undefined) {
        apiReq.temperature = req.temperature;
    }

    if (req.topP !== undefined) {
        apiReq.top_p = req.topP;
    }

    if (req.stop?.length) {
        apiReq.stop_sequences = req.stop;
    }

    // Convert tool choice
    if (req.toolChoice) {
        if (req.toolChoice === 'auto') {
            apiReq.tool_choice = { type: 'auto' };
        } else if (req.toolChoice === 'required') {
            apiReq.tool_choice = { type: 'any' };
        } else if (typeof req.toolChoice === 'object') {
            apiReq.tool_choice = { type: 'tool', name: req.toolChoice.function.name };
        }
        // 'none' - just don't send tools
    }

    // Convert tools
    if (req.tools?.length) {
        apiReq.tools = req.tools.map((t): AnthropicTool => ({
            name: t.function.name,
            description: t.function.description,
            input_schema: t.function.parameters,
        }));
    }

    return apiReq;
}

/**
 * Converts Anthropic API response to canonical format.
 */
function apiResponseToCanonical(resp: AnthropicResponse): CanonicalResponse {
    let content = '';
    const toolCalls: ToolCall[] = [];

    for (const c of resp.content) {
        if (c.type === 'text') {
            content += c.text ?? '';
        } else if (c.type === 'tool_use') {
            const args = c.input
                ? typeof c.input === 'string'
                    ? c.input
                    : JSON.stringify(c.input)
                : '{}';
            toolCalls.push({
                id: c.id ?? '',
                type: 'function',
                function: {
                    name: c.name ?? '',
                    arguments: args,
                },
            });
        }
    }

    const finishReason = mapStopReason(resp.stop_reason);

    const message: Message = {
        role: 'assistant',
        content,
        toolCalls: toolCalls.length > 0 ? toolCalls : undefined,
    };

    return {
        id: resp.id,
        object: 'chat.completion',
        created: Math.floor(Date.now() / 1000),
        model: resp.model,
        choices: [
            {
                index: 0,
                message,
                finishReason,
            },
        ],
        usage: {
            promptTokens: resp.usage.input_tokens,
            completionTokens: resp.usage.output_tokens,
            totalTokens: resp.usage.input_tokens + resp.usage.output_tokens,
        },
        sourceAPIType: 'anthropic',
    };
}

/**
 * Converts canonical response to Anthropic API format.
 */
function canonicalToApiResponse(resp: CanonicalResponse): AnthropicResponse {
    const choice = resp.choices[0];
    const content: AnthropicResponseContent[] = [];

    if (choice?.message.content) {
        content.push({ type: 'text', text: choice.message.content });
    }

    if (choice?.message.toolCalls) {
        for (const tc of choice.message.toolCalls) {
            let input: unknown;
            try {
                input = JSON.parse(tc.function.arguments);
            } catch {
                input = tc.function.arguments;
            }
            content.push({
                type: 'tool_use',
                id: tc.id,
                name: tc.function.name,
                input,
            });
        }
    }

    // Ensure at least one content block
    if (content.length === 0) {
        content.push({ type: 'text', text: '' });
    }

    return {
        id: resp.id,
        type: 'message',
        role: 'assistant',
        model: resp.model,
        content,
        stop_reason: mapFinishReason(choice?.finishReason ?? null),
        stop_sequence: null,
        usage: {
            input_tokens: resp.usage.promptTokens,
            output_tokens: resp.usage.completionTokens,
        },
    };
}

/**
 * Converts Anthropic streaming event to canonical event.
 */
function streamEventToCanonical(event: AnthropicStreamEvent): CanonicalEvent {
    switch (event.type) {
        case 'message_start':
            return {
                type: 'message_start',
                role: event.message.role,
                model: event.message.model,
                usage: {
                    promptTokens: event.message.usage.input_tokens,
                    completionTokens: 0,
                    totalTokens: event.message.usage.input_tokens,
                },
            };

        case 'content_block_delta':
            return {
                type: 'content_delta',
                contentDelta: event.delta.text,
                index: event.index,
            };

        case 'message_delta':
            return {
                type: 'message_delta',
                finishReason: event.delta.stop_reason
                    ? mapStopReason(event.delta.stop_reason) ?? undefined
                    : undefined,
                usage: event.usage
                    ? {
                        promptTokens: 0,
                        completionTokens: event.usage.output_tokens,
                        totalTokens: event.usage.output_tokens,
                    }
                    : undefined,
            };

        case 'message_stop':
            return { type: 'message_stop' };

        case 'content_block_start':
            return {
                type: 'content_block_start',
                index: event.index,
            };

        case 'content_block_stop':
            return {
                type: 'content_block_stop',
                index: event.index,
            };

        case 'ping':
            return { type: 'content_delta' }; // No-op

        default:
            return { type: 'content_delta' };
    }
}

/**
 * Converts canonical event to Anthropic streaming event.
 */
function canonicalToStreamEvent(event: CanonicalEvent): object {
    if (event.contentDelta) {
        return {
            type: 'content_block_delta',
            index: event.index ?? 0,
            delta: {
                type: 'text_delta',
                text: event.contentDelta,
            },
        };
    }

    if (event.type === 'message_stop' || event.type === 'done') {
        return { type: 'message_stop' };
    }

    return { type: 'ping' };
}

/**
 * Maps Anthropic stop_reason to OpenAI finish_reason.
 */
function mapStopReason(stopReason: string | null): Choice['finishReason'] {
    switch (stopReason) {
        case 'end_turn':
        case 'stop_sequence':
            return 'stop';
        case 'max_tokens':
            return 'length';
        case 'tool_use':
            return 'tool_calls';
        default:
            return null;
    }
}

/**
 * Maps OpenAI finish_reason to Anthropic stop_reason.
 */
function mapFinishReason(finishReason: Choice['finishReason']): string | null {
    switch (finishReason) {
        case 'stop':
            return 'end_turn';
        case 'length':
            return 'max_tokens';
        case 'tool_calls':
            return 'tool_use';
        case 'content_filter':
            return 'end_turn';
        default:
            return null;
    }
}

/**
 * Maps Anthropic error type to canonical error type.
 */
function mapAnthropicErrorType(type: string): APIError['type'] {
    switch (type) {
        case 'invalid_request_error':
            return 'invalid_request';
        case 'authentication_error':
            return 'authentication';
        case 'permission_error':
            return 'permission';
        case 'not_found_error':
            return 'not_found';
        case 'rate_limit_error':
            return 'rate_limit';
        case 'overloaded_error':
            return 'overloaded';
        case 'api_error':
        default:
            return 'server';
    }
}

// Export singleton instance
export const anthropicCodec = new AnthropicCodec();
