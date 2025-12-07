/**
 * Built-in log step.
 *
 * @module middleware/steps/log
 */

import type { PipelineContext, StepResult, LogStepConfig } from '../types.js';
import { continueResult } from '../types.js';

/**
 * Creates a log middleware step.
 */
export function createLogStep(
    config: LogStepConfig,
    log: (level: string, message: string, fields: Record<string, unknown>) => void,
): (ctx: PipelineContext) => Promise<StepResult> {
    const { level = 'info', message = 'Pipeline step', fields = [] } = config;

    return async (ctx: PipelineContext): Promise<StepResult> => {
        // Build log fields
        const logFields: Record<string, unknown> = {
            tenantId: ctx.tenantId,
            appName: ctx.appName,
            interactionId: ctx.interactionId,
            model: ctx.request.model,
            stream: ctx.request.stream,
            messageCount: ctx.request.messages.length,
        };

        // Add configured fields
        for (const field of fields) {
            if (field.startsWith('request.')) {
                const path = field.slice(8);
                logFields[field] = getNestedValue(ctx.request, path);
            } else if (field.startsWith('response.') && ctx.response) {
                const path = field.slice(9);
                logFields[field] = getNestedValue(ctx.response, path);
            } else if (field.startsWith('meta.')) {
                const key = field.slice(5);
                logFields[field] = ctx.metadata.get(key);
            }
        }

        // Log
        log(level, message, logFields);

        return continueResult();
    };
}

/**
 * Gets a nested value from an object.
 */
function getNestedValue(obj: unknown, path: string): unknown {
    const parts = path.split('.');
    let current = obj;

    for (const part of parts) {
        if (current === null || current === undefined) return undefined;
        if (typeof current !== 'object') return undefined;
        current = (current as Record<string, unknown>)[part];
    }

    return current;
}
