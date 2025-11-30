import { describe, it, expect, vi, afterEach } from 'vitest';
import { screen, waitFor, fireEvent } from '@testing-library/react';
import { render } from '../test/test-utils';
import { Routing } from './Routing';
import {
  mockStats,
  mockOverview,
  mockMultiTenantOverview,
  mockEmptyOverview,
  mockNullArraysOverview,
  createMockFetch,
} from '../test/mocks';

describe('Routing', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders page header', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    expect(screen.getByText('Model routing rules and tenant configuration')).toBeInTheDocument();
  });

  it('renders default provider banner', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('Default Provider')).toBeInTheDocument();
    });

    expect(screen.getAllByText('openai-provider').length).toBeGreaterThan(0);
  });

  it('shows "Not configured" when no default provider', async () => {
    const overviewNoDefault = {
      ...mockOverview,
      routing: { ...mockOverview.routing, default_provider: '' },
    };
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': overviewNoDefault,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('Default Provider')).toBeInTheDocument();
    });

    expect(screen.getByText('Not configured')).toBeInTheDocument();
  });

  it('renders routing rules section with count', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Routing Rules').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Model-to-provider mappings')).toBeInTheDocument();
    expect(screen.getByText('2 rule(s)')).toBeInTheDocument();
  });

  it('renders routing rules with prefix and exact matches', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('claude-').length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText('gpt-4-turbo').length).toBeGreaterThan(0);
    expect(screen.getAllByText('anthropic-provider').length).toBeGreaterThan(0);
  });

  it('shows empty state when no routing rules', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockEmptyOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Routing Rules').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('No routing rules configured')).toBeInTheDocument();
    expect(screen.getByText('All requests use the default provider')).toBeInTheDocument();
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // This test verifies the fix for the bug where null arrays caused crashes
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockNullArraysOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    // Should not throw error
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Routing Rules').length).toBeGreaterThan(0);
    });

    // Verify the page renders without crashing and shows empty states
    expect(screen.getByText('No routing rules configured')).toBeInTheDocument();
    expect(screen.getByText('0 rule(s)')).toBeInTheDocument();
  });

  it('renders single-tenant mode message', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Tenants').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Single-tenant Mode')).toBeInTheDocument();
    expect(screen.getByText(/running in single-tenant mode/)).toBeInTheDocument();
  });

  it('renders multi-tenant mode with tenant list', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockMultiTenantOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Tenants').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('2 tenant(s)')).toBeInTheDocument();
    expect(screen.getByText('Acme Corp')).toBeInTheDocument();
    expect(screen.getByText('Beta Inc')).toBeInTheDocument();
  });

  it('renders routing summary section', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('Routing Summary')).toBeInTheDocument();
    });

    expect(screen.getByText('Prefix Rules')).toBeInTheDocument();
    expect(screen.getByText('Exact Rules')).toBeInTheDocument();
  });

  it('renders refresh button', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
  });

  it('renders routing explanation section', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('How routing works')).toBeInTheDocument();
    });

    expect(screen.getByText(/Prefix rules:/)).toBeInTheDocument();
    expect(screen.getByText(/Exact rules:/)).toBeInTheDocument();
  });
});
