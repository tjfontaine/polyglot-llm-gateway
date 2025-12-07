/**
 * Middleware module exports.
 *
 * @module middleware
 */

// Types
export type {
    PipelineContext,
    StepResult,
    MiddlewareStep,
    StageConfig,
    TransformStepConfig,
    RateLimitStepConfig,
    ContentFilterStepConfig,
    WebhookStepConfig,
    LogStepConfig,
    StepConfig,
    MiddlewarePipelineConfig,
} from './types.js';

export {
    continueResult,
    modifyResult,
    denyResult,
    respondResult,
} from './types.js';

// Executor
export {
    PipelineExecutor,
    createExecutor,
    type ExecutorOptions,
    type ExecutionResult,
} from './executor.js';

// Built-in steps
export {
    createWebhookStep,
    createTransformStep,
    createContentFilterStep,
    createLogStep,
} from './steps/index.js';
