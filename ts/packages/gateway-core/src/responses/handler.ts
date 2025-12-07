/**
 * Responses API handler - handles OpenAI Responses API (/v1/responses).
 *
 * @module responses/handler
 */

import type { CanonicalRequest, CanonicalResponse, Message, ToolDefinition } from '../domain/types.js';
import type {
    ResponsesAPIRequest,
    ResponsesAPIResponse,
    ResponsesInputItem,
    ResponsesOutputItem,
    ResponsesOutputContent,
    ResponsesUsage,
    Thread,
    ThreadMessage,
} from '../domain/responses.js';
import { responsesInputToMessages } from '../domain/responses.js';
import type { StorageProvider, ResponseRecord } from '../ports/storage.js';
import type { Provider } from '../ports/provider.js';
import type { Logger } from '../utils/logging.js';
import { errNotFound, errInvalidRequest } from '../domain/errors.js';
import { randomUUID } from '../utils/crypto.js';

// ============================================================================
// Handler Options
// ============================================================================

/**
 * Responses handler options.
 */
export interface ResponsesHandlerOptions {
    /** Storage provider for thread state. */
    storage: StorageProvider;

    /** Provider to use for completions. */
    provider: Provider;

    /** Logger. */
    logger?: Logger | undefined;
}

// ============================================================================
// Responses Handler
// ============================================================================

/**
 * Handles Responses API requests.
 */
export class ResponsesHandler {
    private readonly storage: StorageProvider;
    private readonly provider: Provider;
    private readonly logger?: Logger;

    constructor(options: ResponsesHandlerOptions) {
        this.storage = options.storage;
        this.provider = options.provider;
        this.logger = options.logger;
    }

    /**
     * Handles a Responses API request.
     */
    async handle(
        request: ResponsesAPIRequest,
        tenantId: string,
        appName?: string,
    ): Promise<ResponsesAPIResponse> {
        const responseId = `resp_${randomUUID().replace(/-/g, '')}`;
        const now = new Date();

        // Resolve previous response if provided
        let previousMessages: Message[] = [];
        if (request.previousResponseId) {
            previousMessages = await this.resolvePreviousResponse(request.previousResponseId);
        }

        // Convert request to canonical format
        const canonicalRequest = this.toCanonicalRequest(
            request,
            tenantId,
            previousMessages,
        );

        // Make completion request
        const canonicalResponse = await this.provider.complete(canonicalRequest);

        // Build response items from completion
        const outputItems = this.buildOutputItems(canonicalResponse);

        // Create response record
        const response: ResponsesAPIResponse = {
            id: responseId,
            object: 'response',
            createdAt: Math.floor(now.getTime() / 1000),
            status: 'completed',
            model: canonicalResponse.model,
            output: outputItems,
            usage: {
                inputTokens: canonicalResponse.usage.promptTokens,
                outputTokens: canonicalResponse.usage.completionTokens,
                totalTokens: canonicalResponse.usage.totalTokens,
            },
            metadata: request.metadata,
        };

        // Store for threading
        const record: ResponseRecord = {
            id: responseId,
            tenantId,
            appName,
            previousResponseId: request.previousResponseId,
            model: request.model,
            status: 'completed',
            request: canonicalRequest,
            response: canonicalResponse,
            usage: canonicalResponse.usage,
            metadata: request.metadata ?? {},
            createdAt: now,
            updatedAt: now,
        };

        await this.storage.saveResponse(record);

        return response;
    }

    /**
     * Gets a response by ID.
     */
    async get(responseId: string): Promise<ResponsesAPIResponse | null> {
        const record = await this.storage.getResponse(responseId);
        if (!record) return null;

        return this.recordToResponse(record);
    }

    /**
     * Lists responses for a tenant.
     */
    async list(
        tenantId: string,
        options?: { limit?: number; offset?: number },
    ): Promise<ResponsesAPIResponse[]> {
        const records = await this.storage.listResponses(tenantId, options);
        return records.map((r) => this.recordToResponse(r));
    }

