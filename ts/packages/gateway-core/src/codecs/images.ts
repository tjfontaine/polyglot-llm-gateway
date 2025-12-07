/**
 * Image fetcher for converting remote images to base64.
 *
 * Required for OpenAI→Anthropic translation where image URLs
 * need to be converted to base64 format.
 *
 * @module codec/images
 */

// ============================================================================
// Types
// ============================================================================

/**
 * Base64 image source for Anthropic API.
 * Named Base64ImageSource to avoid conflict with domain ImageSource.
 */
export interface Base64ImageSource {
    /** Source type. */
    type: 'base64';

    /** Media type (e.g., image/jpeg). */
    mediaType: string;

    /** Base64 encoded image data. */
    data: string;
}

/**
 * Image fetcher options.
 */
export interface ImageFetcherOptions {
    /** Custom fetch function. */
    fetch?: typeof fetch | undefined;

    /** Maximum allowed image size in bytes (default: 20MB). */
    maxSize?: number | undefined;

    /** Request timeout in milliseconds (default: 10000). */
    timeoutMs?: number | undefined;
}

// ============================================================================
// Image Fetcher
// ============================================================================

/**
 * Default max image size (20MB).
 */
const DEFAULT_MAX_SIZE = 20 * 1024 * 1024;

/**
 * Default request timeout (10 seconds).
 */
const DEFAULT_TIMEOUT_MS = 10000;

/**
 * Supported media types for images.
 */
const SUPPORTED_MEDIA_TYPES = new Set([
    'image/jpeg',
    'image/jpg',
    'image/png',
    'image/gif',
    'image/webp',
]);

/**
 * Fetches and converts images to base64 format.
 */
export class ImageFetcher {
    private readonly fetchFn: typeof fetch;
    private readonly maxSize: number;
    private readonly timeoutMs: number;

    constructor(options: ImageFetcherOptions = {}) {
        this.fetchFn = options.fetch ?? globalThis.fetch.bind(globalThis);
        this.maxSize = options.maxSize ?? DEFAULT_MAX_SIZE;
        this.timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    }

    /**
     * Fetches an image from URL and converts to base64.
     */
    async fetchAndConvert(url: string): Promise<Base64ImageSource> {
        // Handle data URLs
        if (url.startsWith('data:')) {
            return this.parseDataUrl(url);
        }

        // Validate URL scheme
        if (!url.startsWith('http://') && !url.startsWith('https://')) {
            throw new Error('Unsupported URL scheme: must be http:// or https://');
        }

        // Fetch with timeout
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.timeoutMs);

        try {
            const response = await this.fetchFn(url, {
                signal: controller.signal,
            });

            if (!response.ok) {
                throw new Error(`Failed to fetch image: status ${response.status}`);
            }

            // Check content length if available
            const contentLength = response.headers.get('content-length');
            if (contentLength && parseInt(contentLength, 10) > this.maxSize) {
                throw new Error(
                    `Image too large: ${contentLength} bytes (max ${this.maxSize})`,
                );
            }

            // Determine media type
            let mediaType = response.headers.get('content-type') ?? '';
            if (!mediaType) {
                mediaType = this.inferMediaType(url);
            }

            // Validate media type
            const normalizedType = this.normalizeMediaType(mediaType);
            if (!SUPPORTED_MEDIA_TYPES.has(normalizedType)) {
                throw new Error(`Unsupported media type: ${mediaType}`);
            }

            // Read and check size
            const buffer = await response.arrayBuffer();
            if (buffer.byteLength > this.maxSize) {
                throw new Error(`Image too large: exceeds ${this.maxSize} bytes`);
            }

            // Encode to base64
            const encoded = this.arrayBufferToBase64(buffer);

            return {
                type: 'base64',
                mediaType: normalizedType,
                data: encoded,
            };
        } finally {
            clearTimeout(timeoutId);
        }
    }

    /**
     * Parses a data URL to extract base64 content.
     */
    private parseDataUrl(url: string): Base64ImageSource {
        // Format: data:image/jpeg;base64,/9j/4AAQSkZ...
        if (!url.startsWith('data:')) {
            throw new Error('Not a data URL');
        }

        const content = url.slice(5);
        const commaIdx = content.indexOf(',');
        if (commaIdx === -1) {
            throw new Error('Invalid data URL: missing comma separator');
        }

        const metadata = content.slice(0, commaIdx);
        const data = content.slice(commaIdx + 1);

        const parts = metadata.split(';');
        if (parts.length === 0) {
            throw new Error('Invalid data URL: missing media type');
        }

        const mediaType = this.normalizeMediaType(parts[0]!);
        if (!SUPPORTED_MEDIA_TYPES.has(mediaType)) {
            throw new Error(`Unsupported media type: ${parts[0]}`);
        }

        // Check for base64 encoding
        if (!parts.includes('base64')) {
            throw new Error('Data URL must be base64 encoded');
        }

        return {
            type: 'base64',
            mediaType,
            data,
        };
    }

    /**
     * Infers media type from URL extension.
     */
    private inferMediaType(url: string): string {
        const lower = url.toLowerCase();

        if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) {
            return 'image/jpeg';
        }
        if (lower.endsWith('.png')) {
            return 'image/png';
        }
        if (lower.endsWith('.gif')) {
            return 'image/gif';
        }
        if (lower.endsWith('.webp')) {
            return 'image/webp';
        }

        return 'image/jpeg'; // Default assumption
    }

    /**
     * Normalizes media type.
     */
    private normalizeMediaType(mediaType: string): string {
        const main = mediaType.split(';')[0]?.trim().toLowerCase() ?? '';

        // Normalize image/jpg to image/jpeg
        if (main === 'image/jpg') {
            return 'image/jpeg';
        }

        return main;
    }

    /**
     * Converts ArrayBuffer to base64 string.
     */
    private arrayBufferToBase64(buffer: ArrayBuffer): string {
        const bytes = new Uint8Array(buffer);

        // For environments with Buffer (Node.js)
        if (typeof Buffer !== 'undefined') {
            return Buffer.from(bytes).toString('base64');
        }

        // For browser/Cloudflare Workers
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]!);
        }
        return btoa(binary);
    }
}

/**
 * Creates an image fetcher with default options.
 */
export function createImageFetcher(options?: ImageFetcherOptions): ImageFetcher {
    return new ImageFetcher(options);
}
