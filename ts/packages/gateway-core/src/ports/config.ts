/**
 * Configuration provider port.
 *
 * @module ports/config
 */

import type { ShadowConfig } from '../domain/shadow.js';

// ============================================================================
// Configuration Types
// ============================================================================

/**
 * Gateway configuration.
 */
export interface GatewayConfig {
    /** Server configuration. */
    server?: ServerConfig | undefined;

    /** Storage configuration. */
    storage?: StorageConfig | undefined;

    /** Tenant configurations. */
    tenants?: TenantConfig[] | undefined;

    /** App configurations. */
    apps: AppConfig[];

    /** Provider configurations. */
    providers: ProviderConfig[];

    /** Global routing configuration. */
    routing?: RoutingConfig | undefined;
}

/** Server configuration. */
export interface ServerConfig {
    /** Port to listen on. */
    port?: number | undefined;
}

/** Storage configuration. */
export interface StorageConfig {
    /** Storage type. */
    type: 'sqlite' | 'postgres' | 'mysql' | 'memory' | 'd1' | 'none';

    /** SQLite configuration. */
    sqlite?: {
        path: string;
    } | undefined;

    /** Generic database configuration. */
    database?: {
        driver: string;
        dsn: string;
    } | undefined;
}

/** Tenant configuration. */
export interface TenantConfig {
    /** Tenant ID. */
    id: string;

    /** Tenant name. */
    name: string;

    /** API keys for this tenant. */
    apiKeys?: APIKeyConfig[] | undefined;

    /** Tenant-specific providers. */
    providers?: ProviderConfig[] | undefined;

    /** Tenant-specific routing. */
    routing?: RoutingConfig | undefined;
}

/** API key configuration. */
export interface APIKeyConfig {
    /** SHA-256 hash of the API key. */
    keyHash: string;

    /** Description of the key. */
    description?: string | undefined;
}

/** App configuration. */
export interface AppConfig {
    /** App name. */
    name: string;

    /** Frontdoor type. */
    frontdoor: string;

    /** Base path for this app. */
    path: string;

    /** Force specific provider. */
    provider?: string | undefined;

    /** Default model. */
    defaultModel?: string | undefined;

    /** Model routing configuration. */
    modelRouting?: ModelRoutingConfig | undefined;

    /** Models to expose. */
    models?: ModelListItem[] | undefined;

    /** Enable Responses API. */
    enableResponses?: boolean | undefined;

    /** Force recording even when client sends store:false. */
    forceStore?: boolean | undefined;

    /** Shadow mode configuration. */
    shadow?: ShadowConfig | undefined;

    /** Pipeline configuration. */
    pipeline?: PipelineConfig | undefined;
}

/** Pipeline configuration. */
export interface PipelineConfig {
    /** Pipeline stages. */
    stages: PipelineStageConfig[];
}

/** Pipeline stage configuration. */
export interface PipelineStageConfig {
    /** Stage name. */
    name: string;

    /** Stage type. */
    type: 'pre' | 'post';

    /** Webhook URL. */
    url: string;

    /** Timeout duration. */
    timeout?: string | undefined;

    /** Error handling mode. */
    onError?: 'allow' | 'deny' | undefined;

    /** Retry count. */
    retries?: number | undefined;

    /** Suppress response if denied. */
    squelch?: boolean | undefined;

    /** Extra headers. */
    headers?: Record<string, string> | undefined;

    /** Execution order. */
    order?: number | undefined;
}

/** Provider configuration. */
export interface ProviderConfig {
    /** Provider name. */
    name: string;

    /** Provider type. */
    type: string;

    /** API key. */
    apiKey: string;

    /** Custom base URL. */
    baseUrl?: string | undefined;

    /** Supports native Responses API. */
    supportsResponses?: boolean | undefined;

    /** Enable pass-through mode. */
    enablePassthrough?: boolean | undefined;

    /** Use Responses API instead of Chat Completions. */
    useResponsesApi?: boolean | undefined;

    /** JSON path to derive thread key. */
    responsesThreadKeyPath?: string | undefined;

    /** Persist thread state. */
    responsesThreadPersistence?: boolean | undefined;
}

/** Routing configuration. */
export interface RoutingConfig {
    /** Routing rules. */
    rules?: RoutingRule[] | undefined;

    /** Default provider. */
    defaultProvider?: string | undefined;
}

/** Routing rule. */
export interface RoutingRule {
    /** Match model by prefix. */
    modelPrefix?: string | undefined;

    /** Match model exactly. */
    modelExact?: string | undefined;

    /** Target provider. */
    provider: string;
}

/** Model routing configuration. */
export interface ModelRoutingConfig {
    /** Prefix to provider mapping. */
    prefixProviders?: Record<string, string> | undefined;

    /** Rewrite rules. */
    rewrites?: ModelRewriteRule[] | undefined;

    /** Fallback rule. */
    fallback?: ModelRewriteRule | undefined;
}

/** Model rewrite rule. */
export interface ModelRewriteRule {
    /** Match model exactly. */
    modelExact?: string | undefined;

    /** Match model by prefix. */
    modelPrefix?: string | undefined;

    /** Target provider. */
    provider?: string | undefined;

    /** Target model. */
    model?: string | undefined;

    /** Rewrite response model to original. */
    rewriteResponseModel?: boolean | undefined;
}

/** Model list item. */
export interface ModelListItem {
    /** Model ID. */
    id: string;

    /** Object type. */
    object?: string | undefined;

    /** Owner. */
    ownedBy?: string | undefined;

    /** Creation timestamp. */
    created?: number | undefined;
}

// ============================================================================
// ConfigProvider Interface
// ============================================================================

/**
 * Callback for config changes.
 */
export type ConfigChangeCallback = (config: GatewayConfig) => void;

/**
 * Provides configuration to the gateway.
 * Implementations: KV-based (CF), file-based (Node), etc.
 */
export interface ConfigProvider {
    /**
     * Loads the gateway configuration.
     */
    load(): Promise<GatewayConfig>;
}

/**
 * A ConfigProvider that supports watching for changes.
 * Extend your ConfigProvider with this for hot reload support.
 */
export interface WatchableConfigProvider extends ConfigProvider {
    /**
     * Watches for configuration changes.
     * Calls the callback when configuration changes.
     * 
     * @param onChange - Callback invoked when config changes
     * @param signal - AbortSignal to stop watching
     * @returns Promise that resolves when watching stops
     */
    watch(onChange: ConfigChangeCallback, signal?: AbortSignal): Promise<void>;
}

/**
 * Type guard to check if a ConfigProvider is watchable.
 */
export function isWatchableConfigProvider(
    provider: ConfigProvider,
): provider is WatchableConfigProvider {
    return 'watch' in provider && typeof provider.watch === 'function';
}
