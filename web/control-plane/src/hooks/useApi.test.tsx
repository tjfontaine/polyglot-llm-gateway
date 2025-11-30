import { describe, it, expect, vi, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { useApi, ApiProvider, formatBytesToMB, formatTimestamp, formatShortDate, friendlyDuration } from './useApi';
import { mockStats, mockOverview, mockInteractions, createMockFetch } from '../test/mocks';

describe('useApi hook', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('fetches stats and overview on mount', async () => {
    const mockFetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: mockInteractions, total: mockInteractions.length },
    }));
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.stats).not.toBeNull();
    });

    expect(result.current.stats).toEqual(mockStats);
    expect(result.current.overview).toEqual(mockOverview);
    expect(mockFetch).toHaveBeenCalledWith('/admin/api/stats');
    expect(mockFetch).toHaveBeenCalledWith('/admin/api/overview');
  });

  it('fetches interactions after overview loads', async () => {
    const mockFetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: mockInteractions, total: 3 },
    }));
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.interactions.length).toBeGreaterThan(0);
    });

    expect(result.current.interactions).toEqual(mockInteractions);
    expect(result.current.interactionsTotal).toBe(3);
  });

  it('sets error when storage is disabled', async () => {
    const overviewNoStorage = { ...mockOverview, storage: { enabled: false, type: '' } };
    const mockFetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': overviewNoStorage,
    }));
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.overview).not.toBeNull();
    });

    expect(result.current.interactionsError).toBe('Storage is disabled.');
    expect(result.current.interactions).toEqual([]);
  });

  it('refreshStats can be called manually', async () => {
    const mockFetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.stats).not.toBeNull();
    });

    const initialCallCount = mockFetch.mock.calls.filter(
      call => call[0] === '/admin/api/stats'
    ).length;

    await act(async () => {
      await result.current.refreshStats();
    });

    const newCallCount = mockFetch.mock.calls.filter(
      call => call[0] === '/admin/api/stats'
    ).length;

    expect(newCallCount).toBe(initialCallCount + 1);
  });

  it('refreshOverview can be called manually', async () => {
    const mockFetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.overview).not.toBeNull();
    });

    const initialCallCount = mockFetch.mock.calls.filter(
      call => call[0] === '/admin/api/overview'
    ).length;

    await act(async () => {
      await result.current.refreshOverview();
    });

    const newCallCount = mockFetch.mock.calls.filter(
      call => call[0] === '/admin/api/overview'
    ).length;

    expect(newCallCount).toBe(initialCallCount + 1);
  });

  it('fetchInteractionDetail returns detail data', async () => {
    const detail = { id: 'test-123', type: 'conversation', messages: [] };
    const mockFetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
      '/interactions/test-123': detail,
    }));
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.overview).not.toBeNull();
    });

    let fetchedDetail;
    await act(async () => {
      fetchedDetail = await result.current.fetchInteractionDetail('test-123');
    });

    expect(fetchedDetail).toEqual(detail);
  });

  it('fetchInteractionDetail returns null on error', async () => {
    const mockFetch = vi.fn((url: string) => {
      if (url.includes('/interactions/nonexistent')) {
        return Promise.resolve({
          ok: false,
          status: 404,
          json: () => Promise.resolve({ error: 'not found' }),
        });
      }
      return createMockFetch({
        '/stats': mockStats,
        '/overview': mockOverview,
        '/interactions': { interactions: [], total: 0 },
      })(url);
    });
    global.fetch = mockFetch;

    const { result } = renderHook(() => useApi(), {
      wrapper: ({ children }) => <ApiProvider>{children}</ApiProvider>,
    });

    await waitFor(() => {
      expect(result.current.overview).not.toBeNull();
    });

    let fetchedDetail;
    await act(async () => {
      fetchedDetail = await result.current.fetchInteractionDetail('nonexistent');
    });

    expect(fetchedDetail).toBeNull();
  });

  it('throws error when used outside ApiProvider', () => {
    expect(() => {
      renderHook(() => useApi());
    }).toThrow('useApi must be used within an ApiProvider');
  });
});

describe('Helper functions', () => {
  describe('formatBytesToMB', () => {
    it('formats bytes to MB', () => {
      expect(formatBytesToMB(15728640)).toBe('15.0 MB');
      expect(formatBytesToMB(1048576)).toBe('1.0 MB');
      expect(formatBytesToMB(52428800)).toBe('50.0 MB');
    });

    it('returns 0 MB for undefined', () => {
      expect(formatBytesToMB(undefined)).toBe('0 MB');
    });

    it('returns 0 MB for 0', () => {
      expect(formatBytesToMB(0)).toBe('0 MB');
    });
  });

  describe('formatTimestamp', () => {
    it('formats epoch timestamp', () => {
      const result = formatTimestamp(1700000000);
      expect(result).not.toBe('--');
      expect(result).toContain('2023');
    });

    it('returns -- for undefined', () => {
      expect(formatTimestamp(undefined)).toBe('--');
    });

    it('returns -- for 0', () => {
      expect(formatTimestamp(0)).toBe('--');
    });
  });

  describe('formatShortDate', () => {
    it('formats epoch to short date', () => {
      const result = formatShortDate(1700000000);
      expect(result).not.toBe('--');
    });

    it('returns -- for undefined', () => {
      expect(formatShortDate(undefined)).toBe('--');
    });
  });

  describe('friendlyDuration', () => {
    it('formats duration string', () => {
      expect(friendlyDuration('2h30m15s')).toBe('2h 30m 15s');
      expect(friendlyDuration('1h0m5s')).toBe('1h 0m 5s');
    });

    it('removes fractional seconds', () => {
      expect(friendlyDuration('2h30m15.123456s')).toBe('2h 30m 15');
    });

    it('returns -- for undefined', () => {
      expect(friendlyDuration(undefined)).toBe('--');
    });

    it('returns -- for empty string', () => {
      expect(friendlyDuration('')).toBe('--');
    });
  });
});
