import { describe, it, expect } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { render, type GraphQLMockData } from '../test/test-utils';
import { mockGqlEmptyOverview } from '../test/graphql-mocks';
import { Data } from './Data';

describe('Data page', () => {
  it('handles null arrays safely when storage enabled', async () => {
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlEmptyOverview,
        storage: { __typename: 'StorageSummary', enabled: true, type: 'sqlite', path: '/tmp/db' },
      },
      interactions: { interactions: [], total: 0 },
    };

    render(<Data />, { initialRoute: '/admin/data', mockData });

    await waitFor(() => {
      expect(screen.getByText('Data Explorer')).toBeInTheDocument();
    });

    expect(screen.getByText('0 interactions recorded')).toBeInTheDocument();
    expect(screen.getByText('No interactions yet')).toBeInTheDocument();
  });

  it('shows storage disabled state when persistence is off', async () => {
    const mockData: GraphQLMockData = {
      overview: {
        ...mockGqlEmptyOverview,
        storage: { __typename: 'StorageSummary', enabled: false, type: '', path: null },
      },
    };

    render(<Data />, { initialRoute: '/admin/data', mockData });

    await waitFor(() => {
      expect(screen.getByText('Storage Disabled')).toBeInTheDocument();
    });
    expect(screen.getByText(/Enable storage in your gateway configuration/)).toBeInTheDocument();
  });
});

