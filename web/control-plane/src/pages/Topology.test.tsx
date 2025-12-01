import { describe, it, expect } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render, mockGqlOverview, type GraphQLMockData } from '../test/test-utils';
import { mockGqlEmptyOverview } from '../test/graphql-mocks';
import { Topology } from './Topology';

describe('Topology', () => {
  it('renders page header', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    expect(screen.getByText('Applications and provider configurations')).toBeInTheDocument();
  });

  it('renders applications section with count', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getAllByText('Applications').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Configured frontdoor endpoints')).toBeInTheDocument();
    expect(screen.getByText('2 app(s)')).toBeInTheDocument();
  });

  it('renders app details', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('main-app')).toBeInTheDocument();
    });

    expect(screen.getByText('anthropic-app')).toBeInTheDocument();
    expect(screen.getByText('/v1')).toBeInTheDocument();
    expect(screen.getByText('/anthropic')).toBeInTheDocument();
  });

  it('shows Responses API badge for enabled apps', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('main-app')).toBeInTheDocument();
    });

    // main-app has enableResponses: true
    const responsesBadges = screen.getAllByText('Responses API');
    expect(responsesBadges.length).toBeGreaterThan(0);
  });

  it('renders model routing configuration', async () => {
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
    const mockData: GraphQLMockData = {
      overview: mockGqlEmptyOverview,
    };

    render(<Topology />, { initialRoute: '/admin/topology', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Applications').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('No applications configured')).toBeInTheDocument();
  });

  it('handles null arrays from backend gracefully (bug fix verification)', async () => {
    // GraphQL normalizes null arrays to empty arrays
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlEmptyOverview,
        apps: [],
        frontdoors: [],
        providers: [],
      },
    };

    // Should not throw error
    render(<Topology />, { initialRoute: '/admin/topology', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Applications').length).toBeGreaterThan(0);
    });

    // Verify the page renders without crashing
    expect(screen.getAllByText('Providers').length).toBeGreaterThan(0);
    expect(screen.getByText('0 provider(s)')).toBeInTheDocument();
  });

  it('renders providers section with count', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getAllByText('Providers').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Connected LLM backends')).toBeInTheDocument();
    expect(screen.getByText('2 provider(s)')).toBeInTheDocument();
  });

  it('renders provider details', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('openai-provider')).toBeInTheDocument();
    });

    expect(screen.getAllByText('anthropic-provider').length).toBeGreaterThan(0);
  });

  it('shows provider type information', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('openai-provider')).toBeInTheDocument();
    });

    expect(screen.getByText('OpenAI-compatible API')).toBeInTheDocument();
    expect(screen.getByText('Anthropic Claude API')).toBeInTheDocument();
  });

  it('shows passthrough badge for enabled providers', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getAllByText('anthropic-provider').length).toBeGreaterThan(0);
    });

    // anthropic-provider has enablePassthrough: true
    expect(screen.getAllByText('Passthrough').length).toBeGreaterThan(0);
  });

  it('shows responses ready badge for supported providers', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('openai-provider')).toBeInTheDocument();
    });

    const responsesBadges = screen.getAllByText('Responses ready');
    expect(responsesBadges.length).toBe(2); // Both providers support responses
  });

  it('shows empty state when no providers', async () => {
    const mockData: GraphQLMockData = {
      overview: mockGqlEmptyOverview,
    };

    render(<Topology />, { initialRoute: '/admin/topology', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Providers').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('No providers configured')).toBeInTheDocument();
  });

  it('renders topology summary section', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    await waitFor(() => {
      expect(screen.getByText('Topology Summary')).toBeInTheDocument();
    });

    expect(screen.getByText('Responses-enabled')).toBeInTheDocument();
  });

  it('renders refresh button', async () => {
    render(<Topology />, { initialRoute: '/admin/topology' });

    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
  });

  it('falls back to frontdoors when apps array is empty', async () => {
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlOverview,
        apps: [],
      },
    };

    render(<Topology />, { initialRoute: '/admin/topology', mockData });

    await waitFor(() => {
      expect(screen.getAllByText('Applications').length).toBeGreaterThan(0);
    });

    // Should show frontdoors instead
    expect(screen.getByText('/v1')).toBeInTheDocument();
    expect(screen.getByText('/anthropic')).toBeInTheDocument();
  });
});
