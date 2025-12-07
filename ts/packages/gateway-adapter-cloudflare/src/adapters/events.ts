/**
 * Queue-based event publisher for Cloudflare Workers.
 *
 * @module adapters/events
 */

import type { EventPublisher, LifecycleEvent } from '@polyglot-llm-gateway/gateway-core';

// ============================================================================
// Queue Event Publisher
// ============================================================================

/**
 * Event publisher backed by Cloudflare Queues.
 */
export class QueueEventPublisher implements EventPublisher {
    constructor(
        private readonly queue: Queue,
        private readonly ctx?: ExecutionContext,
    ) { }

    async publish(event: LifecycleEvent): Promise<void> {
        // Use waitUntil if available to not block response
        const sendPromise = this.queue.send({
            id: event.id,
            type: event.type,
            interactionId: event.interactionId,
            tenantId: event.tenantId,
            timestamp: event.timestamp.toISOString(),
            data: event.data,
        });

        if (this.ctx) {
            this.ctx.waitUntil(sendPromise);
        } else {
            await sendPromise;
        }
    }
}

// ============================================================================
// Null Event Publisher
// ============================================================================

/**
 * No-op event publisher.
 */
export class NullEventPublisher implements EventPublisher {
    async publish(_event: LifecycleEvent): Promise<void> {
        // No-op
    }
}

// ============================================================================
// Batch Event Publisher
// ============================================================================

/**
 * Event publisher that batches events before sending.
 */
export class BatchEventPublisher implements EventPublisher {
    private readonly events: LifecycleEvent[] = [];
    private flushTimeout?: ReturnType<typeof setTimeout>;

    constructor(
        private readonly inner: EventPublisher,
        private readonly options?: {
            batchSize?: number;
            flushIntervalMs?: number;
        },
    ) { }

    async publish(event: LifecycleEvent): Promise<void> {
        this.events.push(event);

        const batchSize = this.options?.batchSize ?? 10;
        if (this.events.length >= batchSize) {
            await this.flush();
            return;
        }

        // Schedule flush if not already scheduled
        if (!this.flushTimeout) {
            const flushInterval = this.options?.flushIntervalMs ?? 1000;
            this.flushTimeout = setTimeout(() => this.flush(), flushInterval);
        }
    }

    async flush(): Promise<void> {
        if (this.flushTimeout) {
            clearTimeout(this.flushTimeout);
            this.flushTimeout = undefined;
        }

        if (this.events.length === 0) return;

        const batch = this.events.splice(0, this.events.length);
        await Promise.all(batch.map((e) => this.inner.publish(e)));
    }

    async close(): Promise<void> {
        await this.flush();
    }
}
