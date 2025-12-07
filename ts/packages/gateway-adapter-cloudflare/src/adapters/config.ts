/**
 * KV-based configuration provider for Cloudflare Workers.
 *
 * @module adapters/config
 */

import type { ConfigProvider, GatewayConfig } from '@polyglot-llm-gateway/gateway-core';
import { KV_KEYS } from '../bindings.js';

// ============================================================================
// KV Config Provider
// ============================================================================

/**
 * Configuration provider backed by Cloudflare KV.
 */
export class KVConfigProvider implements ConfigProvider {
    constructor(private readonly kv: KVNamespace) { }

    async load(): Promise<GatewayConfig> {
        // Try to load full config as single JSON
        const configJson = await this.kv.get(KV_KEYS.CONFIG, 'json');
        if (configJson) {
            return configJson as GatewayConfig;
        }

        // Otherwise, load individual pieces
        const [providers, apps, routing] = await Promise.all([
            this.kv.get(KV_KEYS.PROVIDERS, 'json'),
            this.kv.get(KV_KEYS.APPS, 'json'),
            this.kv.get(KV_KEYS.ROUTING, 'json'),
        ]);

        return {
            providers: (providers as GatewayConfig['providers']) ?? [],
            apps: (apps as GatewayConfig['apps']) ?? [],
            routing: routing as GatewayConfig['routing'],
        };
    }
}

// ============================================================================
// Static Config Provider
// ============================================================================

/**
 * Configuration provider with static config.
 * Useful for development or when config is embedded in code.
 */
export class StaticConfigProvider implements ConfigProvider {
    constructor(private readonly config: GatewayConfig) { }

    async load(): Promise<GatewayConfig> {
        return this.config;
    }
}

// ============================================================================
// Environment Config Provider
// ============================================================================

/**
 * Configuration provider that reads from environment variables.
 */
export class EnvConfigProvider implements ConfigProvider {
    constructor(private readonly env: Record<string, string | undefined>) { }

    async load(): Promise<GatewayConfig> {
        const configJson = this.env['GATEWAY_CONFIG'];
        if (configJson) {
            return JSON.parse(configJson) as GatewayConfig;
        }

        // Build minimal config from individual env vars
        const providers: GatewayConfig['providers'] = [];

        // OpenAI
        if (this.env['OPENAI_API_KEY']) {
            providers.push({
                name: 'openai',
                type: 'openai',
                apiKey: this.env['OPENAI_API_KEY'],
                baseUrl: this.env['OPENAI_BASE_URL'],
            });
        }

        // Anthropic
        if (this.env['ANTHROPIC_API_KEY']) {
            providers.push({
                name: 'anthropic',
                type: 'anthropic',
                apiKey: this.env['ANTHROPIC_API_KEY'],
                baseUrl: this.env['ANTHROPIC_BASE_URL'],
            });
        }

        return {
            providers,
            apps: [
                {
                    name: 'default',
                    frontdoor: 'openai',
                    path: '/v1',
                },
            ],
            routing: {
                defaultProvider: 'openai',
            },
        };
    }
}
