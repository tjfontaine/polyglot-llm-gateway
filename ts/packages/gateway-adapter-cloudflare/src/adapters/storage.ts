/**
 * D1-based storage provider for Cloudflare Workers.
 *
 * @module adapters/storage
 */

import type {
    StorageProvider,
    Conversation,
    StoredMessage,
    ResponseRecord,
    InteractionSummary,
    InteractionListOptions,
    ListOptions,
    DivergenceListOptions,
    ShadowResult,
    InteractionEvent,
} from '@polyglot-llm-gateway/gateway-core';
import { D1_TABLES } from '../bindings.js';

// ============================================================================
// D1 Storage Provider
// ============================================================================

/**
 * Storage provider backed by Cloudflare D1.
 */
export class D1StorageProvider implements StorageProvider {
    constructor(private readonly db: D1Database) { }

    // ---- Conversations ----

    async saveConversation(conversation: Conversation): Promise<void> {
        // Upsert conversation
        await this.db
            .prepare(`
        INSERT INTO ${D1_TABLES.CONVERSATIONS} (id, tenant_id, app_name, model, metadata, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          metadata = excluded.metadata,
          updated_at = excluded.updated_at
      `)
            .bind(
                conversation.id,
                conversation.tenantId,
                conversation.appName ?? null,
                conversation.model ?? null,
                JSON.stringify(conversation.metadata ?? {}),
                conversation.createdAt.toISOString(),
                conversation.updatedAt.toISOString(),
            )
            .run();

        // Save messages
        for (const msg of conversation.messages) {
            await this.db
                .prepare(`
          INSERT INTO ${D1_TABLES.MESSAGES} (id, conversation_id, role, content, usage, timestamp)
          VALUES (?, ?, ?, ?, ?, ?)
          ON CONFLICT(id) DO NOTHING
        `)
                .bind(
                    msg.id,
                    conversation.id,
                    msg.role,
                    msg.content,
                    JSON.stringify(msg.usage ?? null),
                    msg.timestamp.toISOString(),
                )
                .run();
        }
    }

    async getConversation(id: string): Promise<Conversation | null> {
        const row = await this.db
            .prepare(`SELECT * FROM ${D1_TABLES.CONVERSATIONS} WHERE id = ?`)
            .bind(id)
            .first<ConversationRow>();

        if (!row) return null;

        const messages = await this.db
            .prepare(`SELECT * FROM ${D1_TABLES.MESSAGES} WHERE conversation_id = ? ORDER BY timestamp ASC`)
            .bind(id)
            .all<MessageRow>();

        return {
            id: row.id,
            tenantId: row.tenant_id,
            appName: row.app_name ?? undefined,
            model: row.model ?? undefined,
            metadata: JSON.parse(row.metadata || '{}'),
            messages: messages.results.map((m) => ({
                id: m.id,
                role: m.role as StoredMessage['role'],
                content: m.content,
                usage: m.usage ? JSON.parse(m.usage) : undefined,
                timestamp: new Date(m.timestamp),
            })),
            createdAt: new Date(row.created_at),
            updatedAt: new Date(row.updated_at),
        };
    }

    async listConversations(
        tenantId: string,
        options?: ListOptions,
    ): Promise<Conversation[]> {
        const limit = options?.limit ?? 50;
        const offset = options?.offset ?? 0;

        const rows = await this.db
            .prepare(`
        SELECT * FROM ${D1_TABLES.CONVERSATIONS}
        WHERE tenant_id = ?
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?
      `)
            .bind(tenantId, limit, offset)
            .all<ConversationRow>();

        return Promise.all(
            rows.results.map((row) => this.getConversation(row.id).then((c) => c!)),
        );
    }

    // ---- Responses ----

    async saveResponse(response: ResponseRecord): Promise<void> {
        await this.db
            .prepare(`
        INSERT INTO ${D1_TABLES.RESPONSES} (
          id, tenant_id, app_name, thread_key, previous_response_id,
          model, status, request, response, error, usage, metadata, created_at, updated_at
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          status = excluded.status,
          response = excluded.response,
          error = excluded.error,
          usage = excluded.usage,
          updated_at = excluded.updated_at
      `)
            .bind(
                response.id,
                response.tenantId,
                response.appName ?? null,
                response.threadKey ?? null,
                response.previousResponseId ?? null,
                response.model,
                response.status,
                JSON.stringify(response.request ?? null),
                JSON.stringify(response.response ?? null),
                JSON.stringify(response.error ?? null),
                JSON.stringify(response.usage ?? null),
                JSON.stringify(response.metadata ?? {}),
                response.createdAt.toISOString(),
                response.updatedAt.toISOString(),
            )
            .run();
    }

