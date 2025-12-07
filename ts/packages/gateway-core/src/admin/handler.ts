/**
 * Admin handler for control plane API.
 *
 * Provides REST endpoints for gateway administration:
 * - /api/stats - System statistics
 * - /api/overview - Configuration overview
 * - /api/interactions - List/view interactions
 * - /api/threads - List/view threads
 * - /api/responses - List/view responses
 *
 * @module admin/handler
 */

import type { StorageProvider } from '../ports/storage.js';
import type { ConfigProvider } from '../ports/config.js';
import type { Logger } from '../utils/logging.js';

// ============================================================================
// Types
// ============================================================================

/**
 * Admin handler options.
 */
export interface AdminHandlerOptions {
    /** Storage provider. */
    storage?: StorageProvider | undefined;

    /** Config provider. */
    config?: ConfigProvider | undefined;

    /** Logger. */
    logger?: Logger | undefined;

    /** Gateway start time. */
    startTime?: Date | undefined;
}

/**
 * Stats response.
 */
export interface StatsResponse {
    uptime: string;
    uptimeMs: number;
    runtime: string;
    memory?: MemoryStats | undefined;
}

/**
 * Memory statistics.
 */
export interface MemoryStats {
    heapUsed?: number | undefined;
    heapTotal?: number | undefined;
    external?: number | undefined;
    rss?: number | undefined;
}

/**
 * Overview response.
 */
export interface OverviewResponse {
    mode: 'single-tenant' | 'multi-tenant';
    storage: StorageSummary;
    apps: AppSummary[];
    providers: ProviderSummary[];
    frontdoors: FrontdoorSummary[];
    routing: AdminRoutingSummary;
}

/**
 * Storage summary.
 */
export interface StorageSummary {
    enabled: boolean;
    type: string;
}

/**
 * App summary.
 */
export interface AppSummary {
    name: string;
    frontdoor: string;
    path: string;
    provider?: string | undefined;
    defaultModel?: string | undefined;
}

/**
 * Provider summary.
 */
export interface ProviderSummary {
    name: string;
    type: string;
    baseUrl?: string | undefined;
}

/**
 * Frontdoor summary.
 */
export interface FrontdoorSummary {
    type: string;
    path: string;
}

/**
 * Routing summary.
 */
export interface AdminRoutingSummary {
    defaultProvider?: string | undefined;
    rules: AdminRoutingRule[];
}

/**
 * Routing rule for admin overview.
 */
export interface AdminRoutingRule {
    modelPrefix?: string | undefined;
    modelExact?: string | undefined;
    provider: string;
}

/**
 * Interaction summary for list display.
 */
export interface AdminInteractionSummary {
    id: string;
    type: string;
    status?: string | undefined;
    model?: string | undefined;
    provider?: string | undefined;
    durationMs?: number | undefined;
    createdAt: number;
    updatedAt: number;
}

/**
 * Interactions list response.
 */
export interface AdminInteractionsListResponse {
    interactions: AdminInteractionSummary[];
    total: number;
}

// ============================================================================
// Admin Handler
// ============================================================================

/**
 * Handles admin API requests.
 */
export class AdminHandler {
    private readonly storage?: StorageProvider;
    private readonly config?: ConfigProvider;
    private readonly logger?: Logger;
    private readonly startTime: Date;

    constructor(options: AdminHandlerOptions = {}) {
        this.storage = options.storage;
        this.config = options.config;
        this.logger = options.logger;
        this.startTime = options.startTime ?? new Date();
    }

    /**
     * Handles an admin API request.
     */
    async handle(request: Request): Promise<Response> {
        const url = new URL(request.url);
        const path = url.pathname;
        const method = request.method;

        try {
            // GET /api/stats
            if (method === 'GET' && path === '/api/stats') {
                return this.handleStats();
            }

            // GET /api/overview
            if (method === 'GET' && path === '/api/overview') {
                return this.handleOverview();
            }

            // GET /api/interactions
            if (method === 'GET' && path === '/api/interactions') {
                const limit = parseInt(url.searchParams.get('limit') ?? '50', 10);
                const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);
                return this.handleListInteractions({ limit, offset });
            }

            // GET /api/interactions/:id
            const interactionMatch = path.match(/^\/api\/interactions\/([^/]+)$/);
            if (method === 'GET' && interactionMatch) {
                return this.handleGetInteraction(interactionMatch[1]!);
            }

            // GET /api/threads
            if (method === 'GET' && path === '/api/threads') {
                const limit = parseInt(url.searchParams.get('limit') ?? '50', 10);
                const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);
                return this.handleListThreads({ limit, offset });
            }

            // GET /api/threads/:id
            const threadMatch = path.match(/^\/api\/threads\/([^/]+)$/);
            if (method === 'GET' && threadMatch) {
                return this.handleGetThread(threadMatch[1]!);
            }

