import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';
import { Layout } from './Layout';
import { Routes, Route, Outlet } from 'react-router-dom';
import {
  mockStats,
  mockOverview,
  mockMultiTenantOverview,
  mockEmptyOverview,
  mockNullArraysOverview,
  createMockFetch,
} from '../test/mocks';

// Test component that renders Layout with routes
function LayoutWithRoutes() {
  return (
    <Routes>
      <Route path="/admin" element={<Layout />}>
        <Route index element={<div data-testid="dashboard-content">Dashboard</div>} />
        <Route path="topology" element={<div data-testid="topology-content">Topology</div>} />
        <Route path="routing" element={<div data-testid="routing-content">Routing</div>} />
        <Route path="data" element={<div data-testid="data-content">Data</div>} />
      </Route>
    </Routes>
  );
}

describe('Layout', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders header with gateway label', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    expect(screen.getByText('Polyglot gateway')).toBeInTheDocument();
    expect(screen.getByText('Control Plane')).toBeInTheDocument();
  });

  it('renders navigation links', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    expect(screen.getByRole('link', { name: /dashboard/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /topology/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /routing/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /data/i })).toBeInTheDocument();
  });

  it('renders quick stats bar', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Uptime')).toBeInTheDocument();
    });

    expect(screen.getByText('Goroutines')).toBeInTheDocument();
    expect(screen.getByText('Memory')).toBeInTheDocument();
    expect(screen.getByText('GC cycles')).toBeInTheDocument();
  });

  it('displays stats values when loaded', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('42')).toBeInTheDocument(); // num_goroutine
    });

    expect(screen.getByText('15.0 MB')).toBeInTheDocument(); // memory.alloc formatted
    expect(screen.getByText('25')).toBeInTheDocument(); // memory.num_gc
  });

  it('shows single-tenant mode label', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('single-tenant')).toBeInTheDocument();
    });
  });

  it('shows multi-tenant mode label with count', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockMultiTenantOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('multi-tenant (2 tenants)')).toBeInTheDocument();
    });
  });

  it('shows storage type when enabled', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Storage: sqlite')).toBeInTheDocument();
    });
  });

  it('shows storage disabled when not enabled', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockEmptyOverview,
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Storage disabled')).toBeInTheDocument();
    });
  });

  it('shows Go version in header', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Go go1.25.3')).toBeInTheDocument();
    });
  });

  it('shows Responses API badge when enabled', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Responses API')).toBeInTheDocument();
    });
  });

  it('shows Passthrough badge when providers have it enabled', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Passthrough')).toBeInTheDocument();
    });
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // This test verifies the fix for the bug where null arrays caused crashes
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockNullArraysOverview,
    }));

    // Should not throw error
    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    await waitFor(() => {
      expect(screen.getByText('Polyglot gateway')).toBeInTheDocument();
    });

    // Verify the layout renders without crashing
    expect(screen.getByRole('link', { name: /dashboard/i })).toBeInTheDocument();
  });

  it('renders child routes via Outlet', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    expect(screen.getByTestId('dashboard-content')).toBeInTheDocument();
  });

  it('renders footer', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin', withApi: true });

    expect(screen.getByText('Polyglot LLM Gateway Control Plane')).toBeInTheDocument();
  });

  it('highlights active navigation link', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions?limit=100': { interactions: [], total: 0 },
    }));

    render(<LayoutWithRoutes />, { initialRoute: '/admin/topology', withApi: true });

    const topologyLink = screen.getByRole('link', { name: /topology/i });
    expect(topologyLink).toHaveClass('bg-amber-500/20');
  });
});
