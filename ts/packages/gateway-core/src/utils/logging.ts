/**
 * Structured logging utilities.
 *
 * @module utils/logging
 */

// ============================================================================
// Log Levels
// ============================================================================

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const LOG_LEVELS: Record<LogLevel, number> = {
    debug: 0,
    info: 1,
    warn: 2,
    error: 3,
};

// ============================================================================
// Logger Interface
// ============================================================================

/**
 * Structured logger interface.
 */
export interface Logger {
    debug(message: string, fields?: Record<string, unknown>): void;
    info(message: string, fields?: Record<string, unknown>): void;
    warn(message: string, fields?: Record<string, unknown>): void;
    error(message: string, fields?: Record<string, unknown>): void;
    child(fields: Record<string, unknown>): Logger;
}

// ============================================================================
// Console Logger
// ============================================================================

/**
 * Console-based structured logger.
 */
export class ConsoleLogger implements Logger {
    private readonly level: LogLevel;
    private readonly fields: Record<string, unknown>;

    constructor(options?: { level?: LogLevel; fields?: Record<string, unknown> }) {
        this.level = options?.level ?? 'info';
        this.fields = options?.fields ?? {};
    }

    debug(message: string, fields?: Record<string, unknown>): void {
        this.log('debug', message, fields);
    }

    info(message: string, fields?: Record<string, unknown>): void {
        this.log('info', message, fields);
    }

    warn(message: string, fields?: Record<string, unknown>): void {
        this.log('warn', message, fields);
    }

    error(message: string, fields?: Record<string, unknown>): void {
        this.log('error', message, fields);
    }

    child(fields: Record<string, unknown>): Logger {
        return new ConsoleLogger({
            level: this.level,
            fields: { ...this.fields, ...fields },
        });
    }

    private log(level: LogLevel, message: string, fields?: Record<string, unknown>): void {
        if (LOG_LEVELS[level] < LOG_LEVELS[this.level]) return;

        const entry = {
            level,
            message,
            timestamp: new Date().toISOString(),
            ...this.fields,
            ...fields,
        };

        const json = JSON.stringify(entry);

        switch (level) {
            case 'debug':
                console.debug(json);
                break;
            case 'info':
                console.info(json);
                break;
            case 'warn':
                console.warn(json);
                break;
            case 'error':
                console.error(json);
                break;
        }
    }
}

// ============================================================================
// Null Logger
// ============================================================================

/**
 * No-op logger for testing or when logging is disabled.
 */
export class NullLogger implements Logger {
    debug(_message: string, _fields?: Record<string, unknown>): void { }
    info(_message: string, _fields?: Record<string, unknown>): void { }
    warn(_message: string, _fields?: Record<string, unknown>): void { }
    error(_message: string, _fields?: Record<string, unknown>): void { }
    child(_fields: Record<string, unknown>): Logger {
        return this;
    }
}

// ============================================================================
// Default Logger
// ============================================================================

/**
 * Default logger instance.
 */
export const defaultLogger: Logger = new ConsoleLogger();

// ============================================================================
// Request Context Logging
// ============================================================================

/**
 * Creates a logger with request context.
 */
export function requestLogger(
    logger: Logger,
    requestId: string,
    tenantId?: string,
): Logger {
    return logger.child({
        requestId,
        tenantId,
    });
}
