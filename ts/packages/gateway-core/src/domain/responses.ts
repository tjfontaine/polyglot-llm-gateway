/**
 * OpenAI Responses API types.
 *
 * @module domain/responses
 */

import type { Message, ToolDefinition, Usage } from './types.js';

// ============================================================================
// Responses API Request
// ============================================================================

/**
 * OpenAI Responses API request format.
 */
export interface ResponsesAPIRequest {
    /** Model to use. */
    model: string;

    /** Input to the model. Can be a string or array of input items. */
    input: string | ResponsesInputItem[];

    /** Instructions for the model. */
    instructions?: string | undefined;

    /** Tools the model can use. */
    tools?: ToolDefinition[] | undefined;

    /** Tool choice mode. */
    toolChoice?: 'auto' | 'none' | 'required' | undefined;

    /** Metadata for the response. */
    metadata?: Record<string, string> | undefined;

    /** Maximum tokens to generate. */
    maxOutputTokens?: number | undefined;

    /** Sampling temperature. */
    temperature?: number | undefined;

    /** Top-p sampling. */
    topP?: number | undefined;

    /** Whether to stream the response. */
    stream?: boolean | undefined;

    /** Whether to store the response. */
    store?: boolean | undefined;

    /** Previous response ID for continuation. */
    previousResponseId?: string | undefined;
}

/** Input item for Responses API. */
export interface ResponsesInputItem {
    /** Item type. */
    type: 'message' | 'item_reference';

    /** Role (for message type). */
    role?: 'user' | 'assistant' | undefined;

    /** Content (for message type). */
    content?: string | ResponsesContentPart[] | undefined;

    /** Item ID reference (for item_reference type). */
    id?: string | undefined;
}

/** Content part for Responses API. */
export interface ResponsesContentPart {
    /** Part type. */
    type: 'input_text' | 'input_image' | 'input_audio';

    /** Text content. */
    text?: string | undefined;

    /** Image URL. */
    imageUrl?: string | undefined;

    /** Image data (base64). */
    imageData?: string | undefined;

    /** Audio data (base64). */
    audioData?: string | undefined;
}

// ============================================================================
// Responses API Response
// ============================================================================

/** Response status. */
export type ResponseStatus =
    | 'in_progress'
    | 'completed'
    | 'incomplete'
    | 'cancelled'
    | 'failed';

/**
 * OpenAI Responses API response format.
 */
export interface ResponsesAPIResponse {
    /** Response ID. */
    id: string;

    /** Object type. */
    object: 'response';

    /** Creation timestamp. */
    createdAt: number;

    /** Response status. */
    status: ResponseStatus;

    /** Model used. */
    model: string;

    /** Output items. */
    output: ResponsesOutputItem[];

    /** Token usage. */
    usage?: ResponsesUsage | undefined;

    /** Metadata. */
    metadata?: Record<string, string> | undefined;

    /** Error info if failed. */
    error?: ResponsesError | undefined;

    /** Incomplete details if status is incomplete. */
    incompleteDetails?: IncompleteDetails | undefined;
}

/** Output item in a Responses API response. */
export interface ResponsesOutputItem {
    /** Item type. */
    type: 'message' | 'function_call' | 'function_call_output';

    /** Item ID. */
    id: string;

    /** Item status. */
    status?: 'in_progress' | 'completed' | 'incomplete' | undefined;

    /** Role (for message type). */
    role?: 'assistant' | undefined;

    /** Content (for message type). */
    content?: ResponsesOutputContent[] | undefined;

    /** Function name (for function_call type). */
    name?: string | undefined;

    /** Call ID (for function_call type). */
    callId?: string | undefined;

    /** Function arguments JSON (for function_call type). */
    arguments?: string | undefined;

    /** Function output (for function_call_output type). */
    output?: string | undefined;
}

/** Output content in a Responses API message. */
export interface ResponsesOutputContent {
    /** Content type. */
    type: 'output_text' | 'refusal';

    /** Text content. */
    text?: string | undefined;

    /** Refusal message. */
    refusal?: string | undefined;
}

/** Usage for Responses API. */
export interface ResponsesUsage {
    /** Input tokens. */
    inputTokens: number;

    /** Output tokens. */
    outputTokens: number;

    /** Total tokens. */
    totalTokens: number;

    /** Input tokens details. */
    inputTokensDetails?: {
        cachedTokens?: number | undefined;
    } | undefined;

    /** Output tokens details. */
    outputTokensDetails?: {
        reasoningTokens?: number | undefined;
    } | undefined;
}

/** Error in Responses API. */
export interface ResponsesError {
    /** Error type. */
    type: string;

    /** Error code. */
    code?: string | undefined;

    /** Error message. */
    message: string;
}

/** Details for incomplete responses. */
export interface IncompleteDetails {
    /** Reason for incompleteness. */
    reason: 'max_output_tokens' | 'content_filter';
}

// ============================================================================
// Thread Types (for stateful conversations)
// ============================================================================

/**
 * A conversation thread.
 */
export interface Thread {
    /** Thread ID. */
    id: string;

    /** Object type. */
    object: 'thread';

    /** Creation timestamp. */
    createdAt: number;

    /** Metadata. */
    metadata?: Record<string, string> | undefined;
}

/**
 * A message in a thread.
 */
export interface ThreadMessage {
    /** Message ID. */
    id: string;

    /** Object type. */
    object: 'thread.message';

    /** Thread this message belongs to. */
    threadId: string;

    /** Creation timestamp. */
    createdAt: number;

    /** Message role. */
    role: 'user' | 'assistant';

    /** Message content. */
    content: ThreadMessageContent[];

    /** Metadata. */
    metadata?: Record<string, string> | undefined;
}

/** Content in a thread message. */
export interface ThreadMessageContent {
    /** Content type. */
    type: 'text';

    /** Text content. */
    text: {
        value: string;
    };
}

// ============================================================================
// Conversion Functions
// ============================================================================

/**
 * Converts Responses API usage to standard Usage.
 */
export function responsesUsageToStandard(usage: ResponsesUsage): Usage {
    return {
        promptTokens: usage.inputTokens,
        completionTokens: usage.outputTokens,
        totalTokens: usage.totalTokens,
    };
}

/**
 * Converts standard Usage to Responses API usage.
 */
export function standardUsageToResponses(usage: Usage): ResponsesUsage {
    return {
        inputTokens: usage.promptTokens,
        outputTokens: usage.completionTokens,
        totalTokens: usage.totalTokens,
    };
}

/**
 * Converts Responses API input to messages.
 */
export function responsesInputToMessages(
    input: string | ResponsesInputItem[],
): Message[] {
    if (typeof input === 'string') {
        return [{ role: 'user', content: input }];
    }

    return input
        .filter((item): item is ResponsesInputItem & { type: 'message' } =>
            item.type === 'message'
        )
        .map((item) => {
            let content = '';
            if (typeof item.content === 'string') {
                content = item.content;
            } else if (Array.isArray(item.content)) {
                content = item.content
                    .filter((p) => p.type === 'input_text')
                    .map((p) => p.text ?? '')
                    .join('');
            }

            return {
                role: item.role ?? 'user',
                content,
            };
        });
}
