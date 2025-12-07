/**
 * Cloudflare Workers gateway entrypoint.
 */

import { Gateway } from '@polyglot-llm-gateway/gateway-core';
import {
    type Env,
    KVConfigProvider,
    KVAuthProvider,
    D1StorageProvider,
    QueueEventPublisher,
    NullEventPublisher,
    StaticConfigProvider,
    createDevAuthProvider,
} from '@polyglot-llm-gateway/gateway-adapter-cloudflare';

// Global gateway instance (reused across requests)
let gateway: Gateway | null = null;

export default {
    async fetch(
        request: Request,
        env: Env,
        ctx: ExecutionContext,
    ): Promise<Response> {
        // Create or reuse gateway
        if (!gateway) {
            const isDev = env.ENVIRONMENT === 'development';

            gateway = new Gateway({
                config: isDev
                    ? new StaticConfigProvider({
                        providers: [
                            {
                                name: 'openai',
                                type: 'openai',
                                apiKey: env.OPENAI_API_KEY ?? '',
                            },
                            {
                                name: 'anthropic',
                                type: 'anthropic',
                                apiKey: env.ANTHROPIC_API_KEY ?? '',
                            },
                        ],
                        apps: [
                            {
                                name: 'default',
                                frontdoor: 'openai',
                                path: '/v1',
                            },
                            {
                                name: 'anthropic',
                                frontdoor: 'anthropic',
                                path: '/anthropic',
                            },
                        ],
                        routing: {
                            defaultProvider: 'openai',
                        },
                    })
                    : new KVConfigProvider(env.CONFIG_KV),
                auth: isDev
                    ? createDevAuthProvider('dev-key', 'dev-tenant')
                    : new KVAuthProvider(env.AUTH_KV),
                storage: env.DB ? new D1StorageProvider(env.DB) : undefined,
                events: env.USAGE_QUEUE
                    ? new QueueEventPublisher(env.USAGE_QUEUE, ctx)
                    : new NullEventPublisher(),
            });
        }

        return gateway.fetch(request);
    },
};

// Extend Env for development
declare module '@polyglot-llm-gateway/gateway-adapter-cloudflare' {
    interface Env {
        OPENAI_API_KEY?: string;
        ANTHROPIC_API_KEY?: string;
    }
}
