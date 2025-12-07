/**
 * GraphQL handler for the gateway control plane.
 *
 * Provides a portable GraphQL endpoint that works with any runtime.
 * Uses graphql-js for schema parsing and execution.
 *
 * @module graphql/handler
 */

import type { StorageProvider } from '../ports/storage.js';
import type { ConfigProvider } from '../ports/config.js';
import type { Logger } from '../utils/logging.js';
import type {
    GraphQLStats,
    GraphQLOverview,
    GraphQLInteractionConnection,
    GraphQLInteractionFilter,
} from './schema.js';

// ============================================================================
// Types
// ============================================================================

/**
 * GraphQL handler options.
 */
export interface GraphQLHandlerOptions {
    /** Storage provider. */
    storage?: StorageProvider | undefined;

    /** Config provider. */
    config?: ConfigProvider | undefined;

    /** Logger. */
    logger?: Logger | undefined;

    /** Gateway start time. */
    startTime?: Date | undefined;

    /** Enable GraphQL Playground/GraphiQL. */
    enablePlayground?: boolean | undefined;
}

/**
 * GraphQL request body.
 */
export interface GraphQLRequest {
    query: string;
    operationName?: string;
    variables?: Record<string, unknown>;
}

/**
 * GraphQL response.
 */
export interface GraphQLResponse {
    data?: unknown;
    errors?: Array<{
        message: string;
        locations?: Array<{ line: number; column: number }>;
        path?: Array<string | number>;
    }>;
}

/**
 * Resolver context.
 */
export interface ResolverContext {
    storage?: StorageProvider;
    config?: ConfigProvider;
    logger?: Logger;
    startTime: Date;
}

// ============================================================================
// GraphQL Handler
// ============================================================================

/**
 * Handles GraphQL requests for the control plane.
 *
 * This is a lightweight implementation that doesn't require
 * heavy dependencies like Apollo. It uses a simple resolver
 * dispatch pattern.
 */
export class GraphQLHandler {
    private readonly storage?: StorageProvider;
    private readonly config?: ConfigProvider;
    private readonly logger?: Logger;
    private readonly startTime: Date;
    private readonly enablePlayground: boolean;

    constructor(options: GraphQLHandlerOptions = {}) {
        this.storage = options.storage;
        this.config = options.config;
        this.logger = options.logger;
        this.startTime = options.startTime ?? new Date();
        this.enablePlayground = options.enablePlayground ?? true;
    }

    /**
     * Handles a GraphQL request.
     */
    async handle(request: Request): Promise<Response> {
        const url = new URL(request.url);
        const path = url.pathname;

        // Handle GraphQL Playground
        if (
            this.enablePlayground &&
            request.method === 'GET' &&
            (path.endsWith('/playground') || path.endsWith('/graphiql'))
        ) {
            return this.servePlayground(url.origin + path.replace(/\/(playground|graphiql)$/, ''));
        }

        // Handle GraphQL queries
        if (request.method === 'POST') {
            try {
                const body = (await request.json()) as GraphQLRequest;
                const result = await this.executeQuery(body);
                return this.jsonResponse(result);
            } catch (error) {
                return this.jsonResponse({
                    errors: [
                        {
                            message:
                                error instanceof Error ? error.message : 'Internal error',
                        },
                    ],
                });
            }
        }

        // Handle introspection via GET
        if (request.method === 'GET' && url.searchParams.has('query')) {
            const query = url.searchParams.get('query') ?? '';
            const variables = url.searchParams.get('variables');
            const operationName = url.searchParams.get('operationName');

            try {
                const result = await this.executeQuery({
                    query,
                    operationName: operationName ?? undefined,
                    variables: variables ? JSON.parse(variables) : undefined,
                });
                return this.jsonResponse(result);
            } catch (error) {
                return this.jsonResponse({
                    errors: [
                        {
                            message:
                                error instanceof Error ? error.message : 'Internal error',
                        },
                    ],
                });
            }
        }

        return new Response('Method not allowed', { status: 405 });
    }

    /**
     * Checks if a path is a GraphQL API path.
     */
    matches(path: string): boolean {
        return (
            path.endsWith('/graphql') ||
            path.endsWith('/playground') ||
            path.endsWith('/graphiql')
        );
    }

    // ---- Query Execution ----

