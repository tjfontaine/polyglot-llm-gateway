/**
 * Codec interface and types.
 *
 * @module codecs/types
 */

import type {
    APIType,
    CanonicalRequest,
    CanonicalResponse,
    CanonicalEvent,
} from '../domain/types.js';

// ============================================================================
// Codec Interface
// ============================================================================

/**
 * A codec translates between API-specific formats and canonical types.
 */
export interface Codec {
    /** Codec name. */
    readonly name: string;

    /** API type this codec handles. */
    readonly apiType: APIType;

    // ---- Request handling ----

    /**
     * Decodes an API request body to canonical format.
     */
    decodeRequest(body: Uint8Array | string): CanonicalRequest;

    /**
     * Encodes a canonical request to API format.
     */
    encodeRequest(request: CanonicalRequest): Uint8Array;

    // ---- Response handling ----

    /**
     * Decodes an API response body to canonical format.
     */
    decodeResponse(body: Uint8Array | string): CanonicalResponse;

    /**
     * Encodes a canonical response to API format.
     */
    encodeResponse(response: CanonicalResponse): Uint8Array;

    // ---- Streaming ----

    /**
     * Decodes a streaming chunk to a canonical event.
     */
    decodeStreamChunk(chunk: string): CanonicalEvent | null;

    /**
     * Encodes a canonical event to a streaming chunk.
     */
    encodeStreamEvent(event: CanonicalEvent, metadata?: StreamMetadata): string;

    // ---- Errors ----

    /**
     * Decodes an error response to an Error object.
     */
    decodeError(body: Uint8Array | string, status: number): Error;

    /**
     * Encodes an error to API format.
     */
    encodeError(error: Error): { body: string; status: number };
}

/**
 * Metadata for streaming responses.
 */
export interface StreamMetadata {
    /** Response ID. */
    id?: string | undefined;

    /** Model name. */
    model?: string | undefined;

    /** Creation timestamp. */
    created?: number | undefined;

    /** Object type. */
    object?: string | undefined;

    /** System fingerprint. */
    systemFingerprint?: string | undefined;
}

// ============================================================================
// Codec Registry
// ============================================================================

/**
 * Registry of codecs by API type.
 */
export interface CodecRegistry {
    /**
     * Registers a codec.
     */
    register(codec: Codec): void;

    /**
     * Gets a codec by API type.
     */
    get(apiType: APIType): Codec | undefined;

    /**
     * Checks if a codec is registered.
     */
    has(apiType: APIType): boolean;

    /**
     * Lists registered API types.
     */
    list(): APIType[];
}

/**
 * Creates a new codec registry.
 */
export function createCodecRegistry(): CodecRegistry {
    const codecs = new Map<APIType, Codec>();

    return {
        register(codec: Codec): void {
            codecs.set(codec.apiType, codec);
        },

        get(apiType: APIType): Codec | undefined {
            return codecs.get(apiType);
        },

        has(apiType: APIType): boolean {
            return codecs.has(apiType);
        },

        list(): APIType[] {
            return Array.from(codecs.keys());
        },
    };
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Converts a string or Uint8Array to string.
 */
export function toText(input: Uint8Array | string): string {
    if (typeof input === 'string') return input;
    return new TextDecoder().decode(input);
}

/**
 * Converts a string to Uint8Array.
 */
export function toBytes(input: string): Uint8Array {
    return new TextEncoder().encode(input);
}

/**
 * Safely parses JSON, returning null on error.
 */
export function safeParseJSON<T>(input: string): T | null {
    try {
        return JSON.parse(input) as T;
    } catch {
        return null;
    }
}
