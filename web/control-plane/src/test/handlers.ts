import { http, HttpResponse } from 'msw';
import {
  mockStats,
  mockOverview,
  mockInteractions,
  mockConversationDetail,
  mockResponseDetail,
} from './mocks';

// Default handlers used across tests. Individual tests can override by using server.use.
export const handlers = [
  http.get('/admin/api/stats', () => HttpResponse.json(mockStats)),
  http.get('/admin/api/overview', () => HttpResponse.json(mockOverview)),
  http.get('/admin/api/interactions', () =>
    HttpResponse.json({ interactions: mockInteractions, total: mockInteractions.length }),
  ),
  http.get('/admin/api/interactions/:id', ({ params }) => {
    const { id } = params as { id: string };
    if (id === 'conv-123') return HttpResponse.json(mockConversationDetail);
    if (id === 'resp-456') return HttpResponse.json(mockResponseDetail);
    return HttpResponse.json(
      {
        id,
        type: 'interaction',
        status: 'completed',
        model: 'gpt-4',
        created_at: 1700000000,
        updated_at: 1700000000,
      },
      { status: 200 },
    );
  }),
  http.get('/admin/api/interactions/:id/events', ({ params }) => {
    const { id } = params as { id: string };
    return HttpResponse.json({ interaction_id: id, events: [] });
  }),
];
