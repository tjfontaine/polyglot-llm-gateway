/**
 * Utils module exports.
 *
 * @module utils
 */

// Streaming
export {
    createSSEStream,
    createAnthropicSSEStream,
    sseHeaders,
    sseResponse,
    collectEvents,
    arrayToGenerator,
    transformEvents,
    createStreamAccumulator,
    accumulateEvent,
    type StreamAccumulator,
} from './streaming.js';

// Crypto
export {
    sha256,
    sha256WithSalt,
    randomUUID,
    randomBytes,
    randomHex,
    arrayToHex,
    hexToArray,
    toBase64,
    fromBase64,
    bytesToBase64,
    base64ToBytes,
    timingSafeEqual,
    timingSafeEqualBytes,
} from './crypto.js';

// Logging
export {
    type LogLevel,
    type Logger,
    ConsoleLogger,
    NullLogger,
    defaultLogger,
    requestLogger,
} from './logging.js';
