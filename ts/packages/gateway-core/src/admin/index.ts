/**
 * Admin module exports.
 *
 * @module admin
 */

export {
    // Handler
    AdminHandler,
    type AdminHandlerOptions,

    // Response types
    type StatsResponse,
    type MemoryStats,
    type OverviewResponse,
    type StorageSummary,
    type AppSummary,
    type ProviderSummary,
    type FrontdoorSummary,
    type AdminRoutingSummary,
    type AdminRoutingRule,
    type AdminInteractionSummary,
    type AdminInteractionsListResponse,
} from './handler.js';
