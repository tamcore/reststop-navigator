import { writable } from 'svelte/store';

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
	}

	return { subscribe: inner.subscribe, start, stop };
}

export const geo = createGeoStore();

// kmh converts a Geolocation-API speed (m/s, possibly null) to km/h, or 0.
export function kmh(speedMS: number | null): number {
	if (speedMS === null || !Number.isFinite(speedMS) || speedMS < 0) return 0;
	return speedMS * 3.6;
}
