/**
 * Built-in content filter step.
 *
 * @module middleware/steps/filter
 */

import type { CanonicalRequest, Message } from '../../domain/types.js';
import type { PipelineContext, StepResult, ContentFilterStepConfig } from '../types.js';
import { continueResult, denyResult, modifyResult } from '../types.js';

/**
 * Creates a content filter middleware step.
 */
export function createContentFilterStep(
    config: ContentFilterStepConfig,
): (ctx: PipelineContext) => Promise<StepResult> {
    const {
        blockPatterns = [],
        allowPatterns = [],
        blockAction = 'deny',
    } = config;

    // Compile regex patterns
    const blockRegexes = blockPatterns.map((p) => new RegExp(p, 'i'));
    const allowRegexes = allowPatterns.map((p) => new RegExp(p, 'i'));

    return async (ctx: PipelineContext): Promise<StepResult> => {
        const request = ctx.request;

        // Extract all text content from messages
        const texts = extractTexts(request);

        // Check each text
        for (const text of texts) {
            // Check allow patterns first (if any)
            if (allowRegexes.length > 0) {
                const allowed = allowRegexes.some((r) => r.test(text));
                if (!allowed) {
                    if (blockAction === 'deny') {
                        return denyResult('Content not in allow list', 400);
                    }
                    // Remove action - handled below
                }
            }

            // Check block patterns
            for (const regex of blockRegexes) {
                if (regex.test(text)) {
                    if (blockAction === 'deny') {
                        return denyResult('Content contains blocked pattern', 400);
                    }

                    // Remove action - filter the message
                    const filteredRequest = filterMessage(request, text, regex);
                    return modifyResult({ request: filteredRequest });
                }
            }
        }

        return continueResult();
    };
}

/**
 * Extracts all text content from a request.
 */
function extractTexts(request: CanonicalRequest): string[] {
    const texts: string[] = [];

    for (const message of request.messages) {
        if (message.content) {
            texts.push(message.content);
        }
    }

    if (request.systemPrompt) {
        texts.push(request.systemPrompt);
    }

    if (request.instructions) {
        texts.push(request.instructions);
    }

    return texts;
}

/**
 * Filters blocked content from a message.
 */
function filterMessage(
    request: CanonicalRequest,
    _text: string,
    regex: RegExp,
): CanonicalRequest {
    const messages = request.messages.map((msg): Message => {
        if (msg.content) {
            return {
                ...msg,
                content: msg.content.replace(regex, '[FILTERED]'),
            };
        }
        return msg;
    });

    return { ...request, messages };
}
