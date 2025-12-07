/**
 * Codecs module exports.
 *
 * @module codecs
 */

// Types
export type { Codec, StreamMetadata, CodecRegistry } from './types.js';
export { createCodecRegistry, toText, toBytes, safeParseJSON } from './types.js';

// OpenAI
export { OpenAICodec, openaiCodec } from './openai.js';

// Anthropic
export { AnthropicCodec, anthropicCodec } from './anthropic.js';

// Images
export {
    ImageFetcher,
    createImageFetcher,
    type Base64ImageSource,
    type ImageFetcherOptions,
} from './images.js';

// Default registry with built-in codecs
import { createCodecRegistry } from './types.js';
import { openaiCodec } from './openai.js';
import { anthropicCodec } from './anthropic.js';

/**
 * Default codec registry with OpenAI and Anthropic codecs.
 */
export const defaultCodecRegistry = createCodecRegistry();
defaultCodecRegistry.register(openaiCodec);
defaultCodecRegistry.register(anthropicCodec);
