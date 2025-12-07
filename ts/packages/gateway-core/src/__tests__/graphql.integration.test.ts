/**
 * Integration tests for the GraphQL handler.
 *
 * These tests verify the GraphQL API endpoints.
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { GraphQLHandler } from '../graphql/handler.js';

describe('GraphQLHandler Integration', () => {
    let handler: GraphQLHandler;
    const startTime = new Date(Date.now() - 120000); // 2 minutes ago

    beforeEach(() => {
        handler = new GraphQLHandler({ startTime });
    });

    describe('matches()', () => {
        it('should match /graphql paths', () => {
            expect(handler.matches('/graphql')).toBe(true);
            expect(handler.matches('/api/graphql')).toBe(true);
            expect(handler.matches('/admin/graphql')).toBe(true);
        });

        it('should match playground paths', () => {
            expect(handler.matches('/playground')).toBe(true);
            expect(handler.matches('/graphiql')).toBe(true);
        });

        it('should not match other paths', () => {
            expect(handler.matches('/api/stats')).toBe(false);
            expect(handler.matches('/v1/chat/completions')).toBe(false);
        });
    });

    describe('POST /graphql - stats query', () => {
        it('should return stats', async () => {
            const request = new Request('http://localhost/graphql', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    query: '{ stats { uptime runtime } }',
                }),
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json() as { data?: { stats?: { uptime?: string; runtime?: string } } };
            expect(body.data).toBeDefined();
            expect(body.data?.stats).toBeDefined();
            expect(body.data?.stats?.uptime).toBeDefined();
            expect(body.data?.stats?.runtime).toBeDefined();
        });
    });

    describe('POST /graphql - overview query', () => {
        it('should return overview', async () => {
            const request = new Request('http://localhost/graphql', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    query: '{ overview { mode storage { enabled type } } }',
                }),
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json() as { data?: { overview?: { mode?: string; storage?: { enabled?: boolean } } } };
            expect(body.data).toBeDefined();
            expect(body.data?.overview).toBeDefined();
            expect(body.data?.overview?.mode).toBe('single-tenant');
        });
    });

    describe('POST /graphql - interactions query', () => {
        it('should return empty interactions list', async () => {
            const request = new Request('http://localhost/graphql', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    query: '{ interactions { interactions { id } total } }',
                }),
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json() as { data?: { interactions?: { interactions?: unknown[]; total?: number } } };
            expect(body.data).toBeDefined();
            expect(body.data?.interactions).toBeDefined();
            expect(body.data?.interactions?.interactions).toEqual([]);
            expect(body.data?.interactions?.total).toBe(0);
        });
    });

    describe('GET /graphql with query param', () => {
        it('should execute query from URL param', async () => {
            const query = encodeURIComponent('{ stats { uptime } }');
            const request = new Request(`http://localhost/graphql?query=${query}`, {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json() as { data?: { stats?: { uptime?: string } } };
            expect(body.data).toBeDefined();
            expect(body.data?.stats).toBeDefined();
        });
    });

    describe('GET /playground', () => {
        it('should serve GraphiQL playground', async () => {
            const request = new Request('http://localhost/graphql/playground', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);
            expect(response.headers.get('Content-Type')).toBe('text/html');

            const html = await response.text();
            expect(html).toContain('GraphiQL');
            expect(html).toContain('graphql');
        });
    });

    describe('error handling', () => {
        it('should return error for invalid query', async () => {
            const request = new Request('http://localhost/graphql', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    query: 'invalid query syntax',
                }),
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json() as { errors?: Array<{ message: string }> };
            expect(body.errors).toBeDefined();
            expect(body.errors?.length).toBeGreaterThan(0);
        });

        it('should return error for unknown field', async () => {
            const request = new Request('http://localhost/graphql', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    query: '{ unknownField { id } }',
                }),
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json() as { errors?: Array<{ message: string }> };
            expect(body.errors).toBeDefined();
        });
    });
});
