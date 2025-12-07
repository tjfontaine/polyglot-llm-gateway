/**
 * Providers module exports.
 *
 * @module providers
 */

// OpenAI
export { OpenAIProvider, createOpenAIProvider } from './openai.js';

// Anthropic
export { AnthropicProvider, createAnthropicProvider } from './anthropic.js';

// Passthrough
export { PassthroughProvider, withPassthrough } from './passthrough.js';
export type { PassthroughOptions, PassthroughableProvider } from './passthrough.js';

// Default registry with built-in providers
import { createProviderRegistry } from '../ports/provider.js';
import { createOpenAIProvider } from './openai.js';
import { createAnthropicProvider } from './anthropic.js';

/**
 * Default provider registry with OpenAI and Anthropic providers.
 */
export const defaultProviderRegistry = createProviderRegistry();
defaultProviderRegistry.register('openai', createOpenAIProvider);
defaultProviderRegistry.register('anthropic', createAnthropicProvider);
