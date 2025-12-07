/**
 * Gateway Admin Worker - Admin API for gateway observability.
 *
 * Provides endpoints for:
 * - /api/stats - Runtime statistics
 * - /api/overview - Gateway configuration summary
 * - /api/interactions - List/get interactions
 * - /api/shadows - Shadow mode results
 */

import type { Env } from './bindings.js';
import { statsHandler } from './api/stats.js';
import { overviewHandler } from './api/overview.js';
import { interactionsHandler, interactionDetailHandler } from './api/interactions.js';
import { shadowsHandler, divergentShadowsHandler } from './api/shadows.js';

export default {
    async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
        const url = new URL(request.url);
        const path = url.pathname;

        // CORS headers for all responses
        const corsHeaders = {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
            'Access-Control-Allow-Headers': 'Content-Type, Authorization',
        };

        // Handle preflight
        if (request.method === 'OPTIONS') {
            return new Response(null, { headers: corsHeaders });
        }

        try {
            let response: Response;

            // Route to handlers
            if (path === '/api/stats') {
                response = await statsHandler(request, env);
            } else if (path === '/api/overview') {
                response = await overviewHandler(request, env);
            } else if (path === '/api/interactions') {
                response = await interactionsHandler(request, env);
            } else if (path.startsWith('/api/interactions/')) {
                response = await interactionDetailHandler(request, env, path);
            } else if (path === '/api/shadows/divergent') {
                response = await divergentShadowsHandler(request, env);
            } else if (path.startsWith('/api/shadows/')) {
                response = await shadowsHandler(request, env, path);
            } else if (path === '/health') {
                response = new Response(JSON.stringify({ status: 'ok' }), {
                    headers: { 'Content-Type': 'application/json' },
                });
            } else {
                response = new Response(JSON.stringify({ error: 'Not found' }), {
                    status: 404,
                    headers: { 'Content-Type': 'application/json' },
                });
            }

            // Add CORS headers to response
            const newHeaders = new Headers(response.headers);
            Object.entries(corsHeaders).forEach(([k, v]) => newHeaders.set(k, v));

            return new Response(response.body, {
                status: response.status,
                headers: newHeaders,
            });
        } catch (error) {
            console.error('Admin API error:', error);
            return new Response(
                JSON.stringify({
                    error: error instanceof Error ? error.message : 'Internal error',
                }),
                {
                    status: 500,
                    headers: { 'Content-Type': 'application/json', ...corsHeaders },
                },
            );
        }
    },
};
