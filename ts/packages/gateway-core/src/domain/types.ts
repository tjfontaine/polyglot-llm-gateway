/**
 * Core domain types for the polyglot LLM gateway.
 * These are the canonical representations used throughout the system.
 *
 * @module domain/types
 */

// ============================================================================
// API Types
// ============================================================================

/** Identifies the API format for frontdoors and providers. */
export type APIType = 'openai' | 'anthropic' | 'responses';

// ============================================================================
// Message Types
// ============================================================================

/** Role of a message participant. */
export type MessageRole = 'system' | 'user' | 'assistant' | 'tool';

/**
 * Represents a chat message with support for both simple text
 * and multimodal content.
 */
export interface Message {
    /** The role of the message author. */
    role: MessageRole;

    /** Simple text content (for backward compatibility). */
    content: string;

    /** Optional name for the author. */
    name?: string | undefined;

    /**
     * Rich multimodal content (images, tool calls, etc.).
     * When set, this takes precedence over content field.
     */
    richContent?: MessageContent | undefined;

    /** Tool calls for assistant messages that invoke tools (OpenAI style). */
    toolCalls?: ToolCall[] | undefined;

    /** Tool call ID for tool messages providing results (OpenAI style). */
    toolCallId?: string | undefined;
}

/** Multimodal message content container. */
export interface MessageContent {
    /** Simple text content. */
    text?: string | undefined;

    /** Array of content parts for multimodal messages. */
    parts?: ContentPart[] | undefined;
}

/** Type of content part in a multimodal message. */
export type ContentPartType =
    | 'text'
    | 'image'
    | 'image_url'
    | 'tool_use'
    | 'tool_result';

/** A single part of multimodal content. */
export interface ContentPart {
    /** The type of this content part. */
    type: ContentPartType;

    /** Text content (for type='text'). */
    text?: string | undefined;

    /** Image source (for type='image'). */
    source?: ImageSource | undefined;

    /** Image URL reference (for type='image_url'). */
    imageUrl?: ImageURL | undefined;

    /** Tool use ID (for type='tool_use'). */
    id?: string | undefined;

    /** Tool name (for type='tool_use'). */
    name?: string | undefined;

    /** Tool input (for type='tool_use'). */
    input?: unknown;

    /** Tool use ID reference (for type='tool_result'). */
    toolUseId?: string | undefined;

    /** Tool result content (for type='tool_result'). */
    resultContent?: string | undefined;

    /** Whether the tool result is an error (for type='tool_result'). */
    isError?: boolean | undefined;
}

/** Image source for base64-encoded images. */
export interface ImageSource {
    /** Source type ('base64'). */
    type: 'base64';

    /** MIME type of the image. */
    mediaType: string;

    /** Base64-encoded image data. */
    data: string;
}

/** Image URL reference. */
export interface ImageURL {
    /** URL to the image. */
    url: string;

    /** Optional detail level for vision models. */
    detail?: 'auto' | 'low' | 'high' | undefined;
}

// ============================================================================
// Tool Types
// ============================================================================

/** Represents a tool call made by the assistant. */
export interface ToolCall {
    /** Unique identifier for this tool call. */
    id: string;

    /** Type of tool call (always 'function' currently). */
    type: 'function';

    /** Function details. */
    function: ToolCallFunction;
}

/** Function details in a tool call. */
export interface ToolCallFunction {
    /** Name of the function to call. */
    name: string;

    /** JSON string of function arguments. */
    arguments: string;
}

/** A streaming tool call chunk. */
export interface ToolCallChunk {
    /** Index of the tool call in the array. */
    index: number;

    /** Tool call ID (only in first chunk). */
    id?: string | undefined;

    /** Tool type (only in first chunk). */
    type?: string | undefined;

    /** Function details. */
    function?: {
        /** Function name (only in first chunk). */
        name?: string | undefined;

        /** Partial arguments string. */
        arguments?: string | undefined;
    } | undefined;
}

/** Definition of a tool that the model can call. */
export interface ToolDefinition {
    /** Tool name (required by OpenAI Responses API). */
    name?: string | undefined;

