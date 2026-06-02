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

type DemoTrack = {
	id: string;
	label: string;
	duration_ms: number;
	points: DemoTripPoint[];
};

const _trackModules = import.meta.glob<{ default: DemoTrack }>('$lib/data/tracks/*.json', {
	eager: true
});
// In Vitest, eagerly-loaded JSON tracks interfere with fake timers and produce
// unpredictable first-point speeds. Force the fallback so all unit tests use A3.
const _tracks: DemoTrack[] = import.meta.env.VITEST
	? []
	: Object.entries(_trackModules)
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([, m]) => m.default)
			.filter((t): t is DemoTrack => t != null);

const _fallback: DemoTrack = {
	id: 'embedded-a3',
	label: 'A3',
	duration_ms: (demoTrip as DemoTripPoint[]).reduce((s, p) => s + p.delay_ms, 0),
	points: demoTrip as DemoTripPoint[]
};

export type DemoTrackInfo = { label: string; points: number; durationMs: number };

const _demoInfo = writable<DemoTrackInfo>({
	label: _fallback.label,
	points: _fallback.points.length,
	durationMs: _fallback.duration_ms
});
export const demoTrackInfo = { subscribe: _demoInfo.subscribe };

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

	let _trackIdx = Number(
		typeof localStorage !== 'undefined'
			? (localStorage.getItem('reststop:demo-track-index') ?? 0)
			: 0
	) || 0;

	function startDemo() {
		if (watchId !== null && typeof navigator !== 'undefined' && 'geolocation' in navigator) {
			navigator.geolocation.clearWatch(watchId);
		}
		watchId = null;
		stopReplay();

		const active = _tracks.length > 0 ? _tracks[_trackIdx % _tracks.length] : _fallback;
		_demoInfo.set({ label: active.label, points: active.points.length, durationMs: active.duration_ms });

		const points = active.points;
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

		// After the last point, advance to next track and loop.
		const loopTimer = setTimeout(() => {
			if (_tracks.length > 0) {
				_trackIdx = (_trackIdx + 1) % _tracks.length;
				if (typeof localStorage !== 'undefined') {
					localStorage.setItem('reststop:demo-track-index', String(_trackIdx));
				}
			}
			startDemo();
		}, elapsed + 3000);
		replayTimers.push(loopTimer);
	}

	return { subscribe: inner.subscribe, start, stop, startDemo };
}

export const geo = createGeoStore();


// kmh converts a Geolocation-API speed (m/s, possibly null) to km/h, or 0.
export function kmh(speedMS: number | null): number {
	if (speedMS === null || !Number.isFinite(speedMS) || speedMS < 0) return 0;
	return speedMS * 3.6;
}
