import React, { type ReactElement, type ReactNode } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ApiProvider } from '../hooks/useApi';

interface WrapperProps {
  children: ReactNode;
  initialRoute?: string;
}

function AllProviders({ children, initialRoute = '/' }: WrapperProps) {
  return (
    <MemoryRouter initialEntries={[initialRoute]}>
      <ApiProvider>
        {children}
      </ApiProvider>
    </MemoryRouter>
  );
}

function RouterOnlyWrapper({ children, initialRoute = '/' }: WrapperProps) {
  return (
    <MemoryRouter initialEntries={[initialRoute]}>
      {children}
    </MemoryRouter>
  );
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  initialRoute?: string;
  withApi?: boolean;
}

function customRender(
  ui: ReactElement,
  { initialRoute = '/', withApi = true, ...options }: CustomRenderOptions = {}
) {
  const Wrapper = withApi ? AllProviders : RouterOnlyWrapper;
  
  return render(ui, {
    wrapper: ({ children }) => (
      <Wrapper initialRoute={initialRoute}>{children}</Wrapper>
    ),
    ...options,
  });
}

export * from '@testing-library/react';
export { customRender as render };
