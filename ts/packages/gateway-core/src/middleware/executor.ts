/**
 * Middleware executor - runs the pipeline.
 *
 * @module middleware/executor
 */

import type { CanonicalRequest, CanonicalResponse } from '../domain/types.js';
import { APIError, errServer } from '../domain/errors.js';
import type { Logger } from '../utils/logging.js';
import type {
    PipelineContext,
    StageConfig,
    StepResult,
    MiddlewareStep,
} from './types.js';

// ============================================================================
// Pipeline Executor
// ============================================================================

/**
 * Executor options.
 */
export interface ExecutorOptions {
    /** Logger. */
    logger?: Logger | undefined;

    /** Default timeout for steps (ms). */
    defaultTimeoutMs?: number | undefined;

    /** Default error handling mode. */
    defaultOnError?: 'allow' | 'deny' | undefined;
}

/**
 * Execution result.
 */
export interface ExecutionResult {
    /** Whether to continue with the request. */
    continue: boolean;

    /** Modified request (if any). */
    request?: CanonicalRequest | undefined;

    /** Early response (if pipeline responded). */
    response?: CanonicalResponse | undefined;

    /** Deny reason (if denied). */
    denyReason?: string | undefined;

    /** Deny status code. */
    denyStatusCode?: number | undefined;
}

/**
 * Pipeline executor.
 */
export class PipelineExecutor {
    private readonly preStages: StageConfig[] = [];
    private readonly postStages: StageConfig[] = [];
    private readonly logger?: Logger;
    private readonly defaultTimeoutMs: number;
    private readonly defaultOnError: 'allow' | 'deny';

    constructor(options?: ExecutorOptions) {
        this.logger = options?.logger;
        this.defaultTimeoutMs = options?.defaultTimeoutMs ?? 30000;
        this.defaultOnError = options?.defaultOnError ?? 'deny';
    }

    /**
     * Adds a pre-request stage.
     */
    addPreStage(stage: StageConfig): void {
        this.preStages.push({ ...stage, type: 'pre' });
        this.sortStages();
    }

    /**
     * Adds a post-request stage.
     */
    addPostStage(stage: StageConfig): void {
        this.postStages.push({ ...stage, type: 'post' });
        this.sortStages();
    }

    /**
     * Runs the pre-request pipeline.
     */
    async runPre(ctx: PipelineContext): Promise<ExecutionResult> {
        return this.runStages(this.preStages, ctx);
    }

    /**
     * Runs the post-request pipeline.
     */
    async runPost(ctx: PipelineContext): Promise<ExecutionResult> {
        return this.runStages(this.postStages, ctx);
    }

    /**
     * Runs a list of stages.
     */
    private async runStages(
        stages: StageConfig[],
        ctx: PipelineContext,
    ): Promise<ExecutionResult> {
        for (const stage of stages) {
            const result = await this.runStage(stage, ctx);

            switch (result.action) {
                case 'continue':
                    // Continue to next stage
                    break;

                case 'modify':
                    // Update context with modifications
                    if (result.request) {
                        ctx.request = result.request;
                    }
                    if (result.response) {
                        ctx.response = result.response;
                    }
                    break;

                case 'deny':
                    return {
                        continue: false,
                        denyReason: result.reason,
                        denyStatusCode: result.statusCode ?? 403,
                    };

                case 'respond':
                    return {
                        continue: false,
                        response: result.response,
                    };
            }
        }

        return {
            continue: true,
            request: ctx.request,
            response: ctx.response,
        };
    }

    /**
     * Runs a single stage with timeout and error handling.
     */
    private async runStage(
        stage: StageConfig,
        ctx: PipelineContext,
    ): Promise<StepResult> {
        const timeoutMs = stage.timeoutMs ?? this.defaultTimeoutMs;
        const onError = stage.onError ?? this.defaultOnError;

        try {
            // Create abort controller for timeout
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

            // Run the step with timeout
            const resultPromise = stage.step({ ...ctx, signal: controller.signal });
            const result = await Promise.race([
                resultPromise,
                new Promise<StepResult>((_, reject) => {
                    controller.signal.addEventListener('abort', () => {
                        reject(new Error(`Stage '${stage.name}' timed out after ${timeoutMs}ms`));
                    });
                }),
            ]);

            clearTimeout(timeoutId);
            return result;
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            this.logger?.error('Middleware stage failed', {
                stage: stage.name,
                error: message,
            });

            if (onError === 'allow') {
                return { action: 'continue' };
            }

            return {
                action: 'deny',
                reason: `Middleware error: ${message}`,
                statusCode: 500,
            };
        }
    }

    /**
     * Sorts stages by order.
     */
    private sortStages(): void {
        const sortFn = (a: StageConfig, b: StageConfig) =>
            (a.order ?? 0) - (b.order ?? 0);
        this.preStages.sort(sortFn);
        this.postStages.sort(sortFn);
    }
}

// ============================================================================
// Factory Functions
// ============================================================================

/**
 * Creates a pipeline executor from configuration.
 */
export function createExecutor(
    stages: StageConfig[],
    options?: ExecutorOptions,
): PipelineExecutor {
    const executor = new PipelineExecutor(options);

    for (const stage of stages) {
        if (stage.type === 'pre') {
            executor.addPreStage(stage);
        } else {
            executor.addPostStage(stage);
        }
    }

    return executor;
}
