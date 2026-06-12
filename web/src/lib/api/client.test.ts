import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, fetchStopDetail, fetchUpcoming, getClientId } from './client';

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
	return new Response(JSON.stringify(body), {
		status: 200,
		headers: { 'Content-Type': 'application/json' },
		...init
	});
}

describe('fetchUpcoming', () => {
	let fetchSpy: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		fetchSpy = vi.fn();
		globalThis.fetch = fetchSpy as typeof fetch;
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('builds query string from params and parses JSON', async () => {
		fetchSpy.mockResolvedValue(jsonResponse({ stops: [], country: 'DE' }));

		const out = await fetchUpcoming({
			lat: 48.13,
			lon: 11.58,
			heading: 90,
			speed: 120,
			filters: ['fuel', 'charging'],
			limit: 5
		});

		expect(out.country).toBe('DE');

		const url = String(fetchSpy.mock.calls[0][0]);
		expect(url).toContain('/api/stops/upcoming?');
		expect(url).toContain('lat=48.13');
		expect(url).toContain('lon=11.58');
		expect(url).toContain('heading=90');
		expect(url).toContain('speed=120');
		expect(url).toContain('filters=fuel%2Ccharging');
		expect(url).toContain('limit=5');
	});

	it('omits optional params when not given', async () => {
		fetchSpy.mockResolvedValue(jsonResponse({ stops: [] }));
		await fetchUpcoming({ lat: 48, lon: 11 });
		const url = String(fetchSpy.mock.calls[0][0]);
		expect(url).toContain('lat=48');
		expect(url).toContain('lon=11');
		expect(url).not.toContain('heading=');
		expect(url).not.toContain('speed=');
		expect(url).not.toContain('filters=');
		expect(url).not.toContain('limit=');
		expect(url).not.toContain('accuracy=');
	});

	it('sends accuracy param when provided', async () => {
		fetchSpy.mockResolvedValue(jsonResponse({ stops: [] }));
		await fetchUpcoming({ lat: 48, lon: 11, accuracy: 150 });
		const url = String(fetchSpy.mock.calls[0][0]);
		expect(url).toContain('accuracy=150');
	});

	it('throws ApiError for non-2xx with json error body', async () => {
		fetchSpy.mockResolvedValue(
			new Response(JSON.stringify({ error: 'lat is required' }), {
				status: 400,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		await expect(fetchUpcoming({ lat: 0, lon: 0 })).rejects.toMatchObject({
			name: 'ApiError',
			status: 400,
			message: 'lat is required'
		});
	});

	it('throws ApiError with statusText when body is not JSON', async () => {
		fetchSpy.mockResolvedValue(new Response('plain', { status: 500, statusText: 'Boom' }));
		await expect(fetchUpcoming({ lat: 0, lon: 0 })).rejects.toBeInstanceOf(ApiError);
	});
});

describe('getClientId', () => {
	const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

	beforeEach(() => {
		localStorage.clear();
	});

	afterEach(() => {
		localStorage.clear();
	});

	it('generates a UUID and persists it in localStorage', () => {
		const id = getClientId();
		expect(id).toMatch(UUID_RE);
		expect(localStorage.getItem('reststop:client-id')).toBe(id);
	});

	it('returns the same id on repeated calls', () => {
		expect(getClientId()).toBe(getClientId());
	});

	it('reuses an existing stored id', () => {
		localStorage.setItem('reststop:client-id', '11111111-2222-4333-8444-555555555555');
		expect(getClientId()).toBe('11111111-2222-4333-8444-555555555555');
	});
});

describe('fetchUpcoming client id header', () => {
	beforeEach(() => {
		localStorage.clear();
	});

	afterEach(() => {
		vi.restoreAllMocks();
		localStorage.clear();
	});

	it('sends X-Client-Id header', async () => {
		const fetchSpy = vi.fn().mockResolvedValue(jsonResponse({ stops: [] }));
		globalThis.fetch = fetchSpy as typeof fetch;

		await fetchUpcoming({ lat: 48, lon: 11 });

		const init = fetchSpy.mock.calls[0][1] as RequestInit;
		const headers = init.headers as Record<string, string>;
		expect(headers['X-Client-Id']).toBe(getClientId());
	});
});

describe('fetchStopDetail', () => {
	it('encodes id and calls /api/stops/detail', async () => {
		const fetchSpy = vi.fn().mockResolvedValue(
			jsonResponse({
				country: 'DE',
				stop: {
					id: 'node/100',
					kind: 'services',
					lat: 0,
					lon: 0,
					distance_m: 0,
					eta_seconds: 0,
					amenities: { fuel: true, charging: false, food: false, toilets: false, open24h: false, dog: false }
				},
				deep_links: { google: 'g', apple: 'a', waze: 'w' }
			})
		);
		globalThis.fetch = fetchSpy as typeof fetch;

		const out = await fetchStopDetail('node/100', { lat: 48, lon: 11.005 });
		expect(out.stop.id).toBe('node/100');
		const url = String(fetchSpy.mock.calls[0][0]);
		expect(url).toContain('/api/stops/detail?');
		expect(url).toContain('id=node%2F100');
		expect(url).toContain('lat=48');
		expect(url).toContain('lon=11.005');
	});
});
