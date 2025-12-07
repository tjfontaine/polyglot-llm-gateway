/**
 * Shadow mode manager - coordinates shadow execution for requests.
 *
 * @module shadow/manager
 */

import type { CanonicalRequest, CanonicalResponse } from '../domain/types.js';
import type { ShadowConfig, ShadowResult } from '../domain/shadow.js';
import type { AppConfig } from '../ports/config.js';
import { ShadowExecutor, type ShadowExecutorOptions } from './executor.js';

// ============================================================================
// Shadow Manager Options
// ============================================================================

/**
 * Options for the shadow manager.
 */
export interface ShadowManagerOptions extends ShadowExecutorOptions {
    /** Default shadow configuration. */
    defaultConfig?: ShadowConfig | undefined;

    /** Sampling rate (0.0-1.0) for shadow execution. */
    samplingRate?: number | undefined;
}

// ============================================================================
// Shadow Manager
// ============================================================================

/**
 * Manages shadow mode execution for the gateway.
 */
export class ShadowManager {
    private readonly executor: ShadowExecutor;
    private readonly defaultConfig?: ShadowConfig;
    private readonly samplingRate: number;

    constructor(options: ShadowManagerOptions) {
        this.executor = new ShadowExecutor(options);
        this.defaultConfig = options.defaultConfig;
        this.samplingRate = options.samplingRate ?? 1.0;
    }

    /**
     * Determines if shadow mode should be executed for a request.
     */
    shouldExecute(
        request: CanonicalRequest,
        app?: AppConfig,
    ): boolean {
        // Check app-level shadow config
        const config = app?.shadow ?? this.defaultConfig;
        if (!config?.enabled) {
            return false;
        }

        // Apply sampling
        return this.matchesSamplingRate();
    }

    /**
     * Gets the shadow configuration to use.
     */
    getConfig(app?: AppConfig): ShadowConfig | undefined {
        return app?.shadow ?? this.defaultConfig;
    }

    /**
     * Executes shadow requests asynchronously.
     * Returns immediately and runs shadows in the background.
     */
    executeAsync(
        interactionId: string,
        request: CanonicalRequest,
        primaryResponse: CanonicalResponse,
        app?: AppConfig,
        onComplete?: (results: ShadowResult[]) => void,
    ): void {
        const config = this.getConfig(app);
        if (!config?.enabled) {
            onComplete?.([]);
            return;
        }

        // Execute in background
        this.executor
            .executeAll(interactionId, request, primaryResponse, config)
            .then((results) => {
                onComplete?.(results);
            })
            .catch((error) => {
                // Log but don't throw - shadow failures shouldn't affect primary
                console.error('Shadow execution failed:', error);
                onComplete?.([]);
            });
    }

    /**
     * Executes shadow requests synchronously.
     * Waits for all shadow executions to complete.
     */
    async executeSync(
        interactionId: string,
        request: CanonicalRequest,
        primaryResponse: CanonicalResponse,
        app?: AppConfig,
    ): Promise<ShadowResult[]> {
        const config = this.getConfig(app);
        if (!config?.enabled) {
            return [];
        }

        return this.executor.executeAll(interactionId, request, primaryResponse, config);
    }

    /**
     * Checks if request matches the sampling rate.
     */
    private matchesSamplingRate(): boolean {
        if (this.samplingRate >= 1) {
            return true;
        }
        if (this.samplingRate <= 0) {
            return false;
        }
        return Math.random() < this.samplingRate;
    }
}