    /**
     * Handles a streaming Responses API request.
     * Yields SSE events in the Responses API format.
     */
    async *handleStream(
        request: ResponsesAPIRequest,
        tenantId: string,
        appName?: string,
    ): AsyncGenerator<string> {
        const responseId = `resp_${randomUUID().replace(/-/g, '')}`;
        const now = new Date();

        // Resolve previous response if provided
        let previousMessages: Message[] = [];
        if (request.previousResponseId) {
            previousMessages = await this.resolvePreviousResponse(request.previousResponseId);
        }

        // Convert request to canonical format with streaming enabled
        const canonicalRequest = this.toCanonicalRequest(
            request,
            tenantId,
            previousMessages,
        );
        canonicalRequest.stream = true;

        // Emit response.created event
        yield this.formatSSE('response.created', {
            type: 'response.created',
            response: {
                id: responseId,
                object: 'response',
                created_at: Math.floor(now.getTime() / 1000),
                status: 'in_progress',
                model: request.model,
                output: [],
            },
        });

        // Emit response.in_progress event
        yield this.formatSSE('response.in_progress', {
            type: 'response.in_progress',
            response: {
                id: responseId,
                status: 'in_progress',
            },
        });

        // Generate output item ID
        const outputItemId = `item_${randomUUID().replace(/-/g, '')}`;
        let outputItemIndex = 0;

        // Emit output_item.added
        yield this.formatSSE('response.output_item.added', {
            type: 'response.output_item.added',
            output_index: 0,
            item: {
                id: outputItemId,
                type: 'message',
                status: 'in_progress',
                role: 'assistant',
                content: [],
            },
        });

        // Stream from provider
        let fullContent = '';
        let usage = { promptTokens: 0, completionTokens: 0, totalTokens: 0 };

        try {
            for await (const event of this.provider.stream(canonicalRequest)) {
                if (event.contentDelta) {
                    fullContent += event.contentDelta;

                    // Emit content delta
                    yield this.formatSSE('response.output_item.delta', {
                        type: 'response.output_item.delta',
                        output_index: 0,
                        item_id: outputItemId,
                        delta: {
                            type: 'text',
                            text: event.contentDelta,
                        },
                    });
                }

                if (event.usage) {
                    usage = event.usage;
                }

                if (event.type === 'done') {
                    break;
                }
            }

            // Emit output_item.done
            yield this.formatSSE('response.output_item.done', {
                type: 'response.output_item.done',
                output_index: 0,
                item: {
                    id: outputItemId,
                    type: 'message',
                    status: 'completed',
                    role: 'assistant',
                    content: [{
                        type: 'output_text',
                        text: fullContent,
                    }],
                },
            });

            // Emit response.completed
            yield this.formatSSE('response.completed', {
                type: 'response.completed',
                response: {
                    id: responseId,
                    object: 'response',
                    created_at: Math.floor(now.getTime() / 1000),
                    status: 'completed',
                    model: request.model,
                    output: [{
                        id: outputItemId,
                        type: 'message',
                        status: 'completed',
                        role: 'assistant',
                        content: [{
                            type: 'output_text',
                            text: fullContent,
                        }],
                    }],
                    usage: {
                        input_tokens: usage.promptTokens,
                        output_tokens: usage.completionTokens,
                        total_tokens: usage.totalTokens,
                    },
                },
            });

            // Emit done marker
            yield this.formatSSE('response.done', {
                type: 'response.done',
            });

            // Store for threading
            const record: ResponseRecord = {
                id: responseId,
                tenantId,
                appName,
                previousResponseId: request.previousResponseId,
                model: request.model,
                status: 'completed',
                request: canonicalRequest,
                response: {
                    id: responseId,
                    model: request.model,
                    content: fullContent,
                },
                usage,
                metadata: request.metadata ?? {},
                createdAt: now,
                updatedAt: new Date(),
            };

            await this.storage.saveResponse(record);

        } catch (error) {
            // Emit error event
            yield this.formatSSE('response.failed', {
                type: 'response.failed',
                response: {
                    id: responseId,
                    status: 'failed',
                    error: {
                        type: 'server_error',
                        message: error instanceof Error ? error.message : 'Unknown error',
                    },
                },
            });
        }
    }

    /**
     * Formats an SSE event.
     */
    private formatSSE(eventType: string, data: unknown): string {
        return `event: ${eventType}\ndata: ${JSON.stringify(data)}\n\n`;
    }


