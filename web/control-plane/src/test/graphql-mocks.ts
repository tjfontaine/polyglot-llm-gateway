/**
 * GraphQL mock data for testing.
 * These mocks are aligned with the GraphQL schema and generated types.
 */
import type {
    Stats,
    Overview,
    InteractionSummary,
    Interaction,
    ShadowResult,
    DivergenceType,
} from '../gql/graphql';

// Stats mock (matches Stats type from GraphQL)
export const mockGqlStats: Stats = {
    __typename: 'Stats',
    uptime: '2h30m15s',
    goVersion: 'go1.25.3',
    numGoroutine: 42,
    memory: {
        __typename: 'MemoryStats',
        alloc: 15728640,
        totalAlloc: 31457280,
        sys: 67108864,
        numGC: 25,
    },
};

// Overview mock (matches Overview type from GraphQL)
export const mockGqlOverview: Overview = {
    __typename: 'Overview',
    mode: 'single-tenant',
    storage: {
        __typename: 'StorageSummary',
        enabled: true,
        type: 'sqlite',
        path: '/data/gateway.db',
    },
    apps: [
        {
            __typename: 'AppSummary',
            name: 'main-app',
            frontdoor: 'openai',
            path: '/v1',
            provider: 'openai-provider',
            defaultModel: 'gpt-4',
            enableResponses: true,
            modelRouting: {
                __typename: 'ModelRoutingSummary',
                prefixProviders: { 'claude-': 'anthropic-provider' },
                rewrites: [
                    {
                        __typename: 'ModelRewriteSummary',
                        modelExact: 'gpt-3.5-turbo',
                        modelPrefix: null,
                        provider: 'openai-provider',
                        model: 'gpt-4o-mini',
                    },
                ],
            },
        },
        {
            __typename: 'AppSummary',
            name: 'anthropic-app',
            frontdoor: 'anthropic',
            path: '/anthropic',
            provider: 'anthropic-provider',
            defaultModel: null,
            enableResponses: false,
            modelRouting: null,
        },
    ],
    frontdoors: [
        {
            __typename: 'FrontdoorSummary',
            type: 'openai',
            path: '/v1',
            provider: 'openai-provider',
            defaultModel: 'gpt-4',
        },
        {
            __typename: 'FrontdoorSummary',
            type: 'anthropic',
            path: '/anthropic',
            provider: 'anthropic-provider',
            defaultModel: null,
        },
    ],
    providers: [
        {
            __typename: 'ProviderSummary',
            name: 'openai-provider',
            type: 'openai',
            baseUrl: 'https://api.openai.com/v1',
            supportsResponses: true,
            enablePassthrough: false,
        },
        {
            __typename: 'ProviderSummary',
            name: 'anthropic-provider',
            type: 'anthropic',
            baseUrl: 'https://api.anthropic.com',
            supportsResponses: true,
            enablePassthrough: true,
        },
    ],
    routing: {
        __typename: 'RoutingSummary',
        defaultProvider: 'openai-provider',
        rules: [
            { __typename: 'RoutingRule', modelPrefix: 'claude-', modelExact: null, provider: 'anthropic-provider' },
            { __typename: 'RoutingRule', modelPrefix: null, modelExact: 'gpt-4-turbo', provider: 'openai-provider' },
        ],
    },
    tenants: [],
};

// Multi-tenant overview
export const mockGqlMultiTenantOverview: Overview = {
    ...mockGqlOverview,
    mode: 'multi-tenant',
    tenants: [
        {
            __typename: 'TenantSummary',
            id: 'tenant-1',
            name: 'Acme Corp',
            providerCount: 2,
            routingRules: 3,
            supportsTenant: true,
        },
        {
            __typename: 'TenantSummary',
            id: 'tenant-2',
            name: 'Beta Inc',
            providerCount: 1,
            routingRules: 1,
            supportsTenant: true,
        },
    ],
};

// Empty overview
export const mockGqlEmptyOverview: Overview = {
    __typename: 'Overview',
    mode: 'single-tenant',
    storage: { __typename: 'StorageSummary', enabled: false, type: '', path: null },
    apps: [],
    frontdoors: [],
    providers: [],
    routing: { __typename: 'RoutingSummary', defaultProvider: '', rules: [] },
    tenants: [],
};

