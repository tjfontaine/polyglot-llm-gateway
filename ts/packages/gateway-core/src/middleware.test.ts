import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
    PipelineExecutor,
    createExecutor,
    continueResult,
    modifyResult,
    denyResult,
    respondResult,
} from './middleware/index';
import type { PipelineContext, StageConfig, StepResult } from './middleware/types';
import type { CanonicalRequest, CanonicalResponse } from './domain/types';

describe('PipelineExecutor', () => {
    let executor: PipelineExecutor;
    let ctx: PipelineContext;

    beforeEach(() => {
        executor = new PipelineExecutor();
        ctx = {
            request: {
                tenantId: 'test-tenant',
                model: 'gpt-4',
                messages: [{ role: 'user', content: 'Hello' }],
                stream: false,
                sourceAPIType: 'openai',
            },
            tenantId: 'test-tenant',
            interactionId: 'int-123',
            metadata: new Map(),
        };
    });

    describe('runPre', () => {
        it('should run pre-stages and return continue result', async () => {
            const step = vi.fn().mockResolvedValue(continueResult());

            executor.addPreStage({
                name: 'test-stage',
                type: 'pre',
                step,
            });

            const result = await executor.runPre(ctx);

            expect(result.continue).toBe(true);
            expect(step).toHaveBeenCalledTimes(1);
        });

        it('should pass modified request through stages', async () => {
            const modifiedRequest = { ...ctx.request, model: 'gpt-4-turbo' };

            executor.addPreStage({
                name: 'modifier',
                type: 'pre',
                step: async () => modifyResult({ request: modifiedRequest as CanonicalRequest }),
            });

            executor.addPreStage({
                name: 'checker',
                type: 'pre',
                step: async (context) => {
                    expect(context.request.model).toBe('gpt-4-turbo');
                    return continueResult();
                },
            });

            const result = await executor.runPre(ctx);

            expect(result.continue).toBe(true);
            expect(result.request?.model).toBe('gpt-4-turbo');
        });

        it('should stop on deny result', async () => {
            const firstStep = vi.fn().mockResolvedValue(denyResult('Blocked', 403));
            const secondStep = vi.fn().mockResolvedValue(continueResult());

            executor.addPreStage({ name: 'blocker', type: 'pre', step: firstStep });
            executor.addPreStage({ name: 'never-runs', type: 'pre', step: secondStep });

            const result = await executor.runPre(ctx);

            expect(result.continue).toBe(false);
            expect(result.denyReason).toBe('Blocked');
            expect(result.denyStatusCode).toBe(403);
            expect(secondStep).not.toHaveBeenCalled();
        });

        it('should stop on respond result', async () => {
            const earlyResponse: CanonicalResponse = {
                id: 'resp-1',
                object: 'chat.completion',
                created: Date.now(),
                model: 'gpt-4',
                choices: [{ index: 0, message: { role: 'assistant', content: 'Cached' }, finishReason: 'stop' }],
                usage: { promptTokens: 0, completionTokens: 5, totalTokens: 5 },
                sourceAPIType: 'openai',
            };

            executor.addPreStage({
                name: 'cache-hit',
                type: 'pre',
                step: async () => respondResult(earlyResponse),
            });

            const result = await executor.runPre(ctx);

            expect(result.continue).toBe(false);
            expect(result.response).toBeDefined();
            expect(result.response?.choices[0]?.message.content).toBe('Cached');
        });
    });

    describe('runPost', () => {
        it('should run post-stages with response in context', async () => {
            const response: CanonicalResponse = {
                id: 'resp-1',
                object: 'chat.completion',
                created: Date.now(),
                model: 'gpt-4',
                choices: [{ index: 0, message: { role: 'assistant', content: 'Hello!' }, finishReason: 'stop' }],
                usage: { promptTokens: 5, completionTokens: 5, totalTokens: 10 },
                sourceAPIType: 'openai',
            };

            ctx.response = response;

            const step = vi.fn().mockResolvedValue(continueResult());
            executor.addPostStage({ name: 'logger', type: 'post', step });

            const result = await executor.runPost(ctx);

            expect(result.continue).toBe(true);
            expect(step).toHaveBeenCalledWith(expect.objectContaining({ response }));
        });
    });

    describe('stage ordering', () => {
        it('should execute stages in order', async () => {
            const order: string[] = [];

            executor.addPreStage({
                name: 'third',
                type: 'pre',
                order: 30,
                step: async () => { order.push('third'); return continueResult(); },
            });

            executor.addPreStage({
                name: 'first',
                type: 'pre',
                order: 10,
                step: async () => { order.push('first'); return continueResult(); },
            });

            executor.addPreStage({
                name: 'second',
                type: 'pre',
                order: 20,
                step: async () => { order.push('second'); return continueResult(); },
            });

            await executor.runPre(ctx);

            expect(order).toEqual(['first', 'second', 'third']);
        });
    });

    describe('error handling', () => {
        it('should deny on step error with deny mode (default)', async () => {
            executor.addPreStage({
                name: 'failing',
                type: 'pre',
                step: async () => { throw new Error('Step failed'); },
            });

            const result = await executor.runPre(ctx);

            expect(result.continue).toBe(false);
            expect(result.denyReason).toContain('Step failed');
        });

        it('should continue on step error with allow mode', async () => {
            executor.addPreStage({
                name: 'failing',
                type: 'pre',
                onError: 'allow',
                step: async () => { throw new Error('Step failed'); },
            });

            const result = await executor.runPre(ctx);

            expect(result.continue).toBe(true);
        });
    });
});

describe('createExecutor', () => {
    it('should create executor with stages', async () => {
        const stages: StageConfig[] = [
            { name: 'pre1', type: 'pre', step: async () => continueResult() },
            { name: 'post1', type: 'post', step: async () => continueResult() },
        ];

        const executor = createExecutor(stages);

        const ctx: PipelineContext = {
            request: {
                tenantId: 'test',
                model: 'gpt-4',
                messages: [],
                stream: false,
                sourceAPIType: 'openai',
            },
            tenantId: 'test',
            interactionId: 'int-1',
            metadata: new Map(),
        };

        const preResult = await executor.runPre(ctx);
        expect(preResult.continue).toBe(true);

        const postResult = await executor.runPost(ctx);
        expect(postResult.continue).toBe(true);
    });
});

describe('result helpers', () => {
    it('continueResult creates continue action', () => {
        const result = continueResult();
        expect(result.action).toBe('continue');
    });

    it('modifyResult creates modify action with request', () => {
        const request = { model: 'test' } as CanonicalRequest;
        const result = modifyResult({ request });
        expect(result.action).toBe('modify');
        if (result.action === 'modify') {
            expect(result.request).toBe(request);
        }
    });

    it('denyResult creates deny action', () => {
        const result = denyResult('Not allowed', 403);
        expect(result.action).toBe('deny');
        if (result.action === 'deny') {
            expect(result.reason).toBe('Not allowed');
            expect(result.statusCode).toBe(403);
        }
    });

    it('respondResult creates respond action', () => {
        const response = { id: 'test' } as CanonicalResponse;
        const result = respondResult(response);
        expect(result.action).toBe('respond');
        if (result.action === 'respond') {
            expect(result.response).toBe(response);
        }
    });
});
