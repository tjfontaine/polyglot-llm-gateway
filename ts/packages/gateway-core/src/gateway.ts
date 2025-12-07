/**
 * Main Gateway class - the core entry point for the LLM gateway.
 *
 * @module gateway
 */

import type {
    ConfigProvider,
    GatewayConfig,
    AppConfig,
    ProviderConfig,
} from './ports/config.js';
import { isWatchableConfigProvider } from './ports/config.js';
import type { AuthProvider, AuthContext } from './ports/auth.js';
import { extractBearerToken } from './ports/auth.js';
import type { StorageProvider } from './ports/storage.js';
import type { EventPublisher } from './ports/events.js';
import type { Provider, ProviderRegistry } from './ports/provider.js';
import { createProviderRegistry } from './ports/provider.js';
import type { Frontdoor, FrontdoorRegistry, FrontdoorContext } from './frontdoors/types.js';
import { createFrontdoorRegistry, openAIFrontdoor, anthropicFrontdoor } from './frontdoors/index.js';
import { createOpenAIProvider } from './providers/openai.js';
import { createAnthropicProvider } from './providers/anthropic.js';
import { Router, stripAppPrefix } from './router.js';
import { APIError, errAuthentication, errNotFound, errServer, toOpenAIError } from './domain/errors.js';
import type { Logger } from './utils/logging.js';
import { ConsoleLogger, requestLogger } from './utils/logging.js';
import { randomUUID } from './utils/crypto.js';

// ============================================================================
// Gateway Options
// ============================================================================

/**
 * Options for creating a Gateway.
 */
export interface GatewayOptions {
    /** Configuration provider. */
    config: ConfigProvider;

    /** Authentication provider. */
    auth: AuthProvider;

    /** Storage provider. */
    storage?: StorageProvider | undefined;

    /** Event publisher. */
    events?: EventPublisher | undefined;

    /** Logger. */
    logger?: Logger | undefined;

    /** Custom provider registry. */
    providerRegistry?: ProviderRegistry | undefined;

    /** Custom frontdoor registry. */
    frontdoorRegistry?: FrontdoorRegistry | undefined;

    /** Additional frontdoors to register. */
    frontdoors?: Frontdoor[] | undefined;
}

// ============================================================================
// Gateway
// ============================================================================

/**
 * The main Gateway class.
 * Handles HTTP requests and routes them to the appropriate frontdoor and provider.
 */
export class Gateway {
    private readonly configProvider: ConfigProvider;
    private readonly authProvider: AuthProvider;
    private readonly storageProvider: StorageProvider | undefined;
    private readonly eventPublisher: EventPublisher | undefined;
    private readonly logger: Logger;
    private readonly providerRegistry: ProviderRegistry;
    private readonly frontdoorRegistry: FrontdoorRegistry;

    // Initialized on first request or reload
    private config: GatewayConfig | undefined;
    private router: Router | undefined;
    private providers: Map<string, Provider> = new Map();

    // Hot reload state
    private watchAbortController: AbortController | undefined;
    private isWatching = false;

    constructor(options: GatewayOptions) {
        this.configProvider = options.config;
        this.authProvider = options.auth;
        this.storageProvider = options.storage;
        this.eventPublisher = options.events;
        this.logger = options.logger ?? new ConsoleLogger();

        // Setup provider registry
        this.providerRegistry = options.providerRegistry ?? createProviderRegistry();
        this.providerRegistry.register('openai', createOpenAIProvider);
        this.providerRegistry.register('anthropic', createAnthropicProvider);

        // Setup frontdoor registry
        this.frontdoorRegistry = options.frontdoorRegistry ?? createFrontdoorRegistry();
        this.frontdoorRegistry.register(openAIFrontdoor);
        this.frontdoorRegistry.register(anthropicFrontdoor);

        // Register additional frontdoors
        if (options.frontdoors) {
            for (const frontdoor of options.frontdoors) {
                this.frontdoorRegistry.register(frontdoor);
            }
        }
    }

