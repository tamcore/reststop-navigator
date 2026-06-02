import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import demoTrip from '$lib/data/demo-trip.json';

type WatchCallback = (pos: GeolocationPosition) => void;
type ErrorCallback = (err: GeolocationPositionError) => void;

const firstPt = demoTrip[0];

class FakeGeolocation {
	watchCount = 0;
	clearedIds: number[] = [];
	currentSuccess?: WatchCallback;
	currentError?: ErrorCallback;

	watchPosition(success: WatchCallback, error?: ErrorCallback): number {
		this.watchCount += 1;
		this.currentSuccess = success;
		this.currentError = error;
		return this.watchCount;
	}

	clearWatch(id: number) {
		this.clearedIds.push(id);
	}

	emit(coords: Partial<GeolocationCoordinates> & { latitude: number; longitude: number }) {
		this.currentSuccess?.({
			coords: {
				accuracy: 5,
				altitude: null,
				altitudeAccuracy: null,
				heading: null,
				speed: null,
				...coords
			} as GeolocationCoordinates,
			timestamp: Date.now()
		} as GeolocationPosition);
	}

	error(code: number) {
		const err = {
			code,
			message: 'fake',
			PERMISSION_DENIED: 1,
			POSITION_UNAVAILABLE: 2,
			TIMEOUT: 3
		} as GeolocationPositionError;
		this.currentError?.(err);
	}
}

describe('geo store', () => {
	let fake: FakeGeolocation;
	let originalNav: Navigator | undefined;

	beforeEach(() => {
		vi.resetModules();
		vi.useFakeTimers();
		fake = new FakeGeolocation();
		originalNav = globalThis.navigator;
		Object.defineProperty(globalThis, 'navigator', {
			value: { geolocation: fake },
			configurable: true,
			writable: true
		});
	});

	afterEach(() => {
		vi.useRealTimers();
		if (originalNav) {
			Object.defineProperty(globalThis, 'navigator', {
				value: originalNav,
				configurable: true,
				writable: true
			});
		}
	});

	it('starts in idle, transitions to pending then live', async () => {
		const { geo } = await import('./geo');
		expect(get(geo).status).toBe('idle');
		geo.start();
		expect(get(geo).status).toBe('pending');

		fake.emit({ latitude: 48.13, longitude: 11.58, heading: 90, speed: 33 });
		const s = get(geo);
		expect(s.status).toBe('live');
		if (s.status === 'live') {
			expect(s.lat).toBe(48.13);
			expect(s.lon).toBe(11.58);
			expect(s.heading).toBe(90);
			expect(s.speed).toBe(33);
		}
	});

	it('handles permission denied', async () => {
		const { geo } = await import('./geo');
		geo.start();
		fake.error(1);
		expect(get(geo).status).toBe('permission-denied');
	});

	it('stop clears the watch', async () => {
		const { geo } = await import('./geo');
		geo.start();
		geo.stop();
		expect(fake.clearedIds.length).toBe(1);
	});

	it('start is idempotent', async () => {
		const { geo } = await import('./geo');
		geo.start();
		geo.start();
		expect(fake.watchCount).toBe(1);
	});

	it('kmh converts m/s to km/h, clamping null/negative to 0', async () => {
		const { kmh } = await import('./geo');
		expect(kmh(null)).toBe(0);
		expect(kmh(-1)).toBe(0);
		expect(kmh(10)).toBeCloseTo(36);
	});

	it('handles non-permission geolocation error as unavailable', async () => {
		const { geo } = await import('./geo');
		geo.start();
		fake.error(2); // POSITION_UNAVAILABLE
		expect(get(geo).status).toBe('unavailable');
	});

	it('startDemo replays trip points starting with the first', async () => {
		const { geo } = await import('./geo');
		geo.startDemo();
		// First point has delay_ms=0, so it fires at setTimeout(fn, 0)
		await vi.advanceTimersByTimeAsync(0);
		const s = get(geo);
		expect(s.status).toBe('live');
		if (s.status === 'live') {
			expect(s.lat).toBe(firstPt.lat);
			expect(s.lon).toBe(firstPt.lon);
			expect(s.heading).toBe(firstPt.heading);
			expect(s.speed).toBeCloseTo(firstPt.speed / 3.6, 1);
			expect(s.accuracy).toBe(firstPt.accuracy);
		}
	});

	it('startDemo advances to second point after its delay', async () => {
		const { geo } = await import('./geo');
		geo.startDemo();
		const secondPt = demoTrip[1];
		// Advance past first point (0ms) and second point (delay_ms)
		await vi.advanceTimersByTimeAsync(secondPt.delay_ms + 1);
		const s = get(geo);
		expect(s.status).toBe('live');
		if (s.status === 'live') {
			expect(s.lat).toBe(secondPt.lat);
			expect(s.lon).toBe(secondPt.lon);
		}
	});

	it('startDemo clears an active real watch', async () => {
		const { geo } = await import('./geo');
		geo.start();
		expect(fake.clearedIds.length).toBe(0);
		geo.startDemo();
		expect(fake.clearedIds.length).toBe(1);
	});

	it('start() resumes real watchPosition after startDemo()', async () => {
		const { geo } = await import('./geo');
		geo.startDemo();
		expect(fake.watchCount).toBe(0);
		geo.start();
		expect(fake.watchCount).toBe(1);
	});
});

describe('geo store round-robin demo tracks', () => {
	beforeEach(() => {
		vi.resetModules();
		vi.useFakeTimers();
		Object.defineProperty(globalThis, 'navigator', {
			value: {},
			configurable: true,
			writable: true
		});
		localStorage.removeItem('reststop:demo-track-index');
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('uses fallback A3 when no glob tracks exist', async () => {
		const { geo, demoTrackInfo } = await import('./geo');
		geo.startDemo();
		await vi.advanceTimersByTimeAsync(0);
		const info = get(demoTrackInfo);
		expect(info.label).toBe('A3');
		const s = get(geo);
		expect(s.status).toBe('live');
		if (s.status === 'live') {
			expect(s.lat).toBe(firstPt.lat);
		}
	});
});

describe('geo store without navigator.geolocation', () => {
	beforeEach(() => {
		vi.resetModules();
		vi.useFakeTimers();
		Object.defineProperty(globalThis, 'navigator', {
			value: {},
			configurable: true,
			writable: true
		});
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('reports unavailable', async () => {
		const { geo } = await import('./geo');
		geo.start();
		expect(get(geo).status).toBe('unavailable');
	});

	it('startDemo still emits live state without geolocation', async () => {
		const { geo } = await import('./geo');
		geo.startDemo();
		await vi.advanceTimersByTimeAsync(0);
		const s = get(geo);
		expect(s.status).toBe('live');
		if (s.status === 'live') {
			expect(s.lat).toBe(firstPt.lat);
		}
	});
});