    /**
     * Resolves a previous response to get its messages.
     */
    private async resolvePreviousResponse(responseId: string): Promise<Message[]> {
        const record = await this.storage.getResponse(responseId);
        if (!record) {
            throw errNotFound(`Previous response '${responseId}' not found`);
        }

        const messages: Message[] = [];

        // Get request messages
        const req = record.request as CanonicalRequest | undefined;
        if (req?.messages) {
            messages.push(...req.messages);
        }

        // Get response message
        const resp = record.response as CanonicalResponse | undefined;
        if (resp?.choices?.[0]?.message) {
            messages.push(resp.choices[0].message);
        }

        return messages;
    }

    /**
     * Converts a Responses API request to canonical format.
     */
    private toCanonicalRequest(
        request: ResponsesAPIRequest,
        tenantId: string,
        previousMessages: Message[],
    ): CanonicalRequest {
        // Convert input to messages
        const inputMessages = responsesInputToMessages(request.input);
        const messages: Message[] = [...previousMessages, ...inputMessages];

        // Convert tools
        const tools = request.tools?.map((t): ToolDefinition => ({
            name: t.function.name,
            type: 'function',
            function: {
                name: t.function.name,
                description: t.function.description,
                parameters: t.function.parameters,
            },
        }));

        return {
            tenantId,
            model: request.model,
            messages,
            stream: request.stream ?? false,
            instructions: request.instructions,
            tools,
            maxTokens: request.maxOutputTokens,
            temperature: request.temperature,
            topP: request.topP,
            metadata: request.metadata,
            sourceAPIType: 'responses',
        };
    }

    /**
     * Builds output items from a canonical response.
     */
    private buildOutputItems(response: CanonicalResponse): ResponsesOutputItem[] {
        const items: ResponsesOutputItem[] = [];

        for (const choice of response.choices) {
            const message = choice.message;

            // Add message item
            if (message.content) {
                const content: ResponsesOutputContent[] = [
                    { type: 'output_text', text: message.content },
                ];
                items.push({
                    type: 'message',
                    id: `item_${randomUUID().replace(/-/g, '').slice(0, 12)}`,
                    role: 'assistant',
                    content,
                    status: 'completed',
                });
            }

            // Add function call items
            if (message.toolCalls) {
                for (const tc of message.toolCalls) {
                    items.push({
                        type: 'function_call',
                        id: `item_${randomUUID().replace(/-/g, '').slice(0, 12)}`,
                        callId: tc.id,
                        name: tc.function.name,
                        arguments: tc.function.arguments,
                        status: 'completed',
                    });
                }
            }
        }

        return items;
    }

    /**
     * Converts a stored record to a Responses API response.
     */
    private recordToResponse(record: ResponseRecord): ResponsesAPIResponse {
        const resp = record.response as CanonicalResponse | undefined;
        const output = resp ? this.buildOutputItems(resp) : [];

        const usage: ResponsesUsage | undefined = record.usage
            ? {
                inputTokens: record.usage.promptTokens,
                outputTokens: record.usage.completionTokens,
                totalTokens: record.usage.totalTokens,
            }
            : undefined;

        return {
            id: record.id,
            object: 'response',
            createdAt: Math.floor(record.createdAt.getTime() / 1000),
            status: record.status as 'completed' | 'failed',
            model: record.model,
            output,
            usage,
            metadata: record.metadata,
            error: record.error
                ? { type: 'server_error', message: String(record.error) }
                : undefined,
        };
    }

    // ---- Thread API Methods ----

    /**
     * Creates a new thread.
     */
    async createThread(
        tenantId: string,
        metadata?: Record<string, string>,
    ): Promise<Thread> {
        const threadId = `thread_${randomUUID().replace(/-/g, '')}`;
        const now = new Date();

        const storedThread: import('../ports/storage.js').StoredThread = {
            id: threadId,
            tenantId,
            messages: [],
            metadata,
            createdAt: now,
            updatedAt: now,
        };

        if (this.storage.createThread) {
            await this.storage.createThread(storedThread);
        }

        return {
            id: threadId,
            object: 'thread',
            createdAt: Math.floor(now.getTime() / 1000),
            metadata,
        };
    }