    /**
     * Loads or reloads the gateway configuration.
     */
    async reload(): Promise<void> {
        this.config = await this.configProvider.load();
        this.router = new Router({
            defaultRouting: this.config.routing,
        });

        // Register apps
        for (const app of this.config.apps) {
            this.router.addApp(app);
        }

        // Register frontdoors
        for (const name of this.frontdoorRegistry.list()) {
            const frontdoor = this.frontdoorRegistry.get(name);
            if (frontdoor) {
                this.router.addFrontdoor(frontdoor);
            }
        }

        // Create providers
        this.providers.clear();
        for (const providerConfig of this.config.providers) {
            try {
                const provider = this.createProvider(providerConfig);
                this.providers.set(providerConfig.name, provider);
            } catch (error) {
                this.logger.error('Failed to create provider', {
                    provider: providerConfig.name,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        this.logger.info('Gateway configuration loaded', {
            apps: this.config.apps.length,
            providers: this.providers.size,
        });
    }

    /**
     * Starts watching for configuration changes.
     * Only works if the ConfigProvider implements WatchableConfigProvider.
     * Call stopWatching() to stop the watcher.
     */
    async startWatching(): Promise<void> {
        if (this.isWatching) {
            this.logger.warn('Config watching already active');
            return;
        }

        if (!isWatchableConfigProvider(this.configProvider)) {
            this.logger.info('ConfigProvider does not support watching');
            return;
        }

        this.isWatching = true;
        this.watchAbortController = new AbortController();

        this.logger.info('Starting config watcher');

        // Handle config changes
        const onChange = async (newConfig: GatewayConfig): Promise<void> => {
            this.logger.info('Config changed, reloading');
            try {
                // Apply the new config directly instead of calling reload()
                // since we already have the new config
                this.config = newConfig;
                this.router = new Router({
                    defaultRouting: newConfig.routing,
                });

                for (const app of newConfig.apps) {
                    this.router.addApp(app);
                }

                for (const name of this.frontdoorRegistry.list()) {
                    const frontdoor = this.frontdoorRegistry.get(name);
                    if (frontdoor) {
                        this.router.addFrontdoor(frontdoor);
                    }
                }

                this.providers.clear();
                for (const providerConfig of newConfig.providers) {
                    try {
                        const provider = this.createProvider(providerConfig);
                        this.providers.set(providerConfig.name, provider);
                    } catch (error) {
                        this.logger.error('Failed to create provider on reload', {
                            provider: providerConfig.name,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }

                this.logger.info('Config reload complete', {
                    apps: newConfig.apps.length,
                    providers: this.providers.size,
                });
            } catch (error) {
                this.logger.error('Failed to reload config', {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        };

        // Start watching (don't await - runs in background)
        this.configProvider
            .watch(onChange, this.watchAbortController.signal)
            .catch((error) => {
                if (error.name !== 'AbortError') {
                    this.logger.error('Config watch error', {
                        error: error instanceof Error ? error.message : String(error),
                    });
                }
            })
            .finally(() => {
                this.isWatching = false;
            });
    }

    /**
     * Stops watching for configuration changes.
     */
    stopWatching(): void {
        if (this.watchAbortController) {
            this.watchAbortController.abort();
            this.watchAbortController = undefined;
            this.isWatching = false;
            this.logger.info('Config watcher stopped');
        }
    }

    /**
     * Whether the gateway is currently watching for config changes.
     */
    get watching(): boolean {
        return this.isWatching;
    }

    /**
     * Handles an HTTP request.
     * This is the main entry point for the gateway.
     */
    async fetch(request: Request): Promise<Response> {
        const interactionId = randomUUID();
        const url = new URL(request.url);
        const path = url.pathname;

        // Ensure config is loaded
        if (!this.config || !this.router) {
            await this.reload();
        }

        // Quick health check
        if (path === '/health' || path === '/healthz') {
            return new Response(JSON.stringify({ status: 'ok' }), {
                status: 200,
                headers: { 'Content-Type': 'application/json' },
            });
        }

        // Authenticate request
        const authHeader = request.headers.get('Authorization');
        const token = extractBearerToken(authHeader);

        if (!token) {
            return this.errorResponse(
                errAuthentication('Missing Authorization header'),
            );
        }

        let auth: AuthContext | null;
        try {
            auth = await this.authProvider.authenticate(token);
        } catch (error) {
            this.logger.error('Authentication error', {
                error: error instanceof Error ? error.message : String(error),
            });
            return this.errorResponse(errAuthentication('Authentication failed'));
        }

        if (!auth) {
            return this.errorResponse(errAuthentication('Invalid API key'));
        }

        // Create request-scoped logger
        const log = requestLogger(this.logger, interactionId, auth.tenantId);

        // Match app and frontdoor
        let app = this.router!.matchApp(path);
        let frontdoor: Frontdoor | undefined;

        if (app) {
            frontdoor = this.router!.getFrontdoor(app);
        } else {
            // Try to match frontdoor directly by path
            frontdoor = this.router!.matchFrontdoor(path);
        }

        if (!frontdoor) {
            return this.errorResponse(errNotFound('No matching endpoint'));
        }

        // Select provider
        // For now, extract model from request body if POST
        let requestModel: string | undefined;
        if (request.method === 'POST') {
            try {
                const clonedRequest = request.clone();
                const body = await clonedRequest.json() as { model?: string };
                requestModel = body.model;
            } catch {
                // Ignore parsing errors, will be caught by frontdoor
            }
        }

        const selection = this.router!.selectProvider(
            requestModel ?? app?.defaultModel ?? '',
            app,
        );

        const provider = this.providers.get(selection.providerName);
        if (!provider) {
            log.error('Provider not found', { provider: selection.providerName });
            return this.errorResponse(
                errServer(`Provider '${selection.providerName}' not configured`),
            );
        }

        // Build frontdoor context
        const ctx: FrontdoorContext = {
            request,
            provider,
            auth,
            app,
            logger: log,
            interactionId,
        };

        // Handle request
        try {
            const result = await frontdoor.handle(ctx);

            // TODO: Publish events, store interaction, trigger shadow mode

            return result.response;
        } catch (error) {
            log.error('Request handling failed', {
                error: error instanceof Error ? error.message : String(error),
            });

            if (error instanceof APIError) {
                return this.errorResponse(error);
            }

            return this.errorResponse(
                errServer(error instanceof Error ? error.message : 'Internal error'),
            );
        }
    }

    /**
     * Creates a provider from configuration.
     */
    private createProvider(config: ProviderConfig): Provider {
        return this.providerRegistry.create(config.type, {
            name: config.name,
            apiKey: config.apiKey,
            baseUrl: config.baseUrl,
        });
    }

    /**
     * Creates an error response.
     */
    private errorResponse(error: APIError): Response {
        const body = JSON.stringify(toOpenAIError(error));
        return new Response(body, {
            status: error.statusCode,
            headers: { 'Content-Type': 'application/json' },
        });
    }
}
