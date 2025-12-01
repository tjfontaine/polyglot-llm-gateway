import { useQuery } from 'urql';
import { useCallback, useMemo } from 'react';
import {
    StatsQuery,
    OverviewQuery,
    InteractionsQuery,
    InteractionQuery,
    InteractionEventsQuery,
    ShadowResultsQuery,
    DivergentShadowsQuery,
    ShadowQuery,
} from './operations';
import type {
    Stats,
    Overview,
    InteractionSummary,
    Interaction,
    InteractionEventsResponse,
    ShadowResultsResponse,
    DivergentShadowsResponse,
    ShadowResult,
    InteractionFilter,
} from './graphql';

// Re-export types for convenience
export type {
    Stats,
    Overview,
    InteractionSummary,
    Interaction,
    InteractionEventsResponse,
    ShadowResultsResponse,
    DivergentShadowsResponse,
    ShadowResult,
    InteractionFilter,
};

// Stats hook
export function useStats() {
    const [result, reexecute] = useQuery({ query: StatsQuery });

    return {
        stats: result.data?.stats as Stats | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Overview hook
export function useOverview() {
    const [result, reexecute] = useQuery({ query: OverviewQuery });

    return {
        overview: result.data?.overview as Overview | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Interactions list hook
export function useInteractions(options?: {
    filter?: InteractionFilter;
    limit?: number;
    offset?: number;
}) {
    const variables = useMemo(() => ({
        filter: options?.filter,
        limit: options?.limit ?? 50,
        offset: options?.offset ?? 0,
    }), [options?.filter, options?.limit, options?.offset]);

    const [result, reexecute] = useQuery({
        query: InteractionsQuery,
        variables,
    });

    return {
        interactions: (result.data?.interactions?.interactions ?? []) as InteractionSummary[],
        total: result.data?.interactions?.total ?? 0,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Single interaction hook
export function useInteraction(id: string | null) {
    const [result, reexecute] = useQuery({
        query: InteractionQuery,
        variables: { id: id ?? '' },
        pause: !id,
    });

    return {
        interaction: result.data?.interaction as Interaction | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Interaction events hook
export function useInteractionEvents(interactionId: string | null, limit?: number) {
    const [result, reexecute] = useQuery({
        query: InteractionEventsQuery,
        variables: { interactionId: interactionId ?? '', limit },
        pause: !interactionId,
    });

    return {
        events: result.data?.interactionEvents as InteractionEventsResponse | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Shadow results hook
export function useShadowResults(interactionId: string | null) {
    const [result, reexecute] = useQuery({
        query: ShadowResultsQuery,
        variables: { interactionId: interactionId ?? '' },
        pause: !interactionId,
    });

    return {
        shadowResults: result.data?.shadowResults as ShadowResultsResponse | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Divergent shadows hook
export function useDivergentShadows(options?: {
    limit?: number;
    offset?: number;
    provider?: string;
}) {
    const variables = useMemo(() => ({
        limit: options?.limit ?? 100,
        offset: options?.offset ?? 0,
        provider: options?.provider,
    }), [options?.limit, options?.offset, options?.provider]);

    const [result, reexecute] = useQuery({
        query: DivergentShadowsQuery,
        variables,
    });

    return {
        divergentShadows: result.data?.divergentShadows as DivergentShadowsResponse | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Single shadow hook
export function useShadow(id: string | null) {
    const [result, reexecute] = useQuery({
        query: ShadowQuery,
        variables: { id: id ?? '' },
        pause: !id,
    });

    return {
        shadow: result.data?.shadow as ShadowResult | null,
        loading: result.fetching,
        error: result.error?.message ?? null,
        refresh: useCallback(() => reexecute({ requestPolicy: 'network-only' }), [reexecute]),
    };
}

// Helper functions for formatting (moved from old useApi)
export const formatBytesToMB = (bytes?: number) => {
    if (!bytes) return '0 MB';
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
};

export const formatTimestamp = (epoch?: number) => {
    if (!epoch) return '--';
    return new Date(epoch * 1000).toLocaleString();
};

export const formatShortDate = (epoch?: number) => {
    if (!epoch) return '--';
    return new Date(epoch * 1000).toLocaleDateString(undefined, {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    });
};

export const friendlyDuration = (raw?: string) => {
    if (!raw) return '--';
    const withoutFraction = raw.split('.')[0];
    return withoutFraction.replace('h', 'h ').replace('m', 'm ').replace('s', 's');
};
