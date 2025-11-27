import { describe, it, expect, vi, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';
import { Topology } from './Topology';
import {
  mockStats,
  mockOverview,
  mockEmptyOverview,
  mockNullArraysOverview,
  createMockFetch,
} from '../test/mocks';

describe('Topology', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders page header', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    expect(screen.getByText('Applications and provider configurations')).toBeInTheDocument();
  });

  it('renders applications section with count', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Applications')).toBeInTheDocument();
    });

    expect(screen.getByText('Configured frontdoor endpoints')).toBeInTheDocument();
    expect(screen.getByText('2 app(s)')).toBeInTheDocument();
  });

  it('renders app details', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('main-app')).toBeInTheDocument();
    });

    expect(screen.getByText('anthropic-app')).toBeInTheDocument();
    expect(screen.getByText('/v1')).toBeInTheDocument();
    expect(screen.getByText('/anthropic')).toBeInTheDocument();
  });

  it('shows Responses API badge for enabled apps', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('main-app')).toBeInTheDocument();
    });

    // main-app has enable_responses: true
    const responsesBadges = screen.getAllByText('Responses API');
    expect(responsesBadges.length).toBeGreaterThan(0);
  });

  it('renders model routing configuration', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('main-app')).toBeInTheDocument();
    });

    // Check for model routing section in main-app
    expect(screen.getByText('Model Routing')).toBeInTheDocument();
    expect(screen.getByText('claude-*')).toBeInTheDocument();
    expect(screen.getByText('gpt-3.5-turbo')).toBeInTheDocument();
  });

  it('shows empty state when no applications', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockEmptyOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Applications')).toBeInTheDocument();
    });

    expect(screen.getByText('No applications configured')).toBeInTheDocument();
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // This test verifies the fix for the bug where null arrays caused crashes
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockNullArraysOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    // Should not throw error
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Applications')).toBeInTheDocument();
    });

    // Verify the page renders without crashing
    expect(screen.getByText('Providers')).toBeInTheDocument();
    expect(screen.getByText('0 provider(s)')).toBeInTheDocument();
  });

  it('renders providers section with count', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Providers')).toBeInTheDocument();
    });

    expect(screen.getByText('Connected LLM backends')).toBeInTheDocument();
    expect(screen.getByText('2 provider(s)')).toBeInTheDocument();
  });

  it('renders provider details', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('openai-provider')).toBeInTheDocument();
    });

    expect(screen.getByText('anthropic-provider')).toBeInTheDocument();
  });

  it('shows provider type information', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('openai-provider')).toBeInTheDocument();
    });

    expect(screen.getByText('OpenAI-compatible API')).toBeInTheDocument();
    expect(screen.getByText('Anthropic Claude API')).toBeInTheDocument();
  });

  it('shows passthrough badge for enabled providers', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('anthropic-provider')).toBeInTheDocument();
    });

    // anthropic-provider has enable_passthrough: true
    expect(screen.getByText('Passthrough')).toBeInTheDocument();
  });

  it('shows responses ready badge for supported providers', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('openai-provider')).toBeInTheDocument();
    });

    const responsesBadges = screen.getAllByText('Responses ready');
    expect(responsesBadges.length).toBe(2); // Both providers support responses
  });

  it('shows empty state when no providers', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockEmptyOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Providers')).toBeInTheDocument();
    });

    expect(screen.getByText('No providers configured')).toBeInTheDocument();
  });

  it('renders topology summary section', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Topology Summary')).toBeInTheDocument();
    });

    expect(screen.getByText('Responses-enabled')).toBeInTheDocument();
  });

  it('renders refresh button', async () => {
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': mockOverview,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
  });

  it('falls back to frontdoors when apps array is empty', async () => {
    const overviewWithFrontdoors = {
      ...mockOverview,
      apps: [],
    };
    global.fetch = vi.fn(createMockFetch({
      '/stats': mockStats,
      '/overview': overviewWithFrontdoors,
      '/interactions': { interactions: [], total: 0 },
    }));

    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Applications')).toBeInTheDocument();
    });

    // Should show frontdoors instead
    expect(screen.getByText('/v1')).toBeInTheDocument();
    expect(screen.getByText('/anthropic')).toBeInTheDocument();
  });
});