    /**
     * Gets a thread by ID.
     */
    async getThread(
        threadId: string,
        tenantId: string,
    ): Promise<Thread | null> {
        if (!this.storage.getThread) {
            return null;
        }

        const thread = await this.storage.getThread(threadId);
        if (!thread || thread.tenantId !== tenantId) {
            return null;
        }

        return {
            id: thread.id,
            object: 'thread',
            createdAt: Math.floor(thread.createdAt.getTime() / 1000),
            metadata: thread.metadata,
        };
    }

    /**
     * Creates a message in a thread.
     */
    async createMessage(
        threadId: string,
        tenantId: string,
        role: 'user' | 'assistant',
        content: string,
    ): Promise<ThreadMessage | null> {
        if (!this.storage.getThread || !this.storage.addMessage) {
            return null;
        }

        const thread = await this.storage.getThread(threadId);
        if (!thread || thread.tenantId !== tenantId) {
            return null;
        }

        const messageId = `msg_${randomUUID().replace(/-/g, '')}`;
        const now = new Date();

        const storedMessage: import('../ports/storage.js').StoredMessage = {
            id: messageId,
            role,
            content,
            timestamp: now,
        };

        await this.storage.addMessage(threadId, storedMessage);

        return {
            id: messageId,
            object: 'thread.message',
            threadId,
            createdAt: Math.floor(now.getTime() / 1000),
            role,
            content: [{ type: 'text', text: { value: content } }],
        };
    }

    /**
     * Lists messages in a thread.
     */
    async listMessages(
        threadId: string,
        tenantId: string,
    ): Promise<ThreadMessage[]> {
        if (!this.storage.getThread || !this.storage.listMessages) {
            return [];
        }

        const thread = await this.storage.getThread(threadId);
        if (!thread || thread.tenantId !== tenantId) {
            return [];
        }

        const messages = await this.storage.listMessages(threadId);
        return messages.map((msg): ThreadMessage => ({
            id: msg.id,
            object: 'thread.message',
            threadId,
            createdAt: Math.floor(msg.timestamp.getTime() / 1000),
            role: msg.role as 'user' | 'assistant',
            content: [{ type: 'text', text: { value: msg.content } }],
        }));
    }

    /**
     * Creates a run (executes the thread through the provider).
     */
    async createRun(
        threadId: string,
        tenantId: string,
        options: { model?: string; instructions?: string } = {},
    ): Promise<ResponsesAPIResponse | null> {
        if (!this.storage.getThread || !this.storage.listMessages) {
            return null;
        }

        const thread = await this.storage.getThread(threadId);
        if (!thread || thread.tenantId !== tenantId) {
            return null;
        }

        const messages = await this.storage.listMessages(threadId);
        const messageContents: Message[] = messages.map((m) => ({
            role: m.role as 'user' | 'assistant',
            content: m.content,
        }));

        // Build a Responses API request from thread messages
        const request: ResponsesAPIRequest = {
            model: options.model ?? 'gpt-4',
            input: messageContents.map((m) => ({
                type: 'message' as const,
                role: m.role as 'user' | 'assistant',
                content: m.content,
            })),
            instructions: options.instructions,
        };

        // Execute as a normal response
        const response = await this.handle(request, tenantId);

        // Add assistant message to thread if successful
        if (response.status === 'completed' && this.storage.addMessage) {
            const content = response.output
                .filter((o) => o.type === 'message')
                .flatMap((o) => o.content ?? [])
                .filter((c) => c.type === 'output_text')
                .map((c) => c.text ?? '')
                .join('');

            if (content) {
                await this.storage.addMessage(threadId, {
                    id: `msg_${randomUUID().replace(/-/g, '')}`,
                    role: 'assistant',
                    content,
                    timestamp: new Date(),
                });
            }
        }

        return response;
    }

    /**
     * Cancels a response.
     */
    async cancel(
        responseId: string,
        tenantId: string,
    ): Promise<ResponsesAPIResponse | null> {
        const record = await this.storage.getResponse(responseId);
        if (!record || record.tenantId !== tenantId) {
            return null;
        }

        if (record.status !== 'in_progress') {
            throw errInvalidRequest('Response is not in progress');
        }

        record.status = 'cancelled';
        record.updatedAt = new Date();

        if (this.storage.updateResponse) {
            await this.storage.updateResponse(responseId, {
                status: 'cancelled',
                updatedAt: record.updatedAt,
            });
        } else {
            await this.storage.saveResponse(record);
        }

        return this.recordToResponse(record);
    }
}
