/**
 * Storage provider port.
 *
 * @module ports/storage
 */

import type { Message, Usage } from '../domain/types.js';
import type { ShadowResult } from '../domain/shadow.js';
import type { InteractionEvent } from '../domain/events.js';

// ============================================================================
// Conversation Types
// ============================================================================

/**
 * A stored conversation (for Chat Completions API).
 */
export interface Conversation {
    /** Conversation ID. */
    id: string;

    /** Tenant ID. */
    tenantId: string;

    /** App name. */
    appName?: string | undefined;

    /** Model used. */
    model?: string | undefined;

    /** Conversation messages. */
    messages: StoredMessage[];

    /** Metadata. */
    metadata?: Record<string, string> | undefined;

    /** Creation timestamp. */
    createdAt: Date;

    /** Last update timestamp. */
    updatedAt: Date;
}

/**
 * A stored message in a conversation.
 */
export interface StoredMessage {
    /** Message ID. */
    id: string;

    /** Message role. */
    role: 'system' | 'user' | 'assistant' | 'tool';

    /** Message content. */
    content: string;

    /** Token usage (for assistant messages). */
    usage?: Usage | undefined;

    /** Timestamp. */
    timestamp: Date;
}

// ============================================================================
// Response Types (for Responses API)
// ============================================================================

/**
 * A stored response record (for Responses API).
 */
export interface ResponseRecord {
    /** Response ID. */
    id: string;

    /** Tenant ID. */
    tenantId: string;

    /** App name. */
    appName?: string | undefined;

    /** Thread key for linking responses. */
    threadKey?: string | undefined;

    /** Previous response ID. */
    previousResponseId?: string | undefined;

    /** Model used. */
    model: string;

    /** Response status. */
    status: 'in_progress' | 'completed' | 'incomplete' | 'cancelled' | 'failed';

    /** Request data. */
    request?: unknown;

    /** Response data. */
    response?: unknown;

    /** Error if failed. */
    error?: unknown;

    /** Token usage. */
    usage?: Usage | undefined;

    /** Metadata. */
    metadata?: Record<string, string> | undefined;

    /** Creation timestamp. */
    createdAt: Date;

    /** Last update timestamp. */
    updatedAt: Date;
}

// ============================================================================
// Interaction Types (Unified)
// ============================================================================

/** Type of interaction. */
export type InteractionType = 'conversation' | 'response';

/**
 * Unified interaction summary for listing.
 */
export interface InteractionSummary {
    /** Interaction ID. */
    id: string;

    /** Interaction type. */
    type: InteractionType;

    /** Tenant ID. */
    tenantId: string;

    /** App name. */
    appName?: string | undefined;

    /** Model used. */
    model?: string | undefined;

    /** Message count (for conversations). */
    messageCount?: number | undefined;

    /** Status (for responses). */
    status?: string | undefined;

    /** Creation timestamp. */
    createdAt: Date;

    /** Last update timestamp. */
    updatedAt: Date;
}

// ============================================================================
// List Options
// ============================================================================

/**
 * Options for listing items.
 */
export interface ListOptions {
    /** Maximum number of items to return. */
    limit?: number | undefined;

    /** Offset for pagination. */
    offset?: number | undefined;

    /** Order by field. */
    orderBy?: string | undefined;

    /** Sort direction. */
    order?: 'asc' | 'desc' | undefined;
}

/**
 * Options for listing interactions.
 */
export interface InteractionListOptions extends ListOptions {
    /** Filter by type. */
    type?: InteractionType | undefined;

    /** Filter by tenant. */
    tenantId?: string | undefined;
}

/**
 * Options for listing divergent shadow results.
 */
export interface DivergenceListOptions extends ListOptions {
    /** Filter by tenant. */
    tenantId?: string | undefined;

    /** Only include structural divergences. */
    structuralOnly?: boolean | undefined;
}

// ============================================================================
// Conversation Store Interface
// ============================================================================

/**
 * Storage for conversations.
 */
export interface ConversationStore {
    /**
     * Saves a conversation.
     */
    saveConversation(conversation: Conversation): Promise<void>;

    /**
     * Gets a conversation by ID.
     */
    getConversation(id: string): Promise<Conversation | null>;

    /**
     * Lists conversations for a tenant.
     */
    listConversations(
        tenantId: string,
        options?: ListOptions,
    ): Promise<Conversation[]>;

