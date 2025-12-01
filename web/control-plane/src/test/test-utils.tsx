import { type ReactElement, type ReactNode, useMemo } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ApiProvider } from '../hooks/useApi';
import { createClient, Provider as UrqlProvider, type Exchange } from 'urql';
import { pipe, map } from 'wonka';
import {
  mockGqlStats,
  mockGqlOverview,
  mockGqlInteractions,
  mockGqlInteraction,
  mockGqlShadowResult,
} from './graphql-mocks';
import type { Stats, Overview, InteractionSummary, Interaction, ShadowResult } from '../gql/graphql';

// GraphQL mock data registry
export interface GraphQLMockData {
  stats?: Stats | null;
  overview?: Overview | null;
  interactions?: { interactions: InteractionSummary[]; total: number };
  interaction?: Interaction | null;
  shadowResults?: { interactionId: string; shadows: ShadowResult[] };
  shadow?: ShadowResult | null;
  interactionEvents?: { interactionId: string; events: unknown[] };
  divergentShadows?: { interactions: InteractionSummary[]; total: number; limit: number; offset: number };
}

// Default mock data
const defaultMockData: GraphQLMockData = {
  stats: mockGqlStats,
  overview: mockGqlOverview,
  interactions: { interactions: mockGqlInteractions, total: mockGqlInteractions.length },
  interaction: mockGqlInteraction,
  shadowResults: { interactionId: 'int-001', shadows: [mockGqlShadowResult] },
  shadow: mockGqlShadowResult,
  interactionEvents: { interactionId: 'int-001', events: [] },
  divergentShadows: { interactions: [], total: 0, limit: 100, offset: 0 },
};

// Create a mock exchange that returns test data
function createMockExchange(mockData: GraphQLMockData = {}): Exchange {
  const data = { ...defaultMockData, ...mockData };

  return () => (ops$) => {
    return pipe(
      ops$,
      map((operation) => {
        // Parse the query to determine which field is being queried
        const queryStr = operation.query.loc?.source.body || '';

        let responseData: Record<string, unknown> = {};

        if (queryStr.includes('query Stats')) {
          responseData = { stats: data.stats };
        } else if (queryStr.includes('query Overview')) {
          responseData = { overview: data.overview };
        } else if (queryStr.includes('query Interactions(')) {
          responseData = { interactions: data.interactions };
        } else if (queryStr.includes('query Interaction(')) {
          responseData = { interaction: data.interaction };
        } else if (queryStr.includes('query ShadowResults')) {
          responseData = { shadowResults: data.shadowResults };
        } else if (queryStr.includes('query Shadow(')) {
          responseData = { shadow: data.shadow };
        } else if (queryStr.includes('query InteractionEvents')) {
          responseData = { interactionEvents: data.interactionEvents };
        } else if (queryStr.includes('query DivergentShadows')) {
          responseData = { divergentShadows: data.divergentShadows };
        }

        return {
          operation,
          data: responseData,
          stale: false,
          hasNext: false,
        };
      })
    );
  };
}

// Create a mock urql client
function createMockClient(mockData?: GraphQLMockData) {
  return createClient({
    url: '/admin/api/graphql',
    exchanges: [createMockExchange(mockData)],
  });
}

interface WrapperProps {
  children: ReactNode;
  initialRoute?: string;
  mockData?: GraphQLMockData;
}

/* eslint-disable react-refresh/only-export-components */
function AllProviders({ children, initialRoute = '/', mockData }: WrapperProps) {
  const client = useMemo(() => createMockClient(mockData), [mockData]);

  return (
    <MemoryRouter initialEntries={[initialRoute]}>
      <UrqlProvider value={client}>
        <ApiProvider>
          {children}
        </ApiProvider>
      </UrqlProvider>
    </MemoryRouter>
  );
}

function RouterOnlyWrapper({ children, initialRoute = '/', mockData }: WrapperProps) {
  const client = useMemo(() => createMockClient(mockData), [mockData]);

  return (
    <MemoryRouter initialEntries={[initialRoute]}>
      <UrqlProvider value={client}>
        {children}
      </UrqlProvider>
    </MemoryRouter>
  );
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  initialRoute?: string;
  withApi?: boolean;
  mockData?: GraphQLMockData;
}

function customRender(
  ui: ReactElement,
  { initialRoute = '/', withApi = true, mockData, ...options }: CustomRenderOptions = {}
) {
  const Wrapper = withApi ? AllProviders : RouterOnlyWrapper;

  return render(ui, {
    wrapper: ({ children }) => (
      <Wrapper initialRoute={initialRoute} mockData={mockData}>{children}</Wrapper>
    ),
    ...options,
  });
}

export * from '@testing-library/react';
export { customRender as render };
export {
  mockGqlStats,
  mockGqlOverview,
  mockGqlInteractions,
  mockGqlInteraction,
  mockGqlShadowResult,
} from './graphql-mocks';
