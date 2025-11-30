import { useEffect, useState, useCallback, createContext, useContext, type ReactNode } from 'react';
import type { Stats, Overview, InteractionSummary, InteractionDetailUnion } from '../types';

const API_BASE = '/admin/api';

interface ApiContextValue {
  stats: Stats | null;
  overview: Overview | null;
  interactions: InteractionSummary[];
  interactionsTotal: number;
  interactionsError: string | null;
  loadingInteractions: boolean;
  refreshStats: () => Promise<void>;
  refreshOverview: () => Promise<void>;
  refreshInteractions: (filter?: 'conversation' | 'response' | 'interaction' | '') => Promise<void>;
  fetchInteractionDetail: (id: string) => Promise<InteractionDetailUnion | null>;
  fetchInteractionEvents: (id: string) => Promise<any>;
}

const ApiContext = createContext<ApiContextValue | null>(null);

export function ApiProvider({ children }: { children: ReactNode }) {
  const [stats, setStats] = useState<Stats | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [interactions, setInteractions] = useState<InteractionSummary[]>([]);
  const [interactionsTotal, setInteractionsTotal] = useState(0);
  const [interactionsError, setInteractionsError] = useState<string | null>(null);
  const [loadingInteractions, setLoadingInteractions] = useState(false);

  const refreshStats = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/stats`);
      if (!res.ok) throw new Error('Failed to load stats');
      const data = await res.json();
      setStats(data);
    } catch (err) {
      console.error(err);
    }
  }, []);

  const refreshOverview = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/overview`);
      if (!res.ok) throw new Error('Failed to load overview');
      const data = await res.json();
      setOverview(data);
    } catch (err) {
      console.error(err);
    }
  }, []);

  const refreshInteractions = useCallback(async (filter: 'conversation' | 'response' | 'interaction' | '' = '') => {
    if (!overview?.storage.enabled) {
      setInteractions([]);
      setInteractionsTotal(0);
      setInteractionsError('Storage is disabled.');
      return;
    }
    setLoadingInteractions(true);
    setInteractionsError(null);
    try {
      const typeParam = filter ? `&type=${filter}` : '';
      const res = await fetch(`${API_BASE}/interactions?limit=100${typeParam}`);
      if (!res.ok) throw new Error('Failed to load interactions');
      const data = await res.json();
      setInteractions(data.interactions ?? []);
      setInteractionsTotal(data.total ?? 0);
    } catch (err) {
      console.error(err);
      setInteractionsError('Unable to load interactions right now.');
    } finally {
      setLoadingInteractions(false);
    }
  }, [overview?.storage.enabled]);

  const fetchInteractionDetail = useCallback(async (id: string): Promise<InteractionDetailUnion | null> => {
    try {
      const res = await fetch(`${API_BASE}/interactions/${id}`);
      if (!res.ok) throw new Error('Failed to load interaction');
      return await res.json();
    } catch (err) {
      console.error(err);
      return null;
    }
  }, []);

  const fetchInteractionEvents = useCallback(async (id: string) => {
    try {
      const res = await fetch(`${API_BASE}/interactions/${id}/events`);
      if (!res.ok) throw new Error('Failed to load interaction events');
      return await res.json();
    } catch (err) {
      console.error(err);
      return { interaction_id: id, events: [] };
    }
  }, []);

  // Initial data fetch
  useEffect(() => {
    refreshOverview();
    refreshStats();
    const interval = setInterval(refreshStats, 8000);
    return () => clearInterval(interval);
  }, [refreshOverview, refreshStats]);

  // Fetch interactions when overview loads
  useEffect(() => {
    if (overview) {
      refreshInteractions();
    }
  }, [overview, refreshInteractions]);

  const value: ApiContextValue = {
    stats,
    overview,
    interactions,
    interactionsTotal,
    interactionsError,
    loadingInteractions,
    refreshStats,
    refreshOverview,
    refreshInteractions,
    fetchInteractionDetail,
    fetchInteractionEvents,
  };

  return <ApiContext.Provider value={value}>{children}</ApiContext.Provider>;
}

export function useApi() {
  const context = useContext(ApiContext);
  if (!context) {
    throw new Error('useApi must be used within an ApiProvider');
  }
  return context;
}

// Helper functions
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
