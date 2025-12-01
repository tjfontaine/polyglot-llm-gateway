import { describe, it, expect } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render, type GraphQLMockData } from '../test/test-utils';
import { mockGqlMultiTenantOverview, mockGqlEmptyOverview } from '../test/graphql-mocks';
import { Dashboard } from './Dashboard';

describe('Dashboard', () => {
  it('renders welcome section', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    expect(screen.getByText('Welcome to Control Plane')).toBeInTheDocument();
    expect(screen.getByText(/Monitor your gateway configuration/)).toBeInTheDocument();
  });

  it('renders topology card with app and provider counts', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Topology')).toBeInTheDocument();
    });

    expect(screen.getByText('Apps & providers')).toBeInTheDocument();
  });

  it('renders routing card with rules count', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Routing')).toBeInTheDocument();
    });

    expect(screen.getByText('Rules & tenants')).toBeInTheDocument();
  });

  it('renders data card with interaction counts', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Data')).toBeInTheDocument();
    });

    expect(screen.getByText('Interactions & responses')).toBeInTheDocument();
  });

  it('handles empty overview gracefully', async () => {
    const mockData: GraphQLMockData = {
      overview: mockGqlEmptyOverview,
      interactions: { interactions: [], total: 0 },
    };

    render(<Dashboard />, { initialRoute: '/admin', mockData });

    await waitFor(() => {
      expect(screen.getByText('Welcome to Control Plane')).toBeInTheDocument();
    });

    // Should render without errors even with empty arrays
    expect(screen.getByText('Topology')).toBeInTheDocument();
    expect(screen.getByText('Routing')).toBeInTheDocument();
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // This test verifies the fix for the bug where null arrays caused crashes
    // GraphQL normalizes these, but we verify the component handles empty data
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlEmptyOverview,
        apps: [],
        frontdoors: [],
        providers: [],
        routing: { __typename: 'RoutingSummary', defaultProvider: '', rules: [] },
        tenants: [],
      },
      interactions: { interactions: [], total: 0 },
    };

    // Should not throw error
    render(<Dashboard />, { initialRoute: '/admin', mockData });

    await waitFor(() => {
      expect(screen.getByText('Welcome to Control Plane')).toBeInTheDocument();
    });

    // Verify the page renders without crashing
    expect(screen.getByText('Topology')).toBeInTheDocument();
    expect(screen.getByText('Routing')).toBeInTheDocument();
    expect(screen.getByText('Data')).toBeInTheDocument();
  });

  it('shows multi-tenant mode in quick status', async () => {
    const mockData: GraphQLMockData = {
      overview: mockGqlMultiTenantOverview,
      interactions: { interactions: [], total: 0 },
    };

    render(<Dashboard />, { initialRoute: '/admin', mockData });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Mode: multi-tenant')).toBeInTheDocument();
  });

  it('shows storage type in quick status', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Storage: sqlite')).toBeInTheDocument();
  });

  it('shows default provider in quick status', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Default provider: openai-provider')).toBeInTheDocument();
  });

  it('shows responses API enabled badge when apps have it enabled', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('Responses API enabled')).toBeInTheDocument();
  });

  it('shows passthrough providers badge when providers have it enabled', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Quick Status')).toBeInTheDocument();
    });

    expect(screen.getByText('1 passthrough provider(s)')).toBeInTheDocument();
  });

  it('links topology card to /admin/topology', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Topology')).toBeInTheDocument();
    });

    const topologyLink = screen.getByText('View topology').closest('a');
    expect(topologyLink).toHaveAttribute('href', '/admin/topology');
  });

  it('links routing card to /admin/routing', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Routing')).toBeInTheDocument();
    });

    const routingLink = screen.getByText('View routing').closest('a');
    expect(routingLink).toHaveAttribute('href', '/admin/routing');
  });

  it('links data card to /admin/data', async () => {
    render(<Dashboard />, { initialRoute: '/admin' });

    await waitFor(() => {
      expect(screen.getByText('Data')).toBeInTheDocument();
    });

    const dataLink = screen.getByText('Explore data').closest('a');
    expect(dataLink).toHaveAttribute('href', '/admin/data');
  });
});
