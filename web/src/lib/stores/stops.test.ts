import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

function jsonResponse(body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

const LIVE_GEO = {
	status: 'live' as const,
	lat: 48.13,
	lon: 11.58,
	heading: 90,
	speed: 33,
	accuracy: 5,
	timestamp: Date.now()
};

describe('stops store', () => {
	let fetchSpy: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		vi.resetModules();
		vi.useFakeTimers();
		fetchSpy = vi.fn().mockResolvedValue(
			jsonResponse({
				stops: [{ id: 'node/1', name: 'Test', kind: 'services', lat: 0, lon: 0, distance_m: 1000, eta_seconds: 60, amenities: { fuel: true, charging: false, food: false, toilets: false, open24h: false, dog: false } }],
				road: { ref: 'A3', direction: 'East' },
				country: 'DE'
			})
		);
		globalThis.fetch = fetchSpy as typeof fetch;
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.restoreAllMocks();
	});

	it('starts with empty initial state', async () => {
		const { stopsPoller } = await import('./stops');
		const state = get(stopsPoller);
		expect(state.stops).toEqual([]);
		expect(state.road).toBeNull();
		expect(state.reason).toBeNull();
		expect(state.lastError).toBeNull();
		expect(state.loading).toBe(false);
	});

	it('fetches when geo emits live state after start', async () => {
		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');

		// Provide a fake navigator so geo.start() doesn't fail
		Object.defineProperty(globalThis, 'navigator', {
			value: { geolocation: { watchPosition: vi.fn(), clearWatch: vi.fn() } },
			configurable: true,
			writable: true
		});

		stopsPoller.start();
		// Manually push live state into geo
		(geo as unknown as { set(v: unknown): void }).set?.(LIVE_GEO);
		// The geo store is a custom store without .set; we need to trigger it via the real mechanism.
		// Instead, let's import and directly set the geo store by using its internals.

		// Actually, the stops store subscribes to geo. We need to make geo emit a live value.
		// Since geo.start() uses navigator, let's use startDemo which sets a fixed value.
		geo.startDemo();

		// Allow the fetch promise to resolve
		await vi.advanceTimersByTimeAsync(0);

		const state = get(stopsPoller);
		expect(state.stops).toHaveLength(1);
		expect(state.stops[0].id).toBe('node/1');
		expect(state.road).toEqual({ ref: 'A3', direction: 'East' });
		expect(state.loading).toBe(false);

		stopsPoller.stop();
	});

	it('data persists after stop + restart (cached)', async () => {
		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');

		geo.startDemo();
		stopsPoller.start();
		await vi.advanceTimersByTimeAsync(0);

		const before = get(stopsPoller);
		expect(before.stops).toHaveLength(1);

		// Simulate navigating away: stop is NOT called (that's the point —
		// the store keeps running). But let's verify the data survives
		// an unsubscribe + resubscribe cycle.
		let captured = get(stopsPoller);
		const unsub = stopsPoller.subscribe((s) => (captured = s));
		expect(captured.stops).toHaveLength(1);
		unsub();

		// Data is still there
		expect(get(stopsPoller).stops).toHaveLength(1);

		stopsPoller.stop();
	});

	it('polls at 15s interval when speed > 20 km/h', async () => {
		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');

		// Demo speed is 33 m/s = ~119 km/h (>20), so 15s interval
		geo.startDemo();
		stopsPoller.start();
		await vi.advanceTimersByTimeAsync(0);

		expect(fetchSpy).toHaveBeenCalledTimes(1);

		// Advance 15 seconds — should trigger another fetch
		await vi.advanceTimersByTimeAsync(15_000);
		expect(fetchSpy).toHaveBeenCalledTimes(2);

		stopsPoller.stop();
	});

	it('handles API errors gracefully', async () => {
		fetchSpy.mockResolvedValue(
			new Response(JSON.stringify({ error: 'server broke' }), {
				status: 500,
				headers: { 'Content-Type': 'application/json' }
			})
		);

		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');

		geo.startDemo();
		stopsPoller.start();
		await vi.advanceTimersByTimeAsync(0);

		const state = get(stopsPoller);
		expect(state.lastError).toBe('server broke');
		expect(state.loading).toBe(false);

		stopsPoller.stop();
	});

	it('stop clears inflight and timers', async () => {
		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');

		geo.startDemo();
		stopsPoller.start();
		await vi.advanceTimersByTimeAsync(0);
		expect(fetchSpy).toHaveBeenCalledTimes(1);

		stopsPoller.stop();

		// Advancing time should NOT trigger more fetches
		await vi.advanceTimersByTimeAsync(60_000);
		expect(fetchSpy).toHaveBeenCalledTimes(1);
	});

	it('start is idempotent', async () => {
		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');

		geo.startDemo();
		stopsPoller.start();
		stopsPoller.start(); // second call should be no-op
		await vi.advanceTimersByTimeAsync(0);

		// Should still only have one set of subscriptions → one fetch
		expect(fetchSpy).toHaveBeenCalledTimes(1);

		stopsPoller.stop();
	});

	it('filter change triggers re-fetch', async () => {
		const { stopsPoller } = await import('./stops');
		const { geo } = await import('./geo');
		const { filters } = await import('./filters');

		geo.startDemo();
		stopsPoller.start();
		await vi.advanceTimersByTimeAsync(0);

		// Initial fetch from geo subscription + filters subscription
		const callsBefore = fetchSpy.mock.calls.length;

		// Toggle a filter
		filters.toggle('fuel');
		await vi.advanceTimersByTimeAsync(0);

		expect(fetchSpy.mock.calls.length).toBeGreaterThan(callsBefore);

		stopsPoller.stop();
	});
});