    async getResponse(id: string): Promise<ResponseRecord | null> {
        const row = await this.db
            .prepare(`SELECT * FROM ${D1_TABLES.RESPONSES} WHERE id = ?`)
            .bind(id)
            .first<ResponseRow>();

        if (!row) return null;

        return this.rowToResponse(row);
    }

    async listResponses(
        tenantId: string,
        options?: ListOptions,
    ): Promise<ResponseRecord[]> {
        const limit = options?.limit ?? 50;
        const offset = options?.offset ?? 0;

        const rows = await this.db
            .prepare(`
        SELECT * FROM ${D1_TABLES.RESPONSES}
        WHERE tenant_id = ?
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?
      `)
            .bind(tenantId, limit, offset)
            .all<ResponseRow>();

        return rows.results.map(this.rowToResponse);
    }

    // ---- Interactions ----

    async listInteractions(
        options?: InteractionListOptions,
    ): Promise<InteractionSummary[]> {
        const limit = options?.limit ?? 50;
        const offset = options?.offset ?? 0;

        // Union query across conversations and responses
        const rows = await this.db
            .prepare(`
        SELECT 'conversation' as type, id, tenant_id, app_name, model, created_at, updated_at
        FROM ${D1_TABLES.CONVERSATIONS}
        ${options?.tenantId ? 'WHERE tenant_id = ?' : ''}
        UNION ALL
        SELECT 'response' as type, id, tenant_id, app_name, model, created_at, updated_at
        FROM ${D1_TABLES.RESPONSES}
        ${options?.tenantId ? 'WHERE tenant_id = ?' : ''}
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?
      `)
            .bind(
                ...(options?.tenantId ? [options.tenantId, options.tenantId] : []),
                limit,
                offset,
            )
            .all<InteractionRow>();

        return rows.results.map((row) => ({
            id: row.id,
            type: row.type as 'conversation' | 'response',
            tenantId: row.tenant_id,
            appName: row.app_name ?? undefined,
            model: row.model ?? undefined,
            createdAt: new Date(row.created_at),
            updatedAt: new Date(row.updated_at),
        }));
    }

    async getInteractionCount(options?: InteractionListOptions): Promise<number> {
        const convCount = await this.db
            .prepare(`SELECT COUNT(*) as count FROM ${D1_TABLES.CONVERSATIONS} ${options?.tenantId ? 'WHERE tenant_id = ?' : ''}`)
            .bind(...(options?.tenantId ? [options.tenantId] : []))
            .first<{ count: number }>();

        const respCount = await this.db
            .prepare(`SELECT COUNT(*) as count FROM ${D1_TABLES.RESPONSES} ${options?.tenantId ? 'WHERE tenant_id = ?' : ''}`)
            .bind(...(options?.tenantId ? [options.tenantId] : []))
            .first<{ count: number }>();

        return (convCount?.count ?? 0) + (respCount?.count ?? 0);
    }

    async saveEvent(event: InteractionEvent): Promise<void> {
        await this.db
            .prepare(`
        INSERT INTO ${D1_TABLES.INTERACTION_EVENTS} (id, interaction_id, type, payload, timestamp)
        VALUES (?, ?, ?, ?, ?)
      `)
            .bind(
                event.id,
                event.interactionId,
                event.type,
                JSON.stringify(event.payload ?? null),
                event.timestamp.toISOString(),
            )
            .run();
    }

    async getEvents(interactionId: string): Promise<InteractionEvent[]> {
        const rows = await this.db
            .prepare(`
        SELECT * FROM ${D1_TABLES.INTERACTION_EVENTS}
        WHERE interaction_id = ?
        ORDER BY timestamp ASC
      `)
            .bind(interactionId)
            .all<EventRow>();

        return rows.results.map((row) => ({
            id: row.id,
            interactionId: row.interaction_id,
            type: row.type as InteractionEvent['type'],
            payload: row.payload ? JSON.parse(row.payload) : undefined,
            timestamp: new Date(row.timestamp),
        }));
    }

    // ---- Shadow Results ----

    async saveShadowResult(result: ShadowResult): Promise<void> {
        await this.db
            .prepare(`
        INSERT INTO ${D1_TABLES.SHADOW_RESULTS} (
          id, interaction_id, provider_name, request, response, error,
          duration_ms, divergences, has_structural_divergence, created_at
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      `)
            .bind(
                result.id,
                result.interactionId,
                result.providerName,
                JSON.stringify(result.request ?? null),
                JSON.stringify(result.response ?? null),
                JSON.stringify(result.error ?? null),
                result.durationMs,
                JSON.stringify(result.divergences),
                result.hasStructuralDivergence ? 1 : 0,
                result.createdAt.toISOString(),
            )
            .run();
    }

    async getShadowResults(interactionId: string): Promise<ShadowResult[]> {
        const rows = await this.db
            .prepare(`
        SELECT * FROM ${D1_TABLES.SHADOW_RESULTS}
        WHERE interaction_id = ?
        ORDER BY created_at ASC
      `)
            .bind(interactionId)
            .all<ShadowRow>();

        return rows.results.map(this.rowToShadowResult);
    }

