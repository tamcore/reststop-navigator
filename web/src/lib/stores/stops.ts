import { writable, get } from 'svelte/store';
import { fetchUpcoming, ApiError } from '$lib/api/client';
import { filters } from '$lib/stores/filters';
import { geo, kmh, type GeoState } from '$lib/stores/geo';
import { ALL_FILTERS, type FilterKey, type StopInfo, type Road } from '$lib/types/api';

export type StopsState = {
	stops: StopInfo[];
	road: Road | null;
	reason: string | null;
	lastError: string | null;
	errorCount: number;
	loading: boolean;
};

const INITIAL: StopsState = {
	stops: [],
	road: null,
	reason: null,
	lastError: null,
	errorCount: 0,
	loading: false
};

function activeFilterKeys(s: Set<FilterKey>): FilterKey[] {
	return ALL_FILTERS.filter((k) => s.has(k));
}

const BURST_INTERVAL_MS = 3_000;
const BURST_WINDOW_MS = 30_000;
const SPEED_THRESHOLD_KMH = 20;

function normalIntervalMS(speedKMH: number): number {
	return speedKMH > SPEED_THRESHOLD_KMH ? 15_000 : 60_000;
}

function speedBucket(state: GeoState): 'fast' | 'slow' {
	if (state.status !== 'live') return 'slow';
	return kmh(state.speed) > SPEED_THRESHOLD_KMH ? 'fast' : 'slow';
}

function createStopsStore() {
	const inner = writable<StopsState>(INITIAL);

	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let inflight: AbortController | null = null;
	let geoUnsub: (() => void) | null = null;
	let filtersUnsub: (() => void) | null = null;
	let running = false;

	// Latest geo state — read by refresh() so the poll timer always uses
	// the most recent position without needing to restart on every update.
	let latestGeo: GeoState = { status: 'idle' };
	let lastSpeedBucket: 'fast' | 'slow' = 'slow';

	// Burst-mode tracking
	let liveStartTime: number | null = null;
	let burstActive = false;

	function isBurstPhase(): boolean {
		if (!burstActive || liveStartTime === null) return false;
		return Date.now() - liveStartTime < BURST_WINDOW_MS;
	}

	function pollIntervalMS(): number {
		if (isBurstPhase()) return BURST_INTERVAL_MS;
		const v = latestGeo.status === 'live' ? kmh(latestGeo.speed) : 0;
		return normalIntervalMS(v);
	}

	async function refresh() {
		const state = latestGeo;
		if (state.status !== 'live') return;
		// Let any in-flight request complete — aborting it while the backend
		// is fetching a cold Overpass tile would waste that work.
		if (inflight) return;
		inflight = new AbortController();
		inner.update((s) => ({ ...s, loading: true, lastError: null }));
		try {
			const speedKMH = kmh(state.speed);
			const res = await fetchUpcoming({
				lat: state.lat,
				lon: state.lon,
				heading: state.heading ?? 0,
				speed: speedKMH,
				accuracy: state.accuracy,
				filters: activeFilterKeys(get(filters)),
				limit: 10,
				signal: inflight.signal
			});
			inflight = null;
			inner.set({
				stops: res.stops,
				road: res.road ?? null,
				reason: res.reason ?? null,
				lastError: null,
				errorCount: 0,
				loading: false
			});

			// Exit burst mode on successful highway match
			if (res.road && burstActive) {
				burstActive = false;
				rescheduleTimer();
			}
		} catch (err) {
			inflight = null;
			if (err instanceof DOMException && err.name === 'AbortError') return;
			const msg = err instanceof ApiError ? err.message : 'Network error';
			inner.update((s) => ({ ...s, lastError: msg, errorCount: s.errorCount + 1, loading: false }));
		}
	}

	// rescheduleTimer resets the poll timer without an immediate refresh.
	function rescheduleTimer() {
		if (pollTimer) clearInterval(pollTimer);
		if (latestGeo.status === 'live') {
			const interval = pollIntervalMS();
			pollTimer = setInterval(() => void refresh(), interval);
		}
	}

	function schedulePoll() {
		if (pollTimer) clearInterval(pollTimer);
		if (latestGeo.status === 'live') {
			void refresh();
			const interval = pollIntervalMS();
			pollTimer = setInterval(() => {
				// Check if burst window expired mid-interval
				if (burstActive && !isBurstPhase()) {
					burstActive = false;
					rescheduleTimer();
					return;
				}
				void refresh();
			}, interval);
		}
	}

	function start() {
		if (running) return;
		running = true;

		geoUnsub = geo.subscribe((s) => {
			const wasLive = latestGeo.status === 'live';
			const prevBucket = lastSpeedBucket;
			latestGeo = s;
			lastSpeedBucket = speedBucket(s);

			if (s.status === 'live' && !wasLive) {
				// Transition to live → start burst mode and begin polling.
				liveStartTime = Date.now();
				burstActive = true;
				schedulePoll();
				return;
			}

			// Adjust poll interval when speed crosses the threshold.
			if (wasLive && lastSpeedBucket !== prevBucket) {
				rescheduleTimer();
			}
		});
		// Skip the immediate subscription call — geo already triggered the first fetch.
		let filterInit = true;
		filtersUnsub = filters.subscribe(() => {
			if (filterInit) {
				filterInit = false;
				return;
			}
			// Abort inflight so the filter change is picked up immediately.
			inflight?.abort();
			inflight = null;
			void refresh();
		});
	}

	function stop() {
		if (!running) return;
		running = false;
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
		inflight?.abort();
		inflight = null;
		geoUnsub?.();
		filtersUnsub?.();
		geoUnsub = null;
		filtersUnsub = null;
		latestGeo = { status: 'idle' };
		lastSpeedBucket = 'slow';
		liveStartTime = null;
		burstActive = false;
	}

	return {
		subscribe: inner.subscribe,
		start,
		stop
	};
}

export const stopsPoller = createStopsStore();
