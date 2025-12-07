/**
 * Overview API handler - returns gateway configuration summary.
 */

import type { Env } from '../bindings.js';

interface OverviewResponse {
    apps: AppSummary[];
    providers: ProviderSummary[];
    routing: RoutingSummary;
}

interface AppSummary {
    name: string;
    frontdoor: string;
    path: string;
    provider?: string;
}

interface ProviderSummary {
    name: string;
    type: string;
}

interface RoutingSummary {
    defaultProvider?: string;
    rules: RoutingRule[];
}

interface RoutingRule {
    model: string;
    provider: string;
}

export async function overviewHandler(request: Request, env: Env): Promise<Response> {
    try {
        // Try to load config from KV
        const configJson = await env.CONFIG_KV.get('gateway-config');

        if (!configJson) {
            return new Response(
                JSON.stringify({
                    apps: [],
                    providers: [],
                    routing: { rules: [] },
                }),
                { headers: { 'Content-Type': 'application/json' } },
            );
        }

        const config = JSON.parse(configJson);

        const overview: OverviewResponse = {
            apps: (config.apps ?? []).map((app: Record<string, unknown>) => ({
                name: app.name,
                frontdoor: app.frontdoor,
                path: app.path,
                provider: app.provider,
            })),
            providers: (config.providers ?? []).map((p: Record<string, unknown>) => ({
                name: p.name,
                type: p.apiType ?? p.type,
            })),
            routing: {
                defaultProvider: config.routing?.defaultProvider,
                rules: (config.routing?.rules ?? []).map((r: Record<string, unknown>) => ({
                    model: r.model,
                    provider: r.provider,
                })),
            },
        };

        return new Response(JSON.stringify(overview), {
            headers: { 'Content-Type': 'application/json' },
        });
    } catch (error) {
        return new Response(
            JSON.stringify({
                apps: [],
                providers: [],
                routing: { rules: [] },
                error: error instanceof Error ? error.message : 'Failed to load config',
            }),
            { headers: { 'Content-Type': 'application/json' } },
        );
    }
}
