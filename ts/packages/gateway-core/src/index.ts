/**
 * Gateway Core - Runtime-agnostic LLM gateway library.
 *
 * @module index
 */

// Main Gateway
export { Gateway, type GatewayOptions } from './gateway.js';

// Router
export { Router, type Route, type ProviderSelection, stripAppPrefix, joinPath } from './router.js';

// Domain types
export * from './domain/index.js';

// Port interfaces
export * from './ports/index.js';

// Codecs
export * from './codecs/index.js';

// Providers
export * from './providers/index.js';

// Frontdoors
export * from './frontdoors/index.js';

// Middleware (pipeline stages)
export * from './middleware/index.js';

// HTTP Middleware
export * from './http/index.js';

// Recorder
export * from './recorder/index.js';

// Admin
export * from './admin/index.js';

// GraphQL
export * from './graphql/index.js';

// Responses API
export * from './responses/index.js';

// Shadow Mode
export * from './shadow/index.js';

// Utilities
export * from './utils/index.js';
