/**
 * Shadows API handler - list shadow results and divergent shadows.
 */

import type { Env } from '../bindings.js';

interface ShadowSummary {
    id: string;
    interactionId: string;
    providerName: string;
    durationMs: number;
    hasStructuralDivergence: boolean;
    divergenceCount: number;
    createdAt: string;
}

interface ShadowDetail extends ShadowSummary {
    request: unknown;
    response?: unknown;
    error?: unknown;
    divergences: unknown[];
}

export async function shadowsHandler(
    request: Request,
    env: Env,
    path: string,
): Promise<Response> {
    // Extract ID from path: /api/shadows/{id}
    const parts = path.split('/');
    const id = parts[parts.length - 1];

    if (!id) {
        return new Response(JSON.stringify({ error: 'Missing shadow ID' }), {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
        });
    }

    try {
        const result = await env.DB.prepare(`
      SELECT * FROM shadow_results WHERE id = ?
    `)
            .bind(id)
            .first();

        if (!result) {
            return new Response(JSON.stringify({ error: 'Shadow result not found' }), {
                status: 404,
                headers: { 'Content-Type': 'application/json' },
            });
        }

        const divergences = result.divergences ? JSON.parse(String(result.divergences)) : [];

        const detail: ShadowDetail = {
            id: String(result.id),
            interactionId: String(result.interaction_id),
            providerName: String(result.provider_name),
            durationMs: Number(result.duration_ms ?? 0),
            hasStructuralDivergence: Boolean(result.has_structural_divergence),
            divergenceCount: divergences.length,
            createdAt: String(result.created_at),
            request: result.request ? JSON.parse(String(result.request)) : null,
            response: result.response ? JSON.parse(String(result.response)) : null,
            error: result.error ? JSON.parse(String(result.error)) : null,
            divergences,
        };

        return new Response(JSON.stringify(detail), {
            headers: { 'Content-Type': 'application/json' },
        });
    } catch (error) {
        return new Response(
            JSON.stringify({
                error: error instanceof Error ? error.message : 'Database error',
            }),
            {
                status: 500,
                headers: { 'Content-Type': 'application/json' },
            },
        );
    }
}

export async function divergentShadowsHandler(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const limit = parseInt(url.searchParams.get('limit') ?? '50', 10);
    const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);

    try {
        const result = await env.DB.prepare(`
      SELECT id, interaction_id, provider_name, duration_ms, 
             has_structural_divergence, divergences, created_at
      FROM shadow_results
      WHERE has_structural_divergence = 1
      ORDER BY created_at DESC
      LIMIT ? OFFSET ?
    `)
            .bind(limit, offset)
            .all();

        const shadows: ShadowSummary[] = (result.results ?? []).map((row) => {
            const divergences = row.divergences ? JSON.parse(String(row.divergences)) : [];
            return {
                id: String(row.id),
                interactionId: String(row.interaction_id),
                providerName: String(row.provider_name),
                durationMs: Number(row.duration_ms ?? 0),
                hasStructuralDivergence: Boolean(row.has_structural_divergence),
                divergenceCount: divergences.length,
                createdAt: String(row.created_at),
            };
        });

        // Get total count
        const countResult = await env.DB.prepare(`
      SELECT COUNT(*) as total FROM shadow_results WHERE has_structural_divergence = 1
    `).first();

        return new Response(
            JSON.stringify({
                shadows,
                total: Number(countResult?.total ?? 0),
                limit,
                offset,
            }),
            { headers: { 'Content-Type': 'application/json' } },
        );
    } catch (error) {
        return new Response(
            JSON.stringify({
                shadows: [],
                error: error instanceof Error ? error.message : 'Database error',
            }),
            { headers: { 'Content-Type': 'application/json' } },
        );
    }
}
