/**
 * Integration tests for the Admin handler.
 *
 * These tests verify the control plane REST API endpoints.
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { AdminHandler } from '../admin/handler.js';

describe('AdminHandler Integration', () => {
    let handler: AdminHandler;
    const startTime = new Date(Date.now() - 60000); // 1 minute ago

    beforeEach(() => {
        handler = new AdminHandler({ startTime });
    });

    describe('matches()', () => {
        it('should match /api/ paths', () => {
            expect(handler.matches('/api/stats')).toBe(true);
            expect(handler.matches('/api/overview')).toBe(true);
            expect(handler.matches('/api/interactions')).toBe(true);
        });

        it('should match /health', () => {
            expect(handler.matches('/health')).toBe(true);
        });

        it('should not match other paths', () => {
            expect(handler.matches('/v1/chat/completions')).toBe(false);
            expect(handler.matches('/anthropic/messages')).toBe(false);
        });
    });

    describe('GET /api/stats', () => {
        it('should return stats', async () => {
            const request = new Request('http://localhost/api/stats', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json();
            expect(body).toHaveProperty('uptime');
            expect(body).toHaveProperty('uptimeMs');
            expect(body).toHaveProperty('runtime');
            expect(body.uptimeMs).toBeGreaterThan(0);
        });
    });

    describe('GET /api/overview', () => {
        it('should return overview', async () => {
            const request = new Request('http://localhost/api/overview', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json();
            expect(body).toHaveProperty('mode');
            expect(body).toHaveProperty('storage');
            expect(body).toHaveProperty('apps');
            expect(body).toHaveProperty('providers');
            expect(body.mode).toBe('single-tenant');
        });
    });

    describe('GET /api/interactions', () => {
        it('should return empty list when storage not configured', async () => {
            const request = new Request('http://localhost/api/interactions', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            // Returns error when storage not configured
            expect(response.status).toBe(503);
        });
    });

    describe('GET /api/health', () => {
        it('should return ok status', async () => {
            const request = new Request('http://localhost/api/health', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json();
            expect(body).toEqual({ status: 'ok' });
        });

        it('should also respond at /health', async () => {
            const request = new Request('http://localhost/health', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(200);

            const body = await response.json();
            expect(body).toEqual({ status: 'ok' });
        });
    });

    describe('unknown routes', () => {
        it('should return 404 for unknown paths', async () => {
            const request = new Request('http://localhost/api/unknown', {
                method: 'GET',
            });

            const response = await handler.handle(request);
            expect(response.status).toBe(404);
        });
    });
});