            // GET /api/responses
            if (method === 'GET' && path === '/api/responses') {
                const limit = parseInt(url.searchParams.get('limit') ?? '50', 10);
                const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);
                return this.handleListResponses({ limit, offset });
            }

            // GET /api/responses/:id
            const responseMatch = path.match(/^\/api\/responses\/([^/]+)$/);
            if (method === 'GET' && responseMatch) {
                return this.handleGetResponse(responseMatch[1]!);
            }

            // GET /api/health
            if (method === 'GET' && (path === '/api/health' || path === '/health')) {
                return this.jsonResponse({ status: 'ok' });
            }

            return this.errorResponse(404, 'Not Found');
        } catch (error) {
            this.logger?.error('Admin API error', {
                path,
                error: error instanceof Error ? error.message : String(error),
            });
            return this.errorResponse(
                500,
                error instanceof Error ? error.message : 'Internal error',
            );
        }
    }

    /**
     * Checks if a path is an admin API path.
     */
    matches(path: string): boolean {
        return path.startsWith('/api/') || path === '/health';
    }

    // ---- Endpoint Handlers ----

    private handleStats(): Response {
        const now = Date.now();
        const uptimeMs = now - this.startTime.getTime();

        const stats: StatsResponse = {
            uptime: this.formatDuration(uptimeMs),
            uptimeMs,
            runtime: this.getRuntime(),
        };

        // Add memory stats if available (Node.js)
        if (typeof process !== 'undefined' && process.memoryUsage) {
            const mem = process.memoryUsage();
            stats.memory = {
                heapUsed: mem.heapUsed,
                heapTotal: mem.heapTotal,
                external: mem.external,
                rss: mem.rss,
            };
        }

        return this.jsonResponse(stats);
    }

    private async handleOverview(): Promise<Response> {
        const overview: OverviewResponse = {
            mode: 'single-tenant',
            storage: {
                enabled: this.storage !== undefined,
                type: this.storage ? 'configured' : 'none',
            },
            apps: [],
            providers: [],
            frontdoors: [],
            routing: {
                rules: [],
            },
        };

        // Get config if available - ConfigProvider doesn't have getConfig, skip for now
        // TODO: Add config introspection method to ConfigProvider



        return this.jsonResponse(overview);
    }

    private async handleListInteractions(options: {
        limit: number;
        offset: number;
    }): Promise<Response> {
        if (!this.storage) {
            return this.errorResponse(503, 'Storage not configured');
        }

        // For now, return empty list - would need InteractionStore interface
        const response: AdminInteractionsListResponse = {
            interactions: [],
            total: 0,
        };

        return this.jsonResponse(response);
    }

    private async handleGetInteraction(id: string): Promise<Response> {
        if (!this.storage) {
            return this.errorResponse(503, 'Storage not configured');
        }

        // Would need InteractionStore interface
        return this.errorResponse(404, 'Interaction not found');
    }

    private async handleListThreads(options: {
        limit: number;
        offset: number;
    }): Promise<Response> {
        if (!this.storage?.listConversations) {
            return this.errorResponse(503, 'Thread storage not configured');
        }

        const conversations = await this.storage.listConversations('default', {
            limit: options.limit,
            offset: options.offset,
        });

        const threads = conversations.map((conv) => ({
            id: conv.id,
            createdAt: conv.createdAt.getTime(),
            updatedAt: conv.updatedAt?.getTime() ?? conv.createdAt.getTime(),
            messageCount: 0, // Would need to query messages
        }));

        return this.jsonResponse({ threads, total: threads.length });
    }

    private async handleGetThread(id: string): Promise<Response> {
        if (!this.storage?.getConversation) {
            return this.errorResponse(503, 'Thread storage not configured');
        }

        const conv = await this.storage.getConversation(id);
        if (!conv) {
            return this.errorResponse(404, 'Thread not found');
        }

        return this.jsonResponse({
            id: conv.id,
            createdAt: conv.createdAt.getTime(),
            updatedAt: conv.updatedAt?.getTime() ?? conv.createdAt.getTime(),
            metadata: conv.metadata,
        });
    }

    private async handleListResponses(options: {
        limit: number;
        offset: number;
    }): Promise<Response> {
        if (!this.storage?.listResponses) {
            return this.errorResponse(503, 'Response storage not configured');
        }

        const records = await this.storage.listResponses('default', {
            limit: options.limit,
            offset: options.offset,
        });

        const responses = records.map((r) => ({
            id: r.id,
            status: r.status,
            model: r.model,
            createdAt: r.createdAt.getTime(),
            updatedAt: r.updatedAt?.getTime() ?? r.createdAt.getTime(),
        }));

        return this.jsonResponse({ responses, total: responses.length });
    }

    private async handleGetResponse(id: string): Promise<Response> {
        if (!this.storage?.getResponse) {
            return this.errorResponse(503, 'Response storage not configured');
        }

        const record = await this.storage.getResponse(id);
        if (!record) {
            return this.errorResponse(404, 'Response not found');
        }

        return this.jsonResponse({
            id: record.id,
            status: record.status,
            model: record.model,
            createdAt: record.createdAt.getTime(),
            updatedAt: record.updatedAt?.getTime() ?? record.createdAt.getTime(),
        });
    }

    // ---- Helpers ----

    private jsonResponse(data: unknown, status = 200): Response {
        return new Response(JSON.stringify(data), {
            status,
            headers: { 'Content-Type': 'application/json' },
        });
    }

    private errorResponse(status: number, message: string): Response {
        return new Response(JSON.stringify({ error: message }), {
            status,
            headers: { 'Content-Type': 'application/json' },
        });
    }

    private formatDuration(ms: number): string {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);

        if (days > 0) {
            return `${days}d ${hours % 24}h ${minutes % 60}m`;
        }
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
        }
        if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        }
        return `${seconds}s`;
    }

    private getRuntime(): string {
        // Detect runtime
        if (typeof Deno !== 'undefined') {
            return 'Deno';
        }
        if (typeof Bun !== 'undefined') {
            return 'Bun';
        }
        if (typeof process !== 'undefined' && process.versions?.node) {
            return `Node.js v${process.versions.node}`;
        }
        // Check for Cloudflare Workers (use a different check)
        if (typeof globalThis !== 'undefined' && 'caches' in globalThis) {
            return 'Cloudflare Workers';
        }
        return 'Unknown';
    }
}

// Declare globals for runtime detection
declare const Deno: unknown;
declare const Bun: unknown;
