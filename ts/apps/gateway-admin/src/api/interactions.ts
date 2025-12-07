/**
 * Interactions API handler - list and get stored interactions.
 */

import type { Env } from '../bindings.js';

interface InteractionSummary {
    id: string;
    type: 'conversation' | 'response';
    model: string;
    createdAt: string;
    updatedAt: string;
    messageCount?: number;
    status?: string;
}

interface InteractionDetail extends InteractionSummary {
    request?: unknown;
    response?: unknown;
    messages?: unknown[];
    metadata?: Record<string, string>;
}

export async function interactionsHandler(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const limit = parseInt(url.searchParams.get('limit') ?? '50', 10);
    const offset = parseInt(url.searchParams.get('offset') ?? '0', 10);
    const typeFilter = url.searchParams.get('type');

    try {
        const interactions: InteractionSummary[] = [];

        // Query conversations
        if (!typeFilter || typeFilter === 'conversation') {
            const convResult = await env.DB.prepare(`
        SELECT id, model, created_at, updated_at,
               (SELECT COUNT(*) FROM messages WHERE conversation_id = conversations.id) as message_count
        FROM conversations
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?
      `)
                .bind(limit, offset)
                .all();

            for (const row of convResult.results ?? []) {
                interactions.push({
                    id: String(row.id),
                    type: 'conversation',
                    model: String(row.model ?? ''),
                    createdAt: String(row.created_at),
                    updatedAt: String(row.updated_at),
                    messageCount: Number(row.message_count ?? 0),
                });
            }
        }

        // Query responses
        if (!typeFilter || typeFilter === 'response') {
            const respResult = await env.DB.prepare(`
        SELECT id, model, status, created_at, updated_at
        FROM responses
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?
      `)
                .bind(limit, offset)
                .all();

            for (const row of respResult.results ?? []) {
                interactions.push({
                    id: String(row.id),
                    type: 'response',
                    model: String(row.model ?? ''),
                    createdAt: String(row.created_at),
                    updatedAt: String(row.updated_at),
                    status: String(row.status ?? ''),
                });
            }
        }

        // Sort by updatedAt
        interactions.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());

        return new Response(
            JSON.stringify({
                interactions: interactions.slice(0, limit),
                total: interactions.length,
                limit,
                offset,
            }),
            { headers: { 'Content-Type': 'application/json' } },
        );
    } catch (error) {
        return new Response(
            JSON.stringify({
                interactions: [],
                error: error instanceof Error ? error.message : 'Database error',
            }),
            { headers: { 'Content-Type': 'application/json' } },
        );
    }
}

export async function interactionDetailHandler(
    request: Request,
    env: Env,
    path: string,
): Promise<Response> {
    // Extract ID from path: /api/interactions/{id}
    const parts = path.split('/');
    const id = parts[parts.length - 1];

    if (!id) {
        return new Response(JSON.stringify({ error: 'Missing interaction ID' }), {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
        });
    }

    try {
        // Try conversations first
        const convResult = await env.DB.prepare(`
      SELECT * FROM conversations WHERE id = ?
    `)
            .bind(id)
            .first();

        if (convResult) {
            // Get messages
            const messagesResult = await env.DB.prepare(`
        SELECT * FROM messages WHERE conversation_id = ? ORDER BY created_at
      `)
                .bind(id)
                .all();

            return new Response(
                JSON.stringify({
                    id: convResult.id,
                    type: 'conversation',
                    model: convResult.model,
                    createdAt: convResult.created_at,
                    updatedAt: convResult.updated_at,
                    messages: messagesResult.results ?? [],
                    metadata: convResult.metadata ? JSON.parse(String(convResult.metadata)) : {},
                }),
                { headers: { 'Content-Type': 'application/json' } },
            );
        }

        // Try responses
        const respResult = await env.DB.prepare(`
      SELECT * FROM responses WHERE id = ?
    `)
            .bind(id)
            .first();

        if (respResult) {
            return new Response(
                JSON.stringify({
                    id: respResult.id,
                    type: 'response',
                    model: respResult.model,
                    status: respResult.status,
                    createdAt: respResult.created_at,
                    updatedAt: respResult.updated_at,
                    request: respResult.request ? JSON.parse(String(respResult.request)) : null,
                    response: respResult.response ? JSON.parse(String(respResult.response)) : null,
                    metadata: respResult.metadata ? JSON.parse(String(respResult.metadata)) : {},
                }),
                { headers: { 'Content-Type': 'application/json' } },
            );
        }

        return new Response(JSON.stringify({ error: 'Interaction not found' }), {
            status: 404,
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