// Interaction summaries
export const mockGqlInteractions: InteractionSummary[] = [
    {
        __typename: 'InteractionSummary',
        id: 'conv-123',
        type: 'conversation',
        status: null,
        model: 'gpt-4',
        metadata: { title: 'Test Conversation' },
        messageCount: 5,
        previousResponseId: null,
        createdAt: 1700000000,
        updatedAt: 1700001000,
    },
    {
        __typename: 'InteractionSummary',
        id: 'resp-456',
        type: 'response',
        status: 'completed',
        model: 'claude-3-opus',
        metadata: {},
        messageCount: null,
        previousResponseId: null,
        createdAt: 1700000500,
        updatedAt: 1700000600,
    },
    {
        __typename: 'InteractionSummary',
        id: 'resp-789',
        type: 'response',
        status: 'in_progress',
        model: 'gpt-4-turbo',
        metadata: null,
        messageCount: null,
        previousResponseId: 'resp-456',
        createdAt: 1700000700,
        updatedAt: 1700000800,
    },
];

// Full interaction detail
export const mockGqlInteraction: Interaction = {
    __typename: 'Interaction',
    id: 'int-001',
    status: 'completed',
    frontdoor: 'openai',
    provider: 'openai-provider',
    requestedModel: 'gpt-4',
    servedModel: 'gpt-4-0613',
    providerModel: 'gpt-4-0613',
    streaming: false,
    tenantId: '',
    appName: 'main-app',
    duration: '1.5s',
    durationNs: 1500000000,
    metadata: { title: 'Test interaction' },
    createdAt: 1700000000,
    updatedAt: 1700001000,
    requestHeaders: { 'content-type': 'application/json' },
    request: {
        __typename: 'InteractionRequest',
        raw: { model: 'gpt-4', messages: [{ role: 'user', content: 'Hello' }] },
        canonical: { model: 'gpt-4', messages: [{ role: 'user', content: [{ type: 'text', text: 'Hello' }] }] },
        providerRequest: null,
        unmappedFields: null,
    },
    response: {
        __typename: 'InteractionResponse',
        raw: { id: 'resp-001', choices: [{ message: { content: 'Hi there!' } }] },
        canonical: { content: [{ type: 'text', text: 'Hi there!' }] },
        clientResponse: null,
        finishReason: 'stop',
        usage: { __typename: 'Usage', inputTokens: 10, outputTokens: 20, totalTokens: 30 },
        unmappedFields: null,
    },
    error: null,
    transformationSteps: null,
    shadows: null,
};

// Shadow result mock
export const mockGqlShadowResult: ShadowResult = {
    __typename: 'ShadowResult',
    id: 'shadow-001',
    interactionId: 'int-001',
    providerName: 'anthropic-provider',
    providerModel: 'claude-3-opus',
    durationNs: 2000000000,
    createdAt: 1700000100,
    tokensIn: 15,
    tokensOut: 25,
    hasStructuralDivergence: true,
    request: {
        __typename: 'ShadowRequest',
        canonical: { model: 'gpt-4', messages: [] },
        providerRequest: { model: 'claude-3-opus', messages: [] },
    },
    response: {
        __typename: 'ShadowResponse',
        raw: { content: [{ text: 'Shadow response' }] },
        canonical: { content: [{ type: 'text', text: 'Shadow response' }] },
        clientResponse: null,
        finishReason: 'end_turn',
        usage: {
            __typename: 'ShadowUsage',
            promptTokens: 15,
            completionTokens: 25,
            totalTokens: 40,
        },
    },
    error: null,
    divergences: [
        {
            __typename: 'Divergence',
            type: 'TYPE_MISMATCH' as DivergenceType,
            path: '.response.tokens',
            description: 'Primary has number, shadow-sm has string',
            primary: 100,
            shadow: '100',
        },
    ],
};

// Shadow result without divergence
export const mockGqlShadowNoDivergence: ShadowResult = {
    ...mockGqlShadowResult,
    id: 'shadow-002',
    hasStructuralDivergence: false,
    divergences: null,
};

// Shadow result with error
export const mockGqlShadowWithError: ShadowResult = {
    ...mockGqlShadowResult,
    id: 'shadow-003',
    response: null,
    error: {
        __typename: 'ShadowError',
        type: 'rate_limit_error',
        code: '429',
        message: 'Rate limit exceeded',
    },
    divergences: null,
    hasStructuralDivergence: false,
};