    /**
     * Deletes a conversation.
     */
    deleteConversation?(id: string): Promise<void>;
}

// ============================================================================
// Response Store Interface
// ============================================================================

/**
 * Storage for responses.
 */
export interface ResponseStore {
    /**
     * Saves a response record.
     */
    saveResponse(response: ResponseRecord): Promise<void>;

    /**
     * Gets a response by ID.
     */
    getResponse(id: string): Promise<ResponseRecord | null>;

    /**
     * Lists responses for a tenant.
     */
    listResponses(
        tenantId: string,
        options?: ListOptions,
    ): Promise<ResponseRecord[]>;

    /**
     * Updates a response.
     */
    updateResponse?(
        id: string,
        updates: Partial<ResponseRecord>,
    ): Promise<void>;

    /**
     * Deletes a response.
     */
    deleteResponse?(id: string): Promise<void>;
}

// ============================================================================
// Interaction Store Interface
// ============================================================================

/**
 * Unified storage for interactions (conversations + responses).
 */
export interface InteractionStore {
    /**
     * Lists all interactions.
     */
    listInteractions(
        options?: InteractionListOptions,
    ): Promise<InteractionSummary[]>;

    /**
     * Gets total interaction count.
     */
    getInteractionCount(options?: InteractionListOptions): Promise<number>;

    /**
     * Saves an interaction event.
     */
    saveEvent(event: InteractionEvent): Promise<void>;

    /**
     * Gets events for an interaction.
     */
    getEvents(interactionId: string): Promise<InteractionEvent[]>;
}

// ============================================================================
// Shadow Store Interface
// ============================================================================

/**
 * Storage for shadow results.
 */
export interface ShadowStore {
    /**
     * Saves a shadow result.
     */
    saveShadowResult(result: ShadowResult): Promise<void>;

    /**
     * Gets shadow results for an interaction.
     */
    getShadowResults(interactionId: string): Promise<ShadowResult[]>;

    /**
     * Gets a shadow result by ID.
     */
    getShadowResult(id: string): Promise<ShadowResult | null>;

    /**
     * Lists divergent shadow results.
     */
    listDivergentShadowResults(
        options?: DivergenceListOptions,
    ): Promise<ShadowResult[]>;
}

// ============================================================================
// Thread State Store Interface
// ============================================================================

/**
 * Storage for thread state (Responses API continuation).
 */
export interface ThreadStateStore {
    /**
     * Sets the thread state (links thread key to response ID).
     */
    setThreadState(threadKey: string, responseId: string): Promise<void>;

    /**
     * Gets the thread state (latest response ID for thread).
     */
    getThreadState(threadKey: string): Promise<string | null>;
}

// ============================================================================
// Thread Store Interface (OpenAI Assistants-style)
// ============================================================================

/**
 * A stored thread.
 */
export interface StoredThread {
    /** Thread ID. */
    id: string;

    /** Tenant ID. */
    tenantId: string;

    /** Messages in the thread. */
    messages: StoredMessage[];

    /** Metadata. */
    metadata?: Record<string, string> | undefined;

    /** Creation timestamp. */
    createdAt: Date;

    /** Last update timestamp. */
    updatedAt: Date;
}

/**
 * Storage for threads (OpenAI Assistants-style API).
 */
export interface ThreadStore {
    /**
     * Creates a new thread.
     */
    createThread(thread: StoredThread): Promise<void>;

    /**
     * Gets a thread by ID.
     */
    getThread(id: string): Promise<StoredThread | null>;

    /**
     * Adds a message to a thread.
     */
    addMessage(threadId: string, message: StoredMessage): Promise<void>;

    /**
     * Lists messages in a thread.
     */
    listMessages(threadId: string, options?: ListOptions): Promise<StoredMessage[]>;

    /**
     * Deletes a thread.
     */
    deleteThread?(id: string): Promise<void>;
}

// ============================================================================
// Combined Storage Provider Interface
// ============================================================================

/**
 * Full storage provider combining all storage interfaces.
 */
export interface StorageProvider
    extends ConversationStore,
    ResponseStore,
    InteractionStore,
    ShadowStore,
    ThreadStateStore,
    Partial<ThreadStore> {
    /**
     * Closes the storage connection.
     */
    close?(): Promise<void>;
}
