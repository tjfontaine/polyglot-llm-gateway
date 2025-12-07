/**
 * Built-in transform step.
 *
 * @module middleware/steps/transform
 */

import type { PipelineContext, StepResult, TransformStepConfig } from '../types.js';
import { continueResult, modifyResult } from '../types.js';

/**
 * Creates a transform middleware step.
 */
export function createTransformStep(
    config: TransformStepConfig,
): (ctx: PipelineContext) => Promise<StepResult> {
    const { path, value, delete: shouldDelete } = config;

    return async (ctx: PipelineContext): Promise<StepResult> => {
        // Work on a copy of the request
        const request = { ...ctx.request };

        if (shouldDelete) {
            // Delete the field
            deleteField(request, path);
        } else if (value !== undefined) {
            // Set the field
            const resolvedValue = resolveTemplate(value, ctx);
            setField(request, path, resolvedValue);
        }

        return modifyResult({ request: request as any });
    };
}

/**
 * Gets a nested field value.
 */
function getField(obj: Record<string, unknown>, path: string): unknown {
    const parts = path.split('.');
    let current: unknown = obj;

    for (const part of parts) {
        if (current === null || current === undefined) return undefined;
        if (typeof current !== 'object') return undefined;
        current = (current as Record<string, unknown>)[part];
    }

    return current;
}

/**
 * Sets a nested field value.
 */
function setField(obj: Record<string, unknown>, path: string, value: unknown): void {
    const parts = path.split('.');
    let current: Record<string, unknown> = obj;

    for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i];
        if (!part) continue;

        if (current[part] === undefined || current[part] === null) {
            current[part] = {};
        }
        current = current[part] as Record<string, unknown>;
    }

    const lastPart = parts[parts.length - 1];
    if (lastPart) {
        current[lastPart] = value;
    }
}

/**
 * Deletes a nested field.
 */
function deleteField(obj: Record<string, unknown>, path: string): void {
    const parts = path.split('.');
    let current: Record<string, unknown> = obj;

    for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i];
        if (!part) continue;

        if (current[part] === undefined || current[part] === null) {
            return; // Path doesn't exist
        }
        current = current[part] as Record<string, unknown>;
    }

    const lastPart = parts[parts.length - 1];
    if (lastPart) {
        delete current[lastPart];
    }
}

/**
 * Resolves template variables in a string.
 * Supports {{field}} syntax where field is a path into the context.
 */
function resolveTemplate(template: string, ctx: PipelineContext): string {
    return template.replace(/\{\{(.+?)\}\}/g, (_, path) => {
        // Check context metadata first
        if (path.startsWith('meta.')) {
            const metaKey = path.slice(5);
            const value = ctx.metadata.get(metaKey);
            return value !== undefined ? String(value) : '';
        }

        // Check request fields
        if (path.startsWith('request.')) {
            const reqPath = path.slice(8);
            const value = getField(ctx.request as any, reqPath);
            return value !== undefined ? String(value) : '';
        }

        // Check response fields
        if (path.startsWith('response.') && ctx.response) {
            const respPath = path.slice(9);
            const value = getField(ctx.response as any, respPath);
            return value !== undefined ? String(value) : '';
        }

        // Check top-level context
        if (path === 'tenantId') return ctx.tenantId;
        if (path === 'appName') return ctx.appName ?? '';
        if (path === 'interactionId') return ctx.interactionId;

        return '';
    });
}
