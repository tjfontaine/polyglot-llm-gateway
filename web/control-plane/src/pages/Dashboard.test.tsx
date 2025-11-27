import { describe, it, expect, vi, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';
import { Dashboard } from './Dashboard';
import {
  mockStats,
  mockOverview,
  mockMultiTenantOverview,
  mockEmptyOverview,
  mockNullArraysOverview,
  mockInteractions,
  createMockFetch,
} from '../test/mocks';

describe('Dashboard', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders welcome section', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: mockInteractions, total: 3 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    expect(screen.getByText('Welcome to Control Plane')).toBeInTheDocument();
    expect(screen.getByText(/Monitor your gateway configuration/)).toBeInTheDocument();
  });

  it('renders topology card with app and provider counts', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: mockInteractions, total: 3 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Topology')).toBeInTheDocument();
    });

    expect(screen.getByText('Apps & providers')).toBeInTheDocument();
  });

  it('renders routing card with rules count', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: mockInteractions, total: 3 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Routing')).toBeInTheDocument();
    });

    expect(screen.getByText('Rules & tenants')).toBeInTheDocument();
  });

  it('renders data card with interaction counts', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: mockInteractions, total: 3 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Data')).toBeInTheDocument();
    });

    expect(screen.getByText('Interactions & responses')).toBeInTheDocument();
  });

  it('handles empty overview gracefully', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockEmptyOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Welcome to Control Plane')).toBeInTheDocument();
    });

    // Should render without errors even with empty arrays
    expect(screen.getByText('Topology')).toBeInTheDocument();
    expect(screen.getByText('Routing')).toBeInTheDocument();
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // This test verifies the fix for the bug where null arrays caused crashes
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockNullArraysOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    // Should not throw error
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Welcome to Control Plane')).toBeInTheDocument();
    });

    // Verify the page renders without crashing
    expect(screen.getByText('Topology')).toBeInTheDocument();
    expect(screen.getByText('Routing')).toBeInTheDocument();
    expect(screen.getByText('Data')).toBeInTheDocument();
  });

  it('shows multi-tenant mode in quick status', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockMultiTenantOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Mode: multi-tenant')).toBeInTheDocument();
  });

  it('shows storage type in quick status', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Storage: sqlite')).toBeInTheDocument();
  });

  it('shows default provider in quick status', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Default provider: openai-provider')).toBeInTheDocument();
  });

  it('shows responses API enabled badge when apps have it enabled', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Responses API enabled')).toBeInTheDocument();
  });

  it('shows passthrough providers badge when providers have it enabled', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('1 passthrough provider(s)')).toBeInTheDocument();
  });

  it('links topology card to /admin/topology', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Topology')).toBeInTheDocument();
    });

    const topologyLink = screen.getByText('View topology').closest('a');
    expect(topologyLink).toHaveAttribute('href', '/admin/topology');
  });

  it('links routing card to /admin/routing', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Routing')).toBeInTheDocument();
    });

    const routingLink = screen.getByText('View routing').closest('a');
    expect(routingLink).toHaveAttribute('href', '/admin/routing');
  });

  it('links data card to /admin/data', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Data')).toBeInTheDocument();
    });

    const dataLink = screen.getByText('Explore data').closest('a');
    expect(dataLink).toHaveAttribute('href', '/admin/data');
  });
});
