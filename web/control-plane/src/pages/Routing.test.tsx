import { describe, it, expect } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render, mockGqlOverview, type GraphQLMockData } from '../test/test-utils';
import { mockGqlMultiTenantOverview, mockGqlEmptyOverview } from '../test/graphql-mocks';
import { Routing } from './Routing';

describe('Routing', () => {
  it('renders page header', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    expect(screen.getByText('Model routing rules and tenant configuration')).toBeInTheDocument();
  });

  it('renders default provider banner', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('Default Provider')).toBeInTheDocument();
    });

    expect(screen.getAllByText('openai-provider').length).toBeGreaterThan(0);
  });

  it('shows "Not configured" when no default provider', async () => {
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlOverview,
        routing: { ...mockGqlOverview.routing, defaultProvider: '' },
      },
    };

    render(<Routing />, { initialRoute: '/admin/routing', mockData });

    await waitFor(() => {
      expect(screen.getByText('Default Provider')).toBeInTheDocument();
    });

    expect(screen.getByText('Not configured')).toBeInTheDocument();
  });

  it('renders routing rules section with count', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Routing Rules').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Model-to-provider mappings')).toBeInTheDocument();
    expect(screen.getByText('2 rule(s)')).toBeInTheDocument();
  });

  it('renders routing rules with prefix and exact matches', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('claude-').length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText('gpt-4-turbo').length).toBeGreaterThan(0);
    expect(screen.getAllByText('anthropic-provider').length).toBeGreaterThan(0);
  });

  it('shows empty state when no routing rules', async () => {
    const mockData: GraphQLMockData = {
      overview: mockGqlEmptyOverview,
    };

    render(<Routing />, { initialRoute: '/admin/routing', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Routing Rules').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('No routing rules configured')).toBeInTheDocument();
    expect(screen.getByText('All requests use the default provider')).toBeInTheDocument();
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // GraphQL normalizes null arrays to empty arrays
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlEmptyOverview,
        routing: { __typename: 'RoutingSummary', defaultProvider: '', rules: [] },
      },
    };

    // Should not throw error
    render(<Routing />, { initialRoute: '/admin/routing', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Routing Rules').length).toBeGreaterThan(0);
    });

    // Verify the page renders without crashing and shows empty states
    expect(screen.getByText('No routing rules configured')).toBeInTheDocument();
    expect(screen.getByText('0 rule(s)')).toBeInTheDocument();
  });

  it('renders single-tenant mode message', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getAllByText('Tenants').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Single-tenant Mode')).toBeInTheDocument();
    expect(screen.getByText(/running in single-tenant mode/)).toBeInTheDocument();
  });

  it('renders multi-tenant mode with tenant list', async () => {
    const mockData: GraphQLMockData = {
      overview: mockGqlMultiTenantOverview,
    };

    render(<Routing />, { initialRoute: '/admin/routing', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Tenants').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('2 tenant(s)')).toBeInTheDocument();
    expect(screen.getByText('Acme Corp')).toBeInTheDocument();
    expect(screen.getByText('Beta Inc')).toBeInTheDocument();
  });

  it('renders routing summary section', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('Routing Summary')).toBeInTheDocument();
    });

    expect(screen.getByText('Prefix Rules')).toBeInTheDocument();
    expect(screen.getByText('Exact Rules')).toBeInTheDocument();
  });

  it('renders refresh button', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
  });

  it('renders routing explanation section', async () => {
    render(<Routing />, { initialRoute: '/admin/routing' });

    await waitFor(() => {
      expect(screen.getByText('How routing works')).toBeInTheDocument();
    });

    expect(screen.getByText(/Prefix rules:/)).toBeInTheDocument();
    expect(screen.getByText(/Exact rules:/)).toBeInTheDocument();
  });
});
