/**
 * File-based configuration provider for Node.js.
 *
 * Reads configuration from a YAML or JSON file and supports hot-reload.
 *
 * @module config/file
 */

import { readFileSync, watchFile, unwatchFile, existsSync } from 'node:fs';
import { parse as parseYaml } from 'yaml';
import type {
    ConfigProvider,
    WatchableConfigProvider,
    GatewayConfig,
    ConfigChangeCallback,
} from '@polyglot-llm-gateway/gateway-core';

/**
 * Options for the file config provider.
 */
export interface FileConfigProviderOptions {
    /** Path to the config file. */
    path: string;

    /** Poll interval for file changes (ms). Default: 1000. */
    pollInterval?: number;

    /** Environment variables with values to substitute. */
    env?: Record<string, string | undefined>;
}

/**
 * Configuration provider that reads from a YAML or JSON file.
 * Supports hot-reload via file watching.
 */
export class FileConfigProvider implements WatchableConfigProvider {
    private readonly path: string;
    private readonly pollInterval: number;
    private readonly env: Record<string, string | undefined>;

    constructor(options: FileConfigProviderOptions) {
        this.path = options.path;
        this.pollInterval = options.pollInterval ?? 1000;
        this.env = options.env ?? process.env;
    }

    /**
     * Loads the configuration from the file.
     */
    async load(): Promise<GatewayConfig> {
        if (!existsSync(this.path)) {
            throw new Error(`Config file not found: ${this.path}`);
        }

        const content = readFileSync(this.path, 'utf-8');
        const expanded = this.expandEnvVars(content);

        let config: GatewayConfig;

        if (this.path.endsWith('.json')) {
            config = JSON.parse(expanded) as GatewayConfig;
        } else {
            // Assume YAML
            config = parseYaml(expanded) as GatewayConfig;
        }

        // Normalize the config (cast through unknown to handle the raw parsed data)
        return this.normalizeConfig(config as unknown as Record<string, unknown>);
    }

    /**
     * Watches for configuration file changes.
     */
    async watch(onChange: ConfigChangeCallback, signal?: AbortSignal): Promise<void> {
        return new Promise((resolve) => {
            let lastMtime = 0;

            const checkForChanges = async () => {
                try {
                    const newConfig = await this.load();
                    onChange(newConfig);
                } catch (error) {
                    console.error('Failed to reload config:', error);
                }
            };

            // Watch for file changes
            watchFile(this.path, { interval: this.pollInterval }, async (curr, prev) => {
                if (curr.mtimeMs !== lastMtime) {
                    lastMtime = curr.mtimeMs;
                    await checkForChanges();
                }
            });

            // Handle abort signal
            if (signal) {
                signal.addEventListener('abort', () => {
                    unwatchFile(this.path);
                    resolve();
                });
            }
        });
    }

    /**
     * Expands environment variable references like ${VAR_NAME}.
     */
    private expandEnvVars(content: string): string {
        return content.replace(/\$\{([^}]+)\}/g, (_, varName) => {
            const value = this.env[varName];
            if (value === undefined) {
                console.warn(`Environment variable ${varName} not set`);
                return '';
            }
            return value;
        });
    }

    /**
     * Normalizes the config to ensure required fields.
     */
    private normalizeConfig(raw: Record<string, unknown>): GatewayConfig {
        const config: GatewayConfig = {
            apps: [],
            providers: [],
        };

        // Server config
        if (raw.server) {
            config.server = raw.server as GatewayConfig['server'];
        }

        // Storage config
        if (raw.storage) {
            config.storage = raw.storage as GatewayConfig['storage'];
        }

        // Providers (with snake_case to camelCase conversion)
        if (Array.isArray(raw.providers)) {
            config.providers = raw.providers.map((p: Record<string, unknown>) => ({
                name: p.name as string,
                type: p.type as string,
                apiKey: (p.api_key ?? p.apiKey) as string,
                baseUrl: (p.base_url ?? p.baseUrl) as string | undefined,
                supportsResponses: (p.supports_responses ?? p.supportsResponses) as boolean | undefined,
                enablePassthrough: (p.enable_passthrough ?? p.enablePassthrough) as boolean | undefined,
                useResponsesApi: (p.use_responses_api ?? p.useResponsesApi) as boolean | undefined,
                responsesThreadKeyPath: (p.responses_thread_key_path ?? p.responsesThreadKeyPath) as string | undefined,
                responsesThreadPersistence: (p.responses_thread_persistence ?? p.responsesThreadPersistence) as boolean | undefined,
            }));
        }

        // Apps (from apps or frontdoors)
        if (Array.isArray(raw.apps)) {
            config.apps = raw.apps.map((a: Record<string, unknown>) => ({
                name: a.name as string,
                frontdoor: a.frontdoor as string,
                path: a.path as string,
                provider: a.provider as string | undefined,
                defaultModel: (a.default_model ?? a.defaultModel) as string | undefined,
                enableResponses: (a.enable_responses ?? a.enableResponses) as boolean | undefined,
                forceStore: (a.force_store ?? a.forceStore) as boolean | undefined,
            }));
        } else if (Array.isArray(raw.frontdoors)) {
            // Legacy frontdoors format
            config.apps = raw.frontdoors.map((fd: Record<string, unknown>) => ({
                name: fd.type as string,
                frontdoor: fd.type as string,
                path: fd.path as string,
                provider: fd.provider as string | undefined,
                defaultModel: (fd.default_model ?? fd.defaultModel) as string | undefined,
                enableResponses: (fd.enable_responses ?? fd.enableResponses) as boolean | undefined,
            }));
        }

        // Routing
        if (raw.routing) {
            const routing = raw.routing as Record<string, unknown>;
            config.routing = {
                defaultProvider: (routing.default_provider ?? routing.defaultProvider) as string | undefined,
                rules: Array.isArray(routing.rules)
                    ? routing.rules.map((r: Record<string, unknown>) => ({
                        modelPrefix: (r.model_prefix ?? r.modelPrefix) as string | undefined,
                        modelExact: (r.model_exact ?? r.modelExact) as string | undefined,
                        provider: r.provider as string,
                    }))
                    : [],
            };
        }

        // Tenants
        if (Array.isArray(raw.tenants)) {
            config.tenants = raw.tenants.map((t: Record<string, unknown>) => ({
                id: t.id as string,
                name: t.name as string,
                apiKeys: Array.isArray(t.api_keys)
                    ? t.api_keys.map((k: Record<string, unknown>) => ({
                        keyHash: (k.key_hash ?? k.keyHash) as string,
                        description: k.description as string | undefined,
                    }))
                    : undefined,
            }));
        }

        return config;
    }
}
