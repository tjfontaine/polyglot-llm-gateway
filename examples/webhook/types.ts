/**
 * TypeScript types for the Gateway IR (Intermediate Representation) schema.
 * 
 * These types match the Go domain types in internal/core/domain/types.go
 * Use these when building webhook pipeline stages.
 * 
 * @example
 * ```typescript
 * import type { StageInput, StageOutput, CanonicalRequest, CanonicalResponse } from "./types";
 * 
 * Deno.serve(async (req) => {
 *   const input: StageInput = await req.json();
 *   
 *   if (input.phase === "request") {
 *     // Mutate request
 *     const mutatedRequest: CanonicalRequest = {
 *       ...input.request,
 *       model: "gpt-4o"  // Force model upgrade
 *     };
 *     return Response.json({ action: "mutate", request: mutatedRequest });
 *   }
 *   
 *   return Response.json({ action: "allow" });
 * });
 * ```
 */

// =============================================================================
// Message Types
// =============================================================================

/** Represents a chat message with support for both simple text and multimodal content. */
export interface Message {
    /** Message role: "system", "user", "assistant", or "tool" */
    role: string;

    /** Simple text content (for backward compatibility) */
    content: string;

    /** Optional name for the message author */
    name?: string;

    /** Multimodal content (images, tool calls, etc.) - takes precedence over content */
    rich_content?: MessageContent;

    /** Tool calls for assistant messages that invoke tools */
    tool_calls?: ToolCall[];

    /** Tool call ID for tool messages providing results */
    tool_call_id?: string;
}

/** Multimodal message content */
export interface MessageContent {
    /** Simple text (when content is just text) */
    text?: string;

    /** Content parts for multimodal messages */
    parts?: ContentPart[];
}

/** A single content part in a multimodal message */
export interface ContentPart {
    /** Part type: "text", "image_url", "tool_use", "tool_result" */
    type: string;

    /** Text content (when type is "text") */
    text?: string;

    /** Image URL (when type is "image_url") */
    image_url?: {
        url: string;
        detail?: "auto" | "low" | "high";
    };

    /** Tool use details (when type is "tool_use") */
    id?: string;
    name?: string;
    input?: unknown;

    /** Tool result (when type is "tool_result") */
    tool_use_id?: string;
    content?: string;
    is_error?: boolean;
}

/** Represents a tool call made by the assistant */
export interface ToolCall {
    /** Unique identifier for this tool call */
    id: string;

    /** Type of tool call (always "function") */
    type: string;

    /** Function details */
    function: ToolCallFunction;
}

/** Function details in a tool call */
export interface ToolCallFunction {
    /** Function name */
    name: string;

    /** Function arguments as JSON string */
    arguments: string;
}

// =============================================================================
// Tool Definitions
// =============================================================================

/** Represents a tool that the model can call */
export interface ToolDefinition {
    /** Tool identifier */
    name?: string;

    /** Tool type (always "function") */
    type: string;

    /** Function definition */
    function: FunctionDef;
}

/** Describes a function signature */
export interface FunctionDef {
    /** Function name */
    name: string;

    /** Optional description */
    description?: string;

    /** Parameters as JSON Schema */
    parameters: unknown;
}

// =============================================================================
// Request Types (IR Schema)
// =============================================================================

/** Response format configuration */
export interface ResponseFormat {
    /** Format type: "text", "json_object", or "json_schema" */
    type: string;

    /** JSON Schema (when type is "json_schema") */
    json_schema?: unknown;
}

/**
 * Canonical request - the superset IR representation of all LLM API requests.
 * This is the format used for all pipeline transformations.
 */
export interface CanonicalRequest {
    /** Tenant identifier */
    tenant_id?: string;

    /** Model to use (e.g., "gpt-4o", "claude-3-sonnet") */
    model: string;

    /** Conversation messages */
    messages: Message[];

    /** Whether to stream the response */
    stream?: boolean;

    /** Maximum tokens to generate */
    max_tokens?: number;

    /** Sampling temperature (0-2) */
    temperature?: number;

    /** Top-p sampling */
    top_p?: number;

