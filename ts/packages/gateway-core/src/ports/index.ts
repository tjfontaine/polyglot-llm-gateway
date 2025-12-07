/**
 * Port interfaces module exports.
 *
 * @module ports
 */

// Config
export type {
    ConfigProvider,
    WatchableConfigProvider,
    ConfigChangeCallback,
    GatewayConfig,
    ServerConfig,
    StorageConfig,
    TenantConfig,
    APIKeyConfig,
    AppConfig,
    PipelineConfig,
    PipelineStageConfig,
    ProviderConfig,
    RoutingConfig,
    RoutingRule,
    ModelRoutingConfig,
    ModelRewriteRule,
    ModelListItem,
} from './config.js';
export { isWatchableConfigProvider } from './config.js';

// Auth
export type { AuthProvider, AuthContext, Tenant } from './auth.js';
export { extractBearerToken, hashAPIKey } from './auth.js';

// Storage
export type {
    StorageProvider,
    ConversationStore,
    ResponseStore,
    InteractionStore,
    ShadowStore,
    ThreadStateStore,
    Conversation,
    StoredMessage,
    ResponseRecord,
    InteractionSummary,
    InteractionType,
    ListOptions,
    InteractionListOptions,
    DivergenceListOptions,
} from './storage.js';

// Events
export type { EventPublisher } from './events.js';
export { NullEventPublisher } from './events.js';

// Provider
export type {
    Provider,
    ProviderFactory,
    ProviderFactoryConfig,
    ProviderRegistry,
} from './provider.js';
export { createProviderRegistry } from './provider.js';
