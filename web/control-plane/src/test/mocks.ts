import type { Stats, Overview, InteractionSummary, InteractionDetail } from '../types';

export const mockStats: Stats = {
  uptime: '2h30m15s',
  go_version: 'go1.25.3',
  num_goroutine: 42,
  memory: {
    alloc: 15728640,
    total_alloc: 31457280,
    sys: 67108864,
    num_gc: 25,
  },
};

export const mockOverview: Overview = {
  mode: 'single-tenant',
  storage: {
    enabled: true,
    type: 'sqlite',
    path: '/data/gateway.db',
  },
  apps: [
    {
      name: 'main-app',
      frontdoor: 'openai',
      path: '/v1',
      provider: 'openai-provider',
      default_model: 'gpt-4',
      enable_responses: true,
      model_routing: {
        prefix_providers: { 'claude-': 'anthropic-provider' },
        rewrites: [
          { model_exact: 'gpt-3.5-turbo', provider: 'openai-provider', model: 'gpt-4o-mini' },
        ],
      },
    },
    {
      name: 'anthropic-app',
      frontdoor: 'anthropic',
      path: '/anthropic',
      provider: 'anthropic-provider',
      enable_responses: false,
    },
  ],
  frontdoors: [
    { type: 'openai', path: '/v1', provider: 'openai-provider', default_model: 'gpt-4' },
    { type: 'anthropic', path: '/anthropic', provider: 'anthropic-provider' },
  ],
  providers: [
    {
      name: 'openai-provider',
      type: 'openai',
      base_url: 'https://api.openai.com/v1',
      supports_responses: true,
      enable_passthrough: false,
    },
    {
      name: 'anthropic-provider',
      type: 'anthropic',
      base_url: 'https://api.anthropic.com',
      supports_responses: true,
      enable_passthrough: true,
    },
  ],
  routing: {
    default_provider: 'openai-provider',
    rules: [
      { model_prefix: 'claude-', provider: 'anthropic-provider' },
      { model_exact: 'gpt-4-turbo', provider: 'openai-provider' },
    ],
  },
  tenants: [],
};

export const mockMultiTenantOverview: Overview = {
  ...mockOverview,
  mode: 'multi-tenant',
  tenants: [
    { id: 'tenant-1', name: 'Acme Corp', provider_count: 2, routing_rules: 3, supports_tenant: true },
    { id: 'tenant-2', name: 'Beta Inc', provider_count: 1, routing_rules: 1, supports_tenant: true },
  ],
};

export const mockEmptyOverview: Overview = {
  mode: 'single-tenant',
  storage: { enabled: false, type: '' },
  apps: [],
  frontdoors: [],
  providers: [],
  routing: { default_provider: '', rules: [] },
  tenants: [],
};

// Simulates null arrays as returned by Go backend
export const mockNullArraysOverview: Overview = {
  mode: 'single-tenant',
  storage: { enabled: false, type: '' },
  apps: null as unknown as Overview['apps'],
  frontdoors: null as unknown as Overview['frontdoors'],
  providers: null as unknown as Overview['providers'],
  routing: {
    default_provider: '',
    rules: null as unknown as Overview['routing']['rules']
  },
  tenants: null as unknown as Overview['tenants'],
};

export const mockInteractions: InteractionSummary[] = [
  {
    id: 'conv-123',
    type: 'interaction',
    model: 'gpt-4',
    metadata: { title: 'Test Conversation' },
    message_count: 5,
    created_at: 1700000000,
    updated_at: 1700001000,
  },
  {
    id: 'resp-456',
    type: 'interaction',
    status: 'completed',
    model: 'claude-3-opus',
    metadata: {},
    created_at: 1700000500,
    updated_at: 1700000600,
  },
  {
    id: 'resp-789',
    type: 'interaction',
    status: 'in_progress',
    model: 'gpt-4-turbo',
    previous_response_id: 'resp-456',
    created_at: 1700000700,
    updated_at: 1700000800,
  },
];

export const mockConversationDetail: InteractionDetail = {
  id: 'conv-123',
  type: 'conversation',
  model: 'gpt-4',
  metadata: { title: 'Test Conversation' },
  created_at: 1700000000,
  updated_at: 1700001000,
  messages: [
    { id: 'msg-1', role: 'user', content: 'Hello!', created_at: 1700000000 },
    { id: 'msg-2', role: 'assistant', content: 'Hi there! How can I help?', created_at: 1700000100 },
    { id: 'msg-3', role: 'user', content: 'What is 2+2?', created_at: 1700000200 },
    { id: 'msg-4', role: 'assistant', content: '2+2 equals 4.', created_at: 1700000300 },
  ],
};

export const mockResponseDetail: InteractionDetail = {
  id: 'resp-456',
  type: 'response',
  status: 'completed',
  model: 'claude-3-opus',
  metadata: {},
  created_at: 1700000500,
  updated_at: 1700000600,
  request: { model: 'claude-3-opus', input: 'Test input' },
  response: { id: 'resp-456', output: [{ type: 'message', content: [{ type: 'text', text: 'Test output' }] }] },
};

export function createMockFetch(responses: Record<string, unknown>) {
  return (url: string) => {
    // Normalize the URL by removing the base and any query params for matching
    const path = url.replace('/admin/api', '');

    // Try exact match first
    let response = responses[path];

    // Try without query params
    if (response === undefined) {
      const pathWithoutQuery = path.split('?')[0];
      response = responses[pathWithoutQuery];
    }

    // Try with query params if exact path didn't match
    if (response === undefined) {
      for (const key of Object.keys(responses)) {
        if (path.startsWith(key.split('?')[0])) {
          response = responses[key];
          break;
        }
      }
    }

    if (response === undefined) {
      return Promise.resolve({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Not found' }),
      });
    }

    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(response),
    });
  };
}
