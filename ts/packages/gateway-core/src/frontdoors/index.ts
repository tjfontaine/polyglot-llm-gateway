/**
 * Frontdoors module exports.
 *
 * @module frontdoors
 */

// Types
export type {
    Frontdoor,
    FrontdoorContext,
    FrontdoorResponse,
    FrontdoorRegistry,
} from './types.js';
export { createFrontdoorRegistry } from './types.js';

// OpenAI
export { openAIFrontdoor } from './openai.js';

// Anthropic
export { anthropicFrontdoor } from './anthropic.js';

// Responses
export { responsesFrontdoor } from './responses.js';

/**
 * Creates a default frontdoor registry with all built-in frontdoors.
 */
import { createFrontdoorRegistry as createRegistry } from './types.js';
import { openAIFrontdoor } from './openai.js';
import { anthropicFrontdoor } from './anthropic.js';
import { responsesFrontdoor } from './responses.js';

export const defaultFrontdoorRegistry = (() => {
    const registry = createRegistry();
    registry.register(openAIFrontdoor);
    registry.register(anthropicFrontdoor);
    registry.register(responsesFrontdoor);
    return registry;
})();
