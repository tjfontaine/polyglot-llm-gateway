import { useEffect, useState, useCallback, createContext, useContext, type ReactNode } from 'react';
import type { Stats, Overview, ThreadSummary, ThreadDetail, ResponseSummary, ResponseDetail } from '../types';

const API_BASE = '/admin/api';

interface ApiContextValue {
  stats: Stats | null;
  overview: Overview | null;
  threads: ThreadSummary[];
  responses: ResponseSummary[];
  threadsError: string | null;
  responsesError: string | null;
  loadingThreads: boolean;
  loadingResponses: boolean;
  refreshStats: () => Promise<void>;
  refreshOverview: () => Promise<void>;
  refreshThreads: () => Promise<void>;
  refreshResponses: () => Promise<void>;
  fetchThreadDetail: (id: string) => Promise<ThreadDetail | null>;
  fetchResponseDetail: (id: string) => Promise<ResponseDetail | null>;
}

const ApiContext = createContext<ApiContextValue | null>(null);

export function ApiProvider({ children }: { children: ReactNode }) {
  const [stats, setStats] = useState<Stats | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [responses, setResponses] = useState<ResponseSummary[]>([]);
  const [threadsError, setThreadsError] = useState<string | null>(null);
  const [responsesError, setResponsesError] = useState<string | null>(null);
  const [loadingThreads, setLoadingThreads] = useState(false);
  const [loadingResponses, setLoadingResponses] = useState(false);

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

  const refreshThreads = useCallback(async () => {
    if (!overview?.storage.enabled) {
      setThreads([]);
      setThreadsError('Conversation storage is disabled.');
      return;
    }
    setLoadingThreads(true);
    setThreadsError(null);
    try {
      const res = await fetch(`${API_BASE}/threads?limit=100`);
      if (!res.ok) throw new Error('Failed to load threads');
      const data = await res.json();
      setThreads(data.threads ?? []);
    } catch (err) {
      console.error(err);
      setThreadsError('Unable to load threads right now.');
    } finally {
      setLoadingThreads(false);
    }
  }, [overview?.storage.enabled]);

  const refreshResponses = useCallback(async () => {
    if (!overview?.storage.enabled) {
      setResponses([]);
      setResponsesError('Conversation storage is disabled.');
      return;
    }
    setLoadingResponses(true);
    setResponsesError(null);
    try {
      const res = await fetch(`${API_BASE}/responses?limit=100`);
      if (!res.ok) throw new Error('Failed to load responses');
      const data = await res.json();
      setResponses(data.responses ?? []);
    } catch (err) {
      console.error(err);
      setResponsesError('Unable to load responses right now.');
    } finally {
      setLoadingResponses(false);
    }
  }, [overview?.storage.enabled]);

  const fetchThreadDetail = useCallback(async (id: string): Promise<ThreadDetail | null> => {
    try {
      const res = await fetch(`${API_BASE}/threads/${id}`);
      if (!res.ok) throw new Error('Failed to load thread');
      return await res.json();
    } catch (err) {
      console.error(err);
      return null;
    }
  }, []);

  const fetchResponseDetail = useCallback(async (id: string): Promise<ResponseDetail | null> => {
    try {
      const res = await fetch(`${API_BASE}/responses/${id}`);
      if (!res.ok) throw new Error('Failed to load response');
      return await res.json();
    } catch (err) {
      console.error(err);
      return null;
    }
  }, []);

  // Initial data fetch
  useEffect(() => {
    refreshOverview();
    refreshStats();
    const interval = setInterval(refreshStats, 8000);
    return () => clearInterval(interval);
  }, [refreshOverview, refreshStats]);

  // Fetch threads and responses when overview loads
  useEffect(() => {
    if (overview) {
      refreshThreads();
      refreshResponses();
    }
  }, [overview, refreshThreads, refreshResponses]);

  const value: ApiContextValue = {
    stats,
    overview,
    threads,
    responses,
    threadsError,
    responsesError,
    loadingThreads,
    loadingResponses,
    refreshStats,
    refreshOverview,
    refreshThreads,
    refreshResponses,
    fetchThreadDetail,
    fetchResponseDetail,
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
