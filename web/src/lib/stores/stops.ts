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
	loading: boolean;
};

const INITIAL: StopsState = {
	stops: [],
	road: null,
	reason: null,
	lastError: null,
	loading: false
};

function activeFilterKeys(s: Set<FilterKey>): FilterKey[] {
	return ALL_FILTERS.filter((k) => s.has(k));
}

function pollIntervalMS(state: GeoState): number {
	if (state.status !== 'live') return 60_000;
	const v = kmh(state.speed);
	return v > 20 ? 15_000 : 60_000;
}

function createStopsStore() {
	const inner = writable<StopsState>(INITIAL);

	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let inflight: AbortController | null = null;
	let geoUnsub: (() => void) | null = null;
	let filtersUnsub: (() => void) | null = null;
	let running = false;

	async function refresh(state: GeoState) {
		if (state.status !== 'live') return;
		inflight?.abort();
		inflight = new AbortController();
		inner.update((s) => ({ ...s, loading: true, lastError: null }));
		try {
			const speedKMH = kmh(state.speed);
			const res = await fetchUpcoming({
				lat: state.lat,
				lon: state.lon,
				heading: state.heading ?? 0,
				speed: speedKMH,
				filters: activeFilterKeys(get(filters)),
				limit: 10,
				signal: inflight.signal
			});
			inner.set({
				stops: res.stops,
				road: res.road ?? null,
				reason: res.reason ?? null,
				lastError: null,
				loading: false
			});
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') return;
			const msg = err instanceof ApiError ? err.message : 'Network error';
			inner.update((s) => ({ ...s, lastError: msg, loading: false }));
		}
	}

	function schedulePoll(state: GeoState) {
		if (pollTimer) clearInterval(pollTimer);
		if (state.status === 'live') {
			void refresh(state);
			pollTimer = setInterval(() => void refresh(state), pollIntervalMS(state));
		}
	}

	function start() {
		if (running) return;
		running = true;

		geoUnsub = geo.subscribe((s) => schedulePoll(s));
		// Skip the immediate subscription call — geo already triggered the first fetch.
		let filterInit = true;
		filtersUnsub = filters.subscribe(() => {
			if (filterInit) {
				filterInit = false;
				return;
			}
			const cur = get(geo);
			void refresh(cur);
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
	}

	return {
		subscribe: inner.subscribe,
		start,
		stop
	};
}

export const stopsPoller = createStopsStore();