    async getShadowResult(id: string): Promise<ShadowResult | null> {
        const row = await this.db
            .prepare(`SELECT * FROM ${D1_TABLES.SHADOW_RESULTS} WHERE id = ?`)
            .bind(id)
            .first<ShadowRow>();

        if (!row) return null;
        return this.rowToShadowResult(row);
    }

    async listDivergentShadowResults(
        options?: DivergenceListOptions,
    ): Promise<ShadowResult[]> {
        const limit = options?.limit ?? 50;
        const offset = options?.offset ?? 0;
        const structuralOnly = options?.structuralOnly ?? true;

        const rows = await this.db
            .prepare(`
        SELECT * FROM ${D1_TABLES.SHADOW_RESULTS}
        WHERE has_structural_divergence = ?
        ${options?.tenantId ? 'AND interaction_id IN (SELECT id FROM conversations WHERE tenant_id = ?)' : ''}
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
      `)
            .bind(
                structuralOnly ? 1 : 0,
                ...(options?.tenantId ? [options.tenantId] : []),
                limit,
                offset,
            )
            .all<ShadowRow>();

        return rows.results.map(this.rowToShadowResult);
    }

    // ---- Thread State ----

    async setThreadState(threadKey: string, responseId: string): Promise<void> {
        await this.db
            .prepare(`
        INSERT INTO ${D1_TABLES.THREAD_STATE} (thread_key, response_id, updated_at)
        VALUES (?, ?, ?)
        ON CONFLICT(thread_key) DO UPDATE SET
          response_id = excluded.response_id,
          updated_at = excluded.updated_at
      `)
            .bind(threadKey, responseId, new Date().toISOString())
            .run();
    }

    async getThreadState(threadKey: string): Promise<string | null> {
        const row = await this.db
            .prepare(`SELECT response_id FROM ${D1_TABLES.THREAD_STATE} WHERE thread_key = ?`)
            .bind(threadKey)
            .first<{ response_id: string }>();

        return row?.response_id ?? null;
    }

    // ---- Helpers ----

    private rowToResponse(row: ResponseRow): ResponseRecord {
        return {
            id: row.id,
            tenantId: row.tenant_id,
            appName: row.app_name ?? undefined,
            threadKey: row.thread_key ?? undefined,
            previousResponseId: row.previous_response_id ?? undefined,
            model: row.model,
            status: row.status as ResponseRecord['status'],
            request: row.request ? JSON.parse(row.request) : undefined,
            response: row.response ? JSON.parse(row.response) : undefined,
            error: row.error ? JSON.parse(row.error) : undefined,
            usage: row.usage ? JSON.parse(row.usage) : undefined,
            metadata: JSON.parse(row.metadata || '{}'),
            createdAt: new Date(row.created_at),
            updatedAt: new Date(row.updated_at),
        };
    }

    private rowToShadowResult(row: ShadowRow): ShadowResult {
        return {
            id: row.id,
            interactionId: row.interaction_id,
            providerName: row.provider_name,
            request: row.request ? JSON.parse(row.request) : undefined,
            response: row.response ? JSON.parse(row.response) : undefined,
            error: row.error ? JSON.parse(row.error) : undefined,
            durationMs: row.duration_ms,
            divergences: JSON.parse(row.divergences),
            hasStructuralDivergence: row.has_structural_divergence === 1,
            createdAt: new Date(row.created_at),
        };
    }
}

// ============================================================================
// Internal Row Types
// ============================================================================

interface ConversationRow {
    id: string;
    tenant_id: string;
    app_name: string | null;
    model: string | null;
    metadata: string;
    created_at: string;
    updated_at: string;
}

interface MessageRow {
    id: string;
    conversation_id: string;
    role: string;
    content: string;
    usage: string | null;
    timestamp: string;
}

interface ResponseRow {
    id: string;
    tenant_id: string;
    app_name: string | null;
    thread_key: string | null;
    previous_response_id: string | null;
    model: string;
    status: string;
    request: string | null;
    response: string | null;
    error: string | null;
    usage: string | null;
    metadata: string;
    created_at: string;
    updated_at: string;
}

interface InteractionRow {
    type: string;
    id: string;
    tenant_id: string;
    app_name: string | null;
    model: string | null;
    created_at: string;
    updated_at: string;
}

interface EventRow {
    id: string;
    interaction_id: string;
    type: string;
    payload: string | null;
    timestamp: string;
}

interface ShadowRow {
    id: string;
    interaction_id: string;
    provider_name: string;
    request: string | null;
    response: string | null;
    error: string | null;
    duration_ms: number;
    divergences: string;
    has_structural_divergence: number;
    created_at: string;
}
