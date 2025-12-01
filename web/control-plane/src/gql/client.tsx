/* eslint-disable react-refresh/only-export-components */
import { createClient, cacheExchange, fetchExchange, Provider } from 'urql';
import type { ReactNode } from 'react';

// Create the urql client
const client = createClient({
  url: '/admin/api/graphql',
  exchanges: [cacheExchange, fetchExchange],
});

// Provider component to wrap the app
export function GraphQLProvider({ children }: { children: ReactNode }) {
  return <Provider value={client}>{children}</Provider>;
}

// Export the client for direct use if needed
export { client };
