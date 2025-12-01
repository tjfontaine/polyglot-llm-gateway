import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { render } from '../../test/test-utils';
import { ShadowSummary } from './ShadowSummary';
import { DivergenceList } from './DivergenceList';
import { ShadowComparison } from './ShadowComparison';
import { ShadowPanel } from './ShadowPanel';
import { DivergenceType, type ShadowResult, type Divergence, type Interaction } from '../../gql/graphql';

const mockShadowResult: ShadowResult = {
    __typename: 'ShadowResult',
    id: 'shadow-123',
    interactionId: 'interaction-456',
    providerName: 'openai',
    providerModel: 'gpt-4o',
    durationNs: '1500000000', // 1.5 seconds
    tokensIn: 100,
    tokensOut: 50,
    hasStructuralDivergence: false,
    createdAt: String(Date.now() / 1000),
    response: {
        __typename: 'ShadowResponse',
        canonical: { content: 'Hello from shadow!' },
    },
};

const mockShadowResultWithError: ShadowResult = {
    ...mockShadowResult,
    id: 'shadow-error',
    error: {
        __typename: 'ShadowError',
        type: 'rate_limit_error',
        code: '429',
        message: 'Rate limit exceeded',
    },
    response: undefined,
};

const mockShadowResultWithDivergences: ShadowResult = {
    ...mockShadowResult,
    id: 'shadow-div',
    hasStructuralDivergence: true,
    divergences: [
        {
            __typename: 'Divergence',
            type: DivergenceType.MissingField,
            path: 'response.usage.prompt_tokens',
            description: 'Field missing in shadow response',
            primary: '100',
        },
        {
            __typename: 'Divergence',
            type: DivergenceType.TypeMismatch,
            path: 'response.model',
            description: 'Type differs between primary and shadow',
            primary: 'claude-3',
            shadow: 'gpt-4o',
        },
    ],
};

const mockPrimaryInteraction: Interaction = {
    __typename: 'Interaction',
    id: 'interaction-456',
    tenantId: 'tenant-1',
    frontdoor: 'anthropic',
    provider: 'anthropic',
    requestedModel: 'claude-3-sonnet',
    servedModel: 'claude-3-sonnet',
    streaming: false,
    status: 'completed',
    duration: '1.2s',
    durationNs: '1200000000',
    createdAt: String(Date.now() / 1000),
    updatedAt: String(Date.now() / 1000),
    request: {
        __typename: 'InteractionRequest',
        canonical: { model: 'claude-3-sonnet', messages: [] },
    },
    response: {
        __typename: 'InteractionResponse',
        canonical: { content: 'Hello from primary!' },
        usage: { inputTokens: 100, outputTokens: 50, totalTokens: 150 },
    },
};