    /** Type of tool (always 'function'). */
    type: 'function';

    /** Function definition. */
    function: FunctionDef;
}

/** Function definition for a tool. */
export interface FunctionDef {
    /** Name of the function. */
    name: string;

    /** Description of what the function does. */
    description?: string | undefined;

    /** JSON Schema for the function parameters. */
    parameters: Record<string, unknown>;
}

/** Tool choice specification. */
export type ToolChoice =
    | 'auto'
    | 'none'
    | 'required'
    | { type: 'function'; function: { name: string } };

// ============================================================================
// Request/Response Types
// ============================================================================

/** Response format specification. */
export interface ResponseFormat {
    /** Format type. */
    type: 'text' | 'json_object' | 'json_schema';

    /** JSON Schema (for type='json_schema'). */
    jsonSchema?: Record<string, unknown> | undefined;
}

/**
 * Canonical request format - superset of all supported features.
 * This is the internal representation used throughout the gateway.
 */
export interface CanonicalRequest {
    /** Tenant ID for multi-tenant mode. */
    tenantId: string;

    /** Model identifier. */
    model: string;

    /** Conversation messages. */
    messages: Message[];

    /** Whether to stream the response. */
    stream: boolean;

    /** Maximum tokens to generate. */
    maxTokens?: number | undefined;

    /** Sampling temperature (0-2). */
    temperature?: number | undefined;

    /** Top-p (nucleus) sampling. */
    topP?: number | undefined;

    /** Tools the model can use. */
    tools?: ToolDefinition[] | undefined;

    /** How the model should choose tools. */
    toolChoice?: ToolChoice | undefined;

    /** Arbitrary metadata. */
    metadata?: Record<string, string> | undefined;

    /** System prompt (for APIs that support separate system instructions). */
    systemPrompt?: string | undefined;

    /** Response format configuration. */
    responseFormat?: ResponseFormat | undefined;

    /** Stop sequences. */
    stop?: string[] | undefined;

    /** Instructions (Responses API - overrides system prompt). */
    instructions?: string | undefined;

    /** Previous response ID (Responses API - for continuation). */
    previousResponseId?: string | undefined;

    /** User-Agent header from incoming request. */
    userAgent?: string | undefined;

    /** Original API format of the incoming request. */
    sourceAPIType: APIType;

    /** Original request body for pass-through mode. */
    rawRequest?: Uint8Array | undefined;
}

/** Token usage statistics. */
export interface Usage {
    /** Number of tokens in the prompt. */
    promptTokens: number;

    /** Number of tokens in the completion. */
    completionTokens: number;

    /** Total tokens used. */
    totalTokens: number;
}

/** A single completion choice. */
export interface Choice {
    /** Index of this choice. */
    index: number;

    /** The generated message. */
    message: Message;

    /** Why generation stopped. */
    finishReason: FinishReason | null;

    /** Log probabilities (if requested). */
    logprobs?: unknown;
}

/** Reason why generation stopped. */
export type FinishReason =
    | 'stop'
    | 'length'
    | 'tool_calls'
    | 'content_filter';

/** Rate limit information from upstream providers. */
export interface RateLimitInfo {
    /** Request rate limit. */
    requestsLimit?: number | undefined;

    /** Remaining requests. */
    requestsRemaining?: number | undefined;

    /** When request limit resets. */
    requestsReset?: string | undefined;

    /** Token rate limit. */
    tokensLimit?: number | undefined;

    /** Remaining tokens. */
    tokensRemaining?: number | undefined;

    /** When token limit resets. */
    tokensReset?: string | undefined;
}

/**
 * Canonical response format.
 * Normalized representation of an LLM response.
 */
export interface CanonicalResponse {
    /** Unique response identifier. */
    id: string;

    /** Object type (e.g., 'chat.completion'). */
    object: string;

    /** Unix timestamp of creation. */
    created: number;

    /** Model that generated the response. */
    model: string;

    /** Generated choices. */
    choices: Choice[];

    /** Token usage statistics. */
    usage: Usage;

    /** API type of the provider that generated this. */
    sourceAPIType: APIType;

