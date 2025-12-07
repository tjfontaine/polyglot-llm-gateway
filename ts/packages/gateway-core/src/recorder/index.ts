/**
 * Recorder module exports.
 *
 * @module recorder
 */

export {
    // Types
    type InteractionStatus,
    type RecordInteractionParams,
    type Interaction,
    type InteractionRequest,
    type InteractionResponse,
    type InteractionError,
    type TransformationStep,

    // Recorder
    InteractionRecorder,
    type InteractionRecorderOptions,

    // Helpers
    extractRelevantHeaders,
} from './interaction.js';