    /** Available tools */
    tools?: ToolDefinition[];

    /** Tool choice: "auto", "none", "required", or specific tool */
    tool_choice?: string | { type: string; function?: { name: string } };

    /** Request metadata */
    metadata?: Record<string, string>;

    /** System prompt (for APIs with separate system instructions) */
    system_prompt?: string;

    /** Response format configuration */
    response_format?: ResponseFormat;

    /** Stop sequences */
    stop?: string[];

    /** Instructions override (Responses API) */
    instructions?: string;

    /** Continue from previous response (Responses API) */
    previous_response_id?: string;
}

// =============================================================================
// Response Types (IR Schema)
// =============================================================================

/** Token usage statistics */
export interface Usage {
    /** Tokens in the prompt */
    prompt_tokens: number;

    /** Tokens in the completion */
    completion_tokens: number;

    /** Total tokens (prompt + completion) */
    total_tokens: number;
}

/** A single completion choice */
export interface Choice {
    /** Choice index */
    index: number;

    /** The response message */
    message: Message;

    /** Why generation stopped: "stop", "length", "tool_calls", "content_filter" */
    finish_reason: string;

    /** Log probabilities (if requested) */
    logprobs?: unknown;
}

/**
 * Canonical response - the IR representation of all LLM API responses.
 * This is the format used for all pipeline transformations.
 */
export interface CanonicalResponse {
    /** Response ID */
    id: string;

    /** Object type (e.g., "chat.completion") */
    object?: string;

    /** Creation timestamp (Unix epoch) */
    created?: number;

    /** Model that generated the response */
    model: string;

    /** Completion choices */
    choices: Choice[];

    /** Token usage statistics */
    usage?: Usage;

    /** System fingerprint (OpenAI specific) */
    system_fingerprint?: string;
}

// =============================================================================
// Pipeline Stage Types
// =============================================================================

/** Pipeline stage action */
export type StageAction = "allow" | "deny" | "mutate";

/** Metadata passed to pipeline stages */
export interface StageMetadata {
    /** Application name */
    app_name: string;

    /** Request ID */
    request_id: string;

    /** Interaction ID */
    interaction_id: string;

    /** ISO 8601 timestamp */
    timestamp: string;

    /** Additional metadata */
    [key: string]: unknown;
}

/**
 * Input to a pipeline stage.
 * This is the JSON body your webhook receives.
 */
export interface StageInput {
    /** Phase: "request" for pre-pipeline, "response" for post-pipeline */
    phase: "request" | "response";

    /** The canonical request (always present) */
    request: CanonicalRequest;

    /** The canonical response (only present in "response" phase) */
    response?: CanonicalResponse;

    /** Contextual metadata */
    metadata: StageMetadata;
}

/**
 * Output from a pipeline stage.
 * This is the JSON your webhook must return.
 */
export interface StageOutput {
    /** Action to take: "allow", "deny", or "mutate" */
    action: StageAction;

    /** 
     * Mutated request (only for "mutate" action in "request" phase).
     * The gateway will use this request instead of the original.
     */
    request?: CanonicalRequest;

    /** 
     * Mutated response (only for "mutate" action in "response" phase).
     * The gateway will return this response instead of the original.
     */
    response?: CanonicalResponse;

    /** Reason for denial (only for "deny" action) */
    deny_reason?: string;
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Create an allow response.
 * Use when the request/response should proceed unchanged.
 */
export function allow(): StageOutput {
    return { action: "allow" };
}

/**
 * Create a deny response.
 * Use to block a request or squelch a response.
 */
export function deny(reason: string): StageOutput {
    return { action: "deny", deny_reason: reason };
}

/**
 * Create a mutate response for request phase.
 * Use to modify the request before it reaches the LLM provider.
 */
export function mutateRequest(request: CanonicalRequest): StageOutput {
    return { action: "mutate", request };
}

/**
 * Create a mutate response for response phase.
 * Use to modify the response before it reaches the client.
 */
export function mutateResponse(response: CanonicalResponse): StageOutput {
    return { action: "mutate", response };
}