    /** Original response body for pass-through mode. */
    rawResponse?: Uint8Array | undefined;

    /** System fingerprint (OpenAI specific). */
    systemFingerprint?: string | undefined;

    /** Rate limit info from upstream. */
    rateLimits?: RateLimitInfo | undefined;

    /** Actual model used by provider (for logging when model is rewritten). */
    providerModel?: string | undefined;

    /** Request body sent to provider (for debugging). */
    providerRequestBody?: Uint8Array | undefined;
}

// ============================================================================
// Streaming Event Types
// ============================================================================

/** Type of streaming event. */
export type StreamEventType =
    // Generic events
    | 'content_delta'
    | 'content_done'
    | 'error'
    | 'done'
    // Message lifecycle events
    | 'message_start'
    | 'message_delta'
    | 'message_stop'
    // Content block events (Anthropic style)
    | 'content_block_start'
    | 'content_block_delta'
    | 'content_block_stop'
    // Responses API events
    | 'response.created'
    | 'response.in_progress'
    | 'response.output_item.added'
    | 'response.output_item.delta'
    | 'response.output_item.done'
    | 'response.completed'
    | 'response.failed'
    | 'response.done';

/**
 * Canonical streaming event.
 * Represents a single event in a streaming response.
 */
export interface CanonicalEvent {
    /** Event type. */
    type: StreamEventType;

    /** Message role (for message start events). */
    role?: string | undefined;

    /** Text content delta. */
    contentDelta?: string | undefined;

    /** Tool call chunk. */
    toolCall?: ToolCallChunk | undefined;

    /** Token usage update. */
    usage?: Usage | undefined;

    /** Error (for error events). */
    error?: Error | undefined;

    /** Content block index. */
    index?: number | undefined;

    /** Content block (for block start events). */
    contentBlock?: ContentPart | undefined;

    /** Finish reason (for completion events). */
    finishReason?: string | undefined;

    /** Response ID (for response events). */
    responseId?: string | undefined;

    /** Model (may be rewritten for client display). */
    model?: string | undefined;

    /** Actual provider model (for logging). */
    providerModel?: string | undefined;

    /** Raw event data for pass-through mode. */
    rawEvent?: Uint8Array | undefined;
}

// ============================================================================
// Model Types
// ============================================================================

/** Model information. */
export interface Model {
    /** Model identifier. */
    id: string;

    /** Object type (e.g., 'model'). */
    object?: string | undefined;

    /** Model owner/creator. */
    ownedBy?: string | undefined;

    /** Creation timestamp. */
    created?: number | undefined;
}

/** List of models. */
export interface ModelList {
    /** Object type. */
    object: string;

    /** Array of models. */
    data: Model[];
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Returns the text content of a message.
 * If richContent is set, returns concatenated text parts.
 */
export function getMessageContent(message: Message): string {
    if (message.richContent) {
        if (message.richContent.text) {
            return message.richContent.text;
        }
        if (message.richContent.parts) {
            return message.richContent.parts
                .filter((p) => p.type === 'text')
                .map((p) => p.text ?? '')
                .join('');
        }
        return '';
    }
    return message.content;
}

/**
 * Returns true if the message has multimodal content.
 */
export function hasRichContent(message: Message): boolean {
    if (!message.richContent) return false;
    if (!message.richContent.parts) return false;
    return message.richContent.parts.some((p) => p.type !== 'text');
}

/**
 * Deep clones a CanonicalRequest.
 */
export function cloneRequest(req: CanonicalRequest): CanonicalRequest {
    return {
        ...req,
        messages: req.messages.map((m) => ({
            ...m,
            toolCalls: m.toolCalls ? [...m.toolCalls] : undefined,
            richContent: m.richContent
                ? {
                    ...m.richContent,
                    parts: m.richContent.parts ? [...m.richContent.parts] : undefined,
                }
                : undefined,
        })),
        tools: req.tools ? [...req.tools] : undefined,
        metadata: req.metadata ? { ...req.metadata } : undefined,
        stop: req.stop ? [...req.stop] : undefined,
        rawRequest: req.rawRequest ? new Uint8Array(req.rawRequest) : undefined,
    };
}
