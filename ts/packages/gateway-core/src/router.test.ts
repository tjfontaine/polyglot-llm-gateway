import { describe, it, expect } from 'vitest';
import { Router } from './router';
import type { AppConfig } from './ports/config';
import type { Frontdoor, FrontdoorContext, FrontdoorResponse } from './frontdoors/types';

// Mock frontdoor for testing
class MockFrontdoor implements Frontdoor {
    readonly name: string;
    constructor(name: string) {
        this.name = name;
    }
    matches(path: string): boolean {
        if (this.name === 'openai') {
            return path.includes('/chat/completions') || path.includes('/models');
        }
        return path.includes('/messages');
    }
    async handle(_ctx: FrontdoorContext): Promise<FrontdoorResponse> {
        return { response: new Response('ok') };
    }
}

describe('Router', () => {
    describe('matchApp', () => {
        it('should match app by path prefix', () => {
            const router = new Router();

            const app: AppConfig = {
                name: 'test-app',
                frontdoor: 'openai',
                path: '/v1',
            };
            router.addApp(app);

            const matched = router.matchApp('/v1/chat/completions');
            expect(matched?.name).toBe('test-app');
        });

        it('should match longest path prefix', () => {
            const router = new Router();

            router.addApp({ name: 'short', frontdoor: 'openai', path: '/api' });
            router.addApp({ name: 'long', frontdoor: 'openai', path: '/api/v1' });

            expect(router.matchApp('/api/v1/chat')?.name).toBe('long');
            expect(router.matchApp('/api/v2/chat')?.name).toBe('short');
        });

        it('should return undefined for unmatched paths', () => {
            const router = new Router();
            router.addApp({ name: 'test', frontdoor: 'openai', path: '/v1' });

            expect(router.matchApp('/unknown/path')).toBeUndefined();
        });
    });

    describe('getFrontdoor', () => {
        it('should get frontdoor by name from app config', () => {
            const router = new Router();
            const frontdoor = new MockFrontdoor('openai');
            router.addFrontdoor(frontdoor);

            const app: AppConfig = { name: 'test', frontdoor: 'openai', path: '/v1' };

            const matched = router.getFrontdoor(app);
            expect(matched?.name).toBe('openai');
        });
    });

    describe('matchFrontdoor', () => {
        it('should match frontdoor by path', () => {
            const router = new Router();
            router.addFrontdoor(new MockFrontdoor('openai'));
            router.addFrontdoor(new MockFrontdoor('anthropic'));

            expect(router.matchFrontdoor('/v1/chat/completions')?.name).toBe('openai');
            expect(router.matchFrontdoor('/v1/messages')?.name).toBe('anthropic');
        });
    });

    describe('selectProvider', () => {
        it('should use app forced provider', () => {
            const router = new Router();
            const app: AppConfig = {
                name: 'test',
                frontdoor: 'openai',
                path: '/v1',
                provider: 'custom-openai',
            };

            const selection = router.selectProvider('gpt-4', app);
            expect(selection.providerName).toBe('custom-openai');
        });

        it('should match model routing prefixes', () => {
            const router = new Router();
            const app: AppConfig = {
                name: 'test',
                frontdoor: 'openai',
                path: '/v1',
                modelRouting: {
                    prefixProviders: {
                        'claude-': 'anthropic',
                        'gpt-': 'openai',
                    },
                },
            };

            expect(router.selectProvider('claude-3-sonnet', app).providerName).toBe('anthropic');
            expect(router.selectProvider('gpt-4', app).providerName).toBe('openai');
        });

        it('should use default provider as fallback', () => {
            const router = new Router({ defaultRouting: { defaultProvider: 'fallback' } });

            const selection = router.selectProvider('unknown-model');
            expect(selection.providerName).toBe('fallback');
        });

        it('should match global routing rules', () => {
            const router = new Router({
                defaultRouting: {
                    rules: [
                        { modelPrefix: 'claude-', provider: 'anthropic' },
                        { modelExact: 'custom-model', provider: 'custom' },
                    ],
                    defaultProvider: 'openai',
                },
            });

            expect(router.selectProvider('claude-3-opus').providerName).toBe('anthropic');
            expect(router.selectProvider('custom-model').providerName).toBe('custom');
            expect(router.selectProvider('gpt-4').providerName).toBe('openai');
        });

        it('should apply model rewrites', () => {
            const router = new Router();
            const app: AppConfig = {
                name: 'test',
                frontdoor: 'openai',
                path: '/v1',
                modelRouting: {
                    rewrites: [
                        {
                            modelExact: 'fast',
                            provider: 'openai',
                            model: 'gpt-4o-mini',
                            rewriteResponseModel: true,
                        },
                    ],
                },
            };

            const selection = router.selectProvider('fast', app);
            expect(selection.providerName).toBe('openai');
            expect(selection.model).toBe('gpt-4o-mini');
            expect(selection.rewriteResponseModel).toBe(true);
        });
    });
});
