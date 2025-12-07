/**
 * Stats API handler - returns runtime statistics.
 */

import type { Env } from '../bindings.js';

interface StatsResponse {
    uptime: number;
    timestamp: string;
    version: string;
    environment: string;
}

export async function statsHandler(request: Request, env: Env): Promise<Response> {
    const stats: StatsResponse = {
        uptime: 0, // Workers don't have process uptime
        timestamp: new Date().toISOString(),
        version: '0.1.0',
        environment: env.ENVIRONMENT ?? 'unknown',
    };

    return new Response(JSON.stringify(stats), {
        headers: { 'Content-Type': 'application/json' },
    });
}
