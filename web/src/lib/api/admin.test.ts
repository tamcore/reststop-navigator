import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	fetchAdminPositions,
	fetchAdminStats,
	fetchAdminTileStops,
	fetchAdminTiles
} from './admin';

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
	return new Response(JSON.stringify(body), {
		status: 200,
		headers: { 'Content-Type': 'application/json' },
		...init
	});
}

describe('admin api client', () => {
	let fetchSpy: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		fetchSpy = vi.fn();
		globalThis.fetch = fetchSpy as typeof fetch;
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('fetches live positions', async () => {
		fetchSpy.mockResolvedValue(
			jsonResponse({
				clients: [
					{
						client_id: '11111111-2222-4333-8444-555555555555',
						lat: 48.1,
						lon: 11.5,
						heading: 90,
						speed: 100,
						accuracy: 10,
						last_seen: '2026-06-12T10:00:00Z'
					}
				],
				count: 1
			})
		);

		const out = await fetchAdminPositions();
		expect(out.count).toBe(1);
		expect(out.clients[0].lat).toBe(48.1);
		expect(String(fetchSpy.mock.calls[0][0])).toBe('/api/admin/positions');
	});

	it('fetches tiles', async () => {
		fetchSpy.mockResolvedValue(jsonResponse({ tiles: [] }));
		const out = await fetchAdminTiles();
		expect(out.tiles).toEqual([]);
		expect(String(fetchSpy.mock.calls[0][0])).toBe('/api/admin/tiles');
	});

	it('fetches tile stops with coords in query', async () => {
		fetchSpy.mockResolvedValue(jsonResponse({ stops: [] }));
		await fetchAdminTileStops(48.0, 11.5);
		const url = String(fetchSpy.mock.calls[0][0]);
		expect(url).toContain('/api/admin/tiles/stops?');
		expect(url).toContain('south=48');
		expect(url).toContain('west=11.5');
	});

	it('fetches stats', async () => {
		fetchSpy.mockResolvedValue(
			jsonResponse({
				uptime_seconds: 12,
				cache: { hits: 3, misses: 1 },
				presence_count: 0,
				redis: { keys: 5, used_memory_bytes: 1024 }
			})
		);
		const out = await fetchAdminStats();
		expect(out.cache.hits).toBe(3);
		expect(out.redis?.keys).toBe(5);
	});

	it('throws ApiError with status on 401', async () => {
		fetchSpy.mockResolvedValue(
			new Response(JSON.stringify({ error: 'unauthorized' }), {
				status: 401,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		await expect(fetchAdminPositions()).rejects.toMatchObject({
			name: 'ApiError',
			status: 401,
			message: 'unauthorized'
		});
	});
});