describe('ShadowSummary', () => {
    it('renders provider name and model', () => {
        render(<ShadowSummary shadow={mockShadowResult} />);

        expect(screen.getByText('openai')).toBeInTheDocument();
        expect(screen.getByText('gpt-4o')).toBeInTheDocument();
    });

    it('shows match status when no errors or divergences', () => {
        render(<ShadowSummary shadow={mockShadowResult} />);

        expect(screen.getByText('Match')).toBeInTheDocument();
    });

    it('shows error status when shadow has error', () => {
        render(<ShadowSummary shadow={mockShadowResultWithError} />);

        expect(screen.getByText('Error')).toBeInTheDocument();
        expect(screen.getByText('rate_limit_error')).toBeInTheDocument();
    });

    it('shows divergence count when divergences exist', () => {
        render(<ShadowSummary shadow={mockShadowResultWithDivergences} />);

        expect(screen.getByText('2 divergences')).toBeInTheDocument();
    });

    it('displays duration and token counts', () => {
        render(<ShadowSummary shadow={mockShadowResult} />);

        // Component displays as 1.50s not 1500ms
        expect(screen.getByText('1.50s')).toBeInTheDocument();
        // Token display format uses " in / " and " out" as separate text
        // Just verify we can find the tokens container
        expect(screen.getByText(/in \//)).toBeInTheDocument();
    });

    it('calls onClick when clicked', () => {
        const onClick = vi.fn();
        render(<ShadowSummary shadow={mockShadowResult} onClick={onClick} />);

        fireEvent.click(screen.getByText('openai').closest('div')!.parentElement!.parentElement!);
        expect(onClick).toHaveBeenCalled();
    });

    it('shows selected state', () => {
        const { container } = render(<ShadowSummary shadow={mockShadowResult} selected />);

        expect(container.firstChild).toHaveClass('border-violet-500/50');
    });
});

describe('DivergenceList', () => {
    const divergences: Divergence[] = [
        {
            __typename: 'Divergence',
            type: DivergenceType.MissingField,
            path: 'response.usage',
            description: 'Field missing in shadow',
            primary: { tokens: 100 },  // Use object, not string - GraphQL JSON scalar
        },
        {
            __typename: 'Divergence',
            type: DivergenceType.ExtraField,
            path: 'response.metadata',
            description: 'Extra field in shadow',
            shadow: { extra: 'data' },  // Use object, not string
        },
        {
            __typename: 'Divergence',
            type: DivergenceType.TypeMismatch,
            path: 'response.status',
            description: 'Type differs',
            primary: 'completed',
            shadow: '200',
        },
    ];

    it('shows no divergences message when empty', () => {
        render(<DivergenceList divergences={[]} />);

        expect(screen.getByText('No structural divergences detected')).toBeInTheDocument();
    });

    it('shows divergence count', () => {
        render(<DivergenceList divergences={divergences} />);

        expect(screen.getByText(/3 structural divergences detected/)).toBeInTheDocument();
    });

    it('groups divergences by type', () => {
        render(<DivergenceList divergences={divergences} />);

        expect(screen.getByText('Missing Fields')).toBeInTheDocument();
        expect(screen.getByText('Extra Fields')).toBeInTheDocument();
        expect(screen.getByText('Type Mismatches')).toBeInTheDocument();
    });

    it('displays JSON paths', () => {
        render(<DivergenceList divergences={divergences} />);

        expect(screen.getByText('response.usage')).toBeInTheDocument();
        expect(screen.getByText('response.metadata')).toBeInTheDocument();
        expect(screen.getByText('response.status')).toBeInTheDocument();
    });

    it('shows primary and shadow values', () => {
        render(<DivergenceList divergences={divergences} />);

        // JSON.stringify on an object produces {"tokens":100}
        expect(screen.getByText(/\{"tokens":100\}/)).toBeInTheDocument();
        expect(screen.getByText(/\{"extra":"data"\}/)).toBeInTheDocument();
    });
});

describe('ShadowComparison', () => {
    it('renders metrics for shadow', () => {
        render(<ShadowComparison shadow={mockShadowResult} />);

        expect(screen.getByText('Shadow: openai')).toBeInTheDocument();
        expect(screen.getByText('gpt-4o')).toBeInTheDocument();
        // Duration displayed as seconds
        expect(screen.getByText('1.50s')).toBeInTheDocument();
    });

    it('renders primary metrics when provided', () => {
        render(<ShadowComparison shadow={mockShadowResult} primary={mockPrimaryInteraction} />);

        expect(screen.getByText('Primary')).toBeInTheDocument();
        expect(screen.getByText('anthropic')).toBeInTheDocument();
    });

    it('shows error state when shadow has error', () => {
        render(<ShadowComparison shadow={mockShadowResultWithError} />);

        expect(screen.getByText('Shadow Execution Error')).toBeInTheDocument();
        expect(screen.getByText('rate_limit_error')).toBeInTheDocument();
        expect(screen.getByText('Rate limit exceeded')).toBeInTheDocument();
    });

    it('shows divergences when present', () => {
        render(<ShadowComparison shadow={mockShadowResultWithDivergences} />);

        expect(screen.getByText(/2 structural divergences detected/)).toBeInTheDocument();
    });

    it('allows switching between request and response views', () => {
        render(<ShadowComparison shadow={mockShadowResult} />);

        const requestBtn = screen.getByRole('button', { name: 'Request' });
        const responseBtn = screen.getByRole('button', { name: 'Response' });

        expect(responseBtn).toHaveClass('bg-violet-500/20'); // Response is default

        fireEvent.click(requestBtn);
        expect(requestBtn).toHaveClass('bg-violet-500/20');
    });

    it('shows side-by-side mode toggle when primary is provided', () => {
        render(<ShadowComparison shadow={mockShadowResult} primary={mockPrimaryInteraction} />);

        expect(screen.getByText('Side by Side')).toBeInTheDocument();
        expect(screen.getByText('Shadow Only')).toBeInTheDocument();
    });
});

describe('ShadowPanel', () => {
    // With the mock GraphQL client, data loads synchronously
    it('renders shadow results', () => {
        render(<ShadowPanel interactionId="test-123" />);

        // The mock provides shadow results immediately
        expect(screen.getByText('Shadow Results')).toBeInTheDocument();
    });
});
