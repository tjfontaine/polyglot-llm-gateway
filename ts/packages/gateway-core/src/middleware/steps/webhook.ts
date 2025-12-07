/**
 * Built-in webhook step.
 *
 * @module middleware/steps/webhook
 */

import type { PipelineContext, StepResult, WebhookStepConfig } from '../types.js';
import { continueResult, denyResult, modifyResult } from '../types.js';

/**
 * Webhook response format.
 */
interface WebhookResponse {
    /** Action to take. */
    action?: 'allow' | 'deny' | 'modify' | undefined;

    /** Deny reason. */
    reason?: string | undefined;

    /** Modified request fields. */
    request?: Record<string, unknown> | undefined;

    /** Modified response fields. */
    response?: Record<string, unknown> | undefined;
}

/**
 * Creates a webhook middleware step.
 */
export function createWebhookStep(config: WebhookStepConfig): (ctx: PipelineContext) => Promise<StepResult> {
    const { url, method = 'POST', headers = {}, timeoutMs = 5000, retries = 0 } = config;

    return async (ctx: PipelineContext): Promise<StepResult> => {
        // Build payload
        const payload = {
            request: ctx.request,
            response: ctx.response,
            tenantId: ctx.tenantId,
            appName: ctx.appName,
            interactionId: ctx.interactionId,
        };

        // Retry loop
        let lastError: Error | undefined;
        for (let attempt = 0; attempt <= retries; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

                const response = await fetch(url, {
                    method,
                    headers: {
                        'Content-Type': 'application/json',
                        ...headers,
                    },
                    body: JSON.stringify(payload),
                    signal: controller.signal,
                });

                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`Webhook returned ${response.status}`);
                }

                const result = await response.json() as WebhookResponse;

                // Process result
                switch (result.action) {
                    case 'deny':
                        return denyResult(result.reason ?? 'Denied by webhook', 403);

                    case 'modify':
                        // Merge modifications into context
                        const modifiedRequest = result.request
                            ? { ...ctx.request, ...result.request }
                            : undefined;
                        const modifiedResponse = result.response && ctx.response
                            ? { ...ctx.response, ...result.response }
                            : undefined;

                        if (modifiedRequest || modifiedResponse) {
                            return modifyResult({
                                request: modifiedRequest as any,
                                response: modifiedResponse as any,
                            });
                        }
                        return continueResult();

                    case 'allow':
                    default:
                        return continueResult();
                }
            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));

                // Don't retry on abort
                if (ctx.signal?.aborted) {
                    break;
                }

                // Wait before retry
                if (attempt < retries) {
                    await new Promise((resolve) => setTimeout(resolve, 100 * (attempt + 1)));
                }
            }
        }

        throw lastError ?? new Error('Webhook failed');
    };
}
