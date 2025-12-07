/**
 * Event publisher port.
 *
 * @module ports/events
 */

import type { LifecycleEvent } from '../domain/events.js';

// ============================================================================
// EventPublisher Interface
// ============================================================================

/**
 * Publishes lifecycle events.
 * Implementations: Queue-based (CF), Redis (Node), direct storage, etc.
 */
export interface EventPublisher {
    /**
     * Publishes a lifecycle event.
     */
    publish(event: LifecycleEvent): Promise<void>;

    /**
     * Closes the publisher.
     */
    close?(): Promise<void>;
}

// ============================================================================
// Null Implementation
// ============================================================================

/**
 * No-op event publisher for when events are not needed.
 */
export class NullEventPublisher implements EventPublisher {
    async publish(_event: LifecycleEvent): Promise<void> {
        // No-op
    }
}
