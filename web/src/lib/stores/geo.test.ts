import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

type WatchCallback = (pos: GeolocationPosition) => void;
type ErrorCallback = (err: GeolocationPositionError) => void;

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
		fake = new FakeGeolocation();
		originalNav = globalThis.navigator;
		Object.defineProperty(globalThis, 'navigator', {
			value: { geolocation: fake },
			configurable: true,
			writable: true
		});
	});

	afterEach(() => {
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
});

describe('geo store without navigator.geolocation', () => {
	beforeEach(() => {
		vi.resetModules();
		Object.defineProperty(globalThis, 'navigator', {
			value: {},
			configurable: true,
			writable: true
		});
	});

	it('reports unavailable', async () => {
		const { geo } = await import('./geo');
		geo.start();
		expect(get(geo).status).toBe('unavailable');
	});
});