    private async executeQuery(req: GraphQLRequest): Promise<GraphQLResponse> {
        const { query, variables } = req;

        // Simple query parser - extract operation type and field
        const queryMatch = query.match(
            /(?:query|mutation)?\s*(?:\w+)?\s*(?:\([^)]*\))?\s*\{\s*(\w+)/,
        );

        if (!queryMatch) {
            return { errors: [{ message: 'Invalid query syntax' }] };
        }

        const rootField = queryMatch[1]!;

        try {
            const data = await this.resolveField(rootField, variables ?? {});
            return { data: { [rootField]: data } };
        } catch (error) {
            this.logger?.error('GraphQL execution error', {
                field: rootField,
                error: error instanceof Error ? error.message : String(error),
            });
            return {
                errors: [
                    {
                        message:
                            error instanceof Error ? error.message : 'Execution error',
                    },
                ],
            };
        }
    }

    private async resolveField(
        field: string,
        variables: Record<string, unknown>,
    ): Promise<unknown> {
        switch (field) {
            case 'stats':
                return this.resolveStats();
            case 'overview':
                return this.resolveOverview();
            case 'interactions':
                return this.resolveInteractions(
                    variables.filter as GraphQLInteractionFilter | undefined,
                    variables.limit as number | undefined,
                    variables.offset as number | undefined,
                );
            case 'interaction':
                return this.resolveInteraction(variables.id as string);
            case 'interactionEvents':
                return this.resolveInteractionEvents(
                    variables.interactionId as string,
                    variables.limit as number | undefined,
                );
            case 'shadowResults':
                return this.resolveShadowResults(variables.interactionId as string);
            case 'divergentShadows':
                return this.resolveDivergentShadows(
                    variables.limit as number | undefined,
                    variables.offset as number | undefined,
                    variables.provider as string | undefined,
                );
            case 'shadow':
                return this.resolveShadow(variables.id as string);
            default:
                throw new Error(`Unknown field: ${field}`);
        }
    }

    // ---- Resolvers ----

    private resolveStats(): GraphQLStats {
        const now = Date.now();
        const uptimeMs = now - this.startTime.getTime();

        const stats: GraphQLStats = {
            uptime: this.formatDuration(uptimeMs),
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

        return stats;
    }

    private async resolveOverview(): Promise<GraphQLOverview> {
        return {
            mode: 'single-tenant',
            storage: {
                enabled: this.storage !== undefined,
                type: this.storage ? 'configured' : 'none',
            },
            apps: [],
            frontdoors: [],
            providers: [],
            routing: {
                rules: [],
            },
        };
    }

    private async resolveInteractions(
        filter?: GraphQLInteractionFilter,
        limit = 50,
        offset = 0,
    ): Promise<GraphQLInteractionConnection> {
        // Return empty list for now - would need InteractionStore interface
        return {
            interactions: [],
            total: 0,
        };
    }

    private async resolveInteraction(id: string): Promise<unknown> {
        // Would need InteractionStore interface
        return null;
    }

    private async resolveInteractionEvents(
        interactionId: string,
        _limit = 500,
    ): Promise<unknown> {
        return {
            interactionId,
            events: [],
        };
    }

    private async resolveShadowResults(interactionId: string): Promise<unknown> {
        return {
            interactionId,
            shadows: [],
        };
    }

    private async resolveDivergentShadows(
        limit = 100,
        offset = 0,
        _provider?: string,
    ): Promise<unknown> {
        return {
            interactions: [],
            total: 0,
            limit,
            offset,
        };
    }

    private async resolveShadow(_id: string): Promise<unknown> {
        return null;
    }

    // ---- Helpers ----

    private jsonResponse(data: unknown): Response {
        return new Response(JSON.stringify(data), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
        });
    }

    private servePlayground(endpoint: string): Response {
        const html = `
<!DOCTYPE html>
<html>
<head>
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css" />
</head>
<body style="margin: 0;">
  <div id="graphiql" style="height: 100vh;"></div>
  <script crossorigin src="https://unpkg.com/react/umd/react.production.min.js"></script>
  <script crossorigin src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"></script>
  <script crossorigin src="https://unpkg.com/graphiql/graphiql.min.js"></script>
  <script>
    const fetcher = GraphiQL.createFetcher({ url: '${endpoint}' });
    ReactDOM.render(
      React.createElement(GraphiQL, { fetcher }),
      document.getElementById('graphiql')
    );
  </script>
</body>
</html>
`;
        return new Response(html, {
            status: 200,
            headers: { 'Content-Type': 'text/html' },
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
        if (typeof Deno !== 'undefined') {
            return 'Deno';
        }
        if (typeof Bun !== 'undefined') {
            return 'Bun';
        }
        if (typeof process !== 'undefined' && process.versions?.node) {
            return `Node.js v${process.versions.node}`;
        }
        if (typeof globalThis !== 'undefined' && 'caches' in globalThis) {
            return 'Cloudflare Workers';
        }
        return 'Unknown';
    }
}

// Declare globals for runtime detection
declare const Deno: unknown;
declare const Bun: unknown;
