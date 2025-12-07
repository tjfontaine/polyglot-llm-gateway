/**
 * Request router for the gateway.
 *
 * @module router
 */

import type { AppConfig, RoutingConfig, RoutingRule, ModelRoutingConfig } from './ports/config.js';
import type { Provider, ProviderFactoryConfig } from './ports/provider.js';
import type { Frontdoor } from './frontdoors/types.js';

// ============================================================================
// Route Types
// ============================================================================

/**
 * A matched route.
 */
export interface Route {
    /** Matched app configuration. */
    app: AppConfig;

    /** Frontdoor to use. */
    frontdoor: Frontdoor;

    /** Provider name to use. */
    providerName: string;

    /** Model to use (may be rewritten). */
    model?: string | undefined;

    /** Whether to rewrite the response model to the original. */
    rewriteResponseModel?: boolean | undefined;
}

/**
 * Provider selection result.
 */
export interface ProviderSelection {
    /** Provider name. */
    providerName: string;

    /** Model to use (may be rewritten). */
    model?: string | undefined;

    /** Whether to rewrite the response model. */
    rewriteResponseModel?: boolean | undefined;
}

// ============================================================================
// Router
// ============================================================================

/**
 * Routes requests to the appropriate app, frontdoor, and provider.
 */
export class Router {
    private readonly apps: Map<string, AppConfig> = new Map();
    private readonly frontdoors: Map<string, Frontdoor> = new Map();
    private readonly defaultRouting: RoutingConfig | undefined;

    constructor(options?: { defaultRouting?: RoutingConfig | undefined }) {
        this.defaultRouting = options?.defaultRouting;
    }

    /**
     * Adds an app configuration.
     */
    addApp(app: AppConfig): void {
        this.apps.set(app.name, app);
    }

    /**
     * Adds a frontdoor.
     */
    addFrontdoor(frontdoor: Frontdoor): void {
        this.frontdoors.set(frontdoor.name, frontdoor);
    }

    /**
     * Matches a request path to an app.
     */
    matchApp(path: string): AppConfig | undefined {
        // Find app with longest matching path prefix
        let bestMatch: AppConfig | undefined;
        let bestMatchLength = 0;

        for (const app of this.apps.values()) {
            const appPath = app.path.replace(/\/$/, ''); // Remove trailing slash
            if (path.startsWith(appPath) && appPath.length > bestMatchLength) {
                bestMatch = app;
                bestMatchLength = appPath.length;
            }
        }

        return bestMatch;
    }

    /**
     * Gets the frontdoor for an app or path.
     */
    getFrontdoor(app: AppConfig): Frontdoor | undefined {
        return this.frontdoors.get(app.frontdoor);
    }

    /**
     * Matches a frontdoor by path.
     */
    matchFrontdoor(path: string): Frontdoor | undefined {
        for (const frontdoor of this.frontdoors.values()) {
            if (frontdoor.matches(path)) {
                return frontdoor;
            }
        }
        return undefined;
    }

    /**
     * Selects a provider based on model and routing configuration.
     */
    selectProvider(
        model: string,
        app?: AppConfig,
        defaultProvider?: string,
    ): ProviderSelection {
        // 1. Check app-level forced provider
        if (app?.provider) {
            return { providerName: app.provider };
        }

        // 2. Check app-level model routing
        if (app?.modelRouting) {
            const selection = this.matchModelRouting(model, app.modelRouting);
            if (selection) {
                return selection;
            }
        }

        // 3. Check global routing rules
        if (this.defaultRouting?.rules) {
            for (const rule of this.defaultRouting.rules) {
                if (this.matchesRoutingRule(model, rule)) {
                    return { providerName: rule.provider };
                }
            }
        }

        // 4. Use default provider
        const provider =
            defaultProvider ??
            this.defaultRouting?.defaultProvider ??
            'openai';

        return { providerName: provider };
    }

    /**
     * Matches model routing configuration.
     */
    private matchModelRouting(
        model: string,
        routing: ModelRoutingConfig,
    ): ProviderSelection | undefined {
        // Check prefix providers
        if (routing.prefixProviders) {
            for (const [prefix, provider] of Object.entries(routing.prefixProviders)) {
                if (model.startsWith(prefix)) {
                    return { providerName: provider };
                }
            }
        }

        // Check rewrites
        if (routing.rewrites) {
            for (const rewrite of routing.rewrites) {
                if (rewrite.modelExact && model === rewrite.modelExact) {
                    return {
                        providerName: rewrite.provider ?? 'openai',
                        model: rewrite.model,
                        rewriteResponseModel: rewrite.rewriteResponseModel,
                    };
                }
                if (rewrite.modelPrefix && model.startsWith(rewrite.modelPrefix)) {
                    return {
                        providerName: rewrite.provider ?? 'openai',
                        model: rewrite.model,
                        rewriteResponseModel: rewrite.rewriteResponseModel,
                    };
                }
            }
        }

        // Check fallback
        if (routing.fallback) {
            return {
                providerName: routing.fallback.provider ?? 'openai',
                model: routing.fallback.model,
                rewriteResponseModel: routing.fallback.rewriteResponseModel,
            };
        }

        return undefined;
    }

    /**
     * Checks if a model matches a routing rule.
     */
    private matchesRoutingRule(model: string, rule: RoutingRule): boolean {
        if (rule.modelExact && model === rule.modelExact) {
            return true;
        }
        if (rule.modelPrefix && model.startsWith(rule.modelPrefix)) {
            return true;
        }
        return false;
    }
}

// ============================================================================
// Path Utilities
// ============================================================================

/**
 * Strips the app prefix from a path.
 */
export function stripAppPrefix(path: string, app: AppConfig): string {
    const appPath = app.path.replace(/\/$/, '');
    if (path.startsWith(appPath)) {
        return path.slice(appPath.length) || '/';
    }
    return path;
}

/**
 * Joins path segments.
 */
export function joinPath(...segments: string[]): string {
    return segments
        .map((s, i) => {
            if (i === 0) return s.replace(/\/$/, '');
            if (i === segments.length - 1) return s.replace(/^\//, '');
            return s.replace(/^\//, '').replace(/\/$/, '');
        })
        .filter(Boolean)
        .join('/');
}
