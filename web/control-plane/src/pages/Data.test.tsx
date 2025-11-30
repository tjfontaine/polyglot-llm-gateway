import { describe, it, expect, vi, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';
import { Data } from './Data';
import { mockStats, mockNullArraysOverview, createMockFetch } from '../test/mocks';

describe('Data page', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('handles null arrays safely when storage enabled', async () => {
    const overviewWithStorage = {
      ...mockNullArraysOverview,
      storage: { enabled: true, type: 'sqlite', path: '/tmp/db' },
    };

    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': overviewWithStorage,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<Data />, { initialRoute: '/admin/data' });

    await waitFor(() => {
      expect(screen.getByText('Data Explorer')).toBeInTheDocument();
    });

    expect(screen.getByText('0 interactions recorded')).toBeInTheDocument();
    expect(screen.getByText('No interactions yet')).toBeInTheDocument();
  });

  it('shows storage disabled state when persistence is off', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': { ...mockNullArraysOverview, storage: { enabled: false, type: '' } },
    }));

    render(<Data />, { initialRoute: '/admin/data' });

    await waitFor(() => {
      expect(screen.getByText('Storage Disabled')).toBeInTheDocument();
    });
    expect(screen.getByText(/Enable storage in your gateway configuration/)).toBeInTheDocument();
  });
});

