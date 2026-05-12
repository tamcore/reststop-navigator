import { writable } from 'svelte/store';
import demoTrip from '$lib/data/demo-trip.json';

export type DemoTripPoint = {
	delay_ms: number;
	lat: number;
	lon: number;
	heading: number;
	speed: number; // km/h (as sent to the API)
	accuracy: number;
};

export type GeoState =
	| { status: 'idle' }
	| { status: 'unavailable' }
	| { status: 'permission-denied' }
	| { status: 'pending' }
	| {
			status: 'live';
			lat: number;
			lon: number;
			heading: number | null;
			speed: number | null; // m/s, per Geolocation API
			accuracy: number;
			timestamp: number;
	  };

function createGeoStore() {
	const inner = writable<GeoState>({ status: 'idle' });
	let watchId: number | null = null;
	let replayTimers: ReturnType<typeof setTimeout>[] = [];

	function start() {
		if (typeof navigator === 'undefined' || !('geolocation' in navigator)) {
			inner.set({ status: 'unavailable' });
			return;
		}
		if (watchId !== null) return;
		inner.set({ status: 'pending' });
		watchId = navigator.geolocation.watchPosition(
			(pos) => {
				inner.set({
					status: 'live',
					lat: pos.coords.latitude,
					lon: pos.coords.longitude,
					heading: Number.isFinite(pos.coords.heading) ? (pos.coords.heading as number) : null,
					speed: Number.isFinite(pos.coords.speed) ? (pos.coords.speed as number) : null,
					accuracy: pos.coords.accuracy,
					timestamp: pos.timestamp
				});
			},
			(err) => {
				if (err.code === err.PERMISSION_DENIED) {
					inner.set({ status: 'permission-denied' });
				} else {
					inner.set({ status: 'unavailable' });
				}
			},
			{ enableHighAccuracy: true, maximumAge: 5000, timeout: 30000 }
		);
	}

	function stop() {
		if (watchId !== null && typeof navigator !== 'undefined' && 'geolocation' in navigator) {
			navigator.geolocation.clearWatch(watchId);
		}
		watchId = null;
		stopReplay();
	}

	function stopReplay() {
		for (const t of replayTimers) clearTimeout(t);
		replayTimers = [];
	}

	function startDemo() {
		if (watchId !== null && typeof navigator !== 'undefined' && 'geolocation' in navigator) {
			navigator.geolocation.clearWatch(watchId);
		}
		watchId = null;
		stopReplay();

		const points: DemoTripPoint[] = demoTrip as DemoTripPoint[];
		let elapsed = 0;
		for (let i = 0; i < points.length; i++) {
			elapsed += points[i].delay_ms;
			const pt = points[i];
			const timer = setTimeout(() => {
				inner.set({
					status: 'live',
					lat: pt.lat,
					lon: pt.lon,
					heading: pt.heading,
					// Speed in the trip data is km/h (as logged by the API).
					// The Geolocation API uses m/s, so convert back.
					speed: pt.speed / 3.6,
					accuracy: pt.accuracy,
					timestamp: Date.now()
				});
			}, elapsed);
			replayTimers.push(timer);
		}

		// After the last point, loop from the beginning.
		const loopTimer = setTimeout(() => startDemo(), elapsed + 3000);
		replayTimers.push(loopTimer);
	}

	return { subscribe: inner.subscribe, start, stop, startDemo };
}

export const geo = createGeoStore();

/** Total duration of the demo trip in milliseconds. */
export const DEMO_TRIP_DURATION_MS = (demoTrip as DemoTripPoint[]).reduce(
	(sum, p) => sum + p.delay_ms,
	0
);

/** Number of data points in the demo trip. */
export const DEMO_TRIP_POINTS = (demoTrip as DemoTripPoint[]).length;

// kmh converts a Geolocation-API speed (m/s, possibly null) to km/h, or 0.
export function kmh(speedMS: number | null): number {
	if (speedMS === null || !Number.isFinite(speedMS) || speedMS < 0) return 0;
	return speedMS * 3.6;
}
