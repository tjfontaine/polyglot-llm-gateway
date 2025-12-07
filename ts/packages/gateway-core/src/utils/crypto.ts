/**
 * Crypto utilities using Web Crypto API.
 *
 * @module utils/crypto
 */

// ============================================================================
// Hashing
// ============================================================================

/**
 * Hashes a string using SHA-256.
 */
export async function sha256(input: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(input);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = new Uint8Array(hashBuffer);
    return arrayToHex(hashArray);
}

/**
 * Hashes a string using SHA-256 with a salt.
 */
export async function sha256WithSalt(input: string, salt: string): Promise<string> {
    return sha256(salt + input);
}

// ============================================================================
// UUID Generation
// ============================================================================

/**
 * Generates a random UUID.
 */
export function randomUUID(): string {
    return crypto.randomUUID();
}

// ============================================================================
// Random Values
// ============================================================================

/**
 * Generates random bytes.
 */
export function randomBytes(length: number): Uint8Array {
    const bytes = new Uint8Array(length);
    crypto.getRandomValues(bytes);
    return bytes;
}

/**
 * Generates a random hex string.
 */
export function randomHex(length: number): string {
    const bytes = randomBytes(Math.ceil(length / 2));
    return arrayToHex(bytes).slice(0, length);
}

// ============================================================================
// Encoding Utilities
// ============================================================================

/**
 * Converts a Uint8Array to a hex string.
 */
export function arrayToHex(array: Uint8Array): string {
    return Array.from(array)
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('');
}

/**
 * Converts a hex string to a Uint8Array.
 */
export function hexToArray(hex: string): Uint8Array {
    const matches = hex.match(/.{1,2}/g);
    if (!matches) return new Uint8Array();
    return new Uint8Array(matches.map((byte) => parseInt(byte, 16)));
}

/**
 * Encodes a string to base64.
 */
export function toBase64(input: string): string {
    const encoder = new TextEncoder();
    const bytes = encoder.encode(input);
    return bytesToBase64(bytes);
}

/**
 * Decodes a base64 string.
 */
export function fromBase64(input: string): string {
    const bytes = base64ToBytes(input);
    const decoder = new TextDecoder();
    return decoder.decode(bytes);
}

/**
 * Encodes bytes to base64.
 */
export function bytesToBase64(bytes: Uint8Array): string {
    // Use built-in btoa if available (works in Workers and browsers)
    const binary = Array.from(bytes)
        .map((b) => String.fromCharCode(b))
        .join('');
    return btoa(binary);
}

/**
 * Decodes base64 to bytes.
 */
export function base64ToBytes(base64: string): Uint8Array {
    const binary = atob(base64);
    return new Uint8Array(binary.split('').map((c) => c.charCodeAt(0)));
}

// ============================================================================
// Timing-Safe Comparison
// ============================================================================

/**
 * Compares two strings in constant time.
 * Prevents timing attacks.
 */
export function timingSafeEqual(a: string, b: string): boolean {
    if (a.length !== b.length) return false;

    let result = 0;
    for (let i = 0; i < a.length; i++) {
        result |= a.charCodeAt(i) ^ b.charCodeAt(i);
    }
    return result === 0;
}

/**
 * Compares two Uint8Arrays in constant time.
 */
export function timingSafeEqualBytes(a: Uint8Array, b: Uint8Array): boolean {
    if (a.length !== b.length) return false;

    let result = 0;
    for (let i = 0; i < a.length; i++) {
        result |= (a[i] ?? 0) ^ (b[i] ?? 0);
    }
    return result === 0;
}
