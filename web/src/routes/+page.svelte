<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { ApiError, fetchUpcoming } from '$lib/api/client';
	import { filters } from '$lib/stores/filters';
	import { geo, kmh, type GeoState } from '$lib/stores/geo';
	import FilterChips from '$lib/components/FilterChips.svelte';
	import StopCard from '$lib/components/StopCard.svelte';
	import { ALL_FILTERS, type FilterKey, type StopInfo } from '$lib/types/api';

	let stops: StopInfo[] = $state([]);
	let road: { ref?: string; name?: string; direction?: string } | null = $state(null);
	let reason = $state<string | null>(null);
	let lastError = $state<string | null>(null);
	let loading = $state(false);

	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let inflight: AbortController | null = null;
	let lastUnsub: (() => void) | null = null;
	let lastFiltersUnsub: (() => void) | null = null;

	function activeFilterKeys(s: Set<FilterKey>): FilterKey[] {
		return ALL_FILTERS.filter((k) => s.has(k));
	}

	async function refresh(state: GeoState) {
		if (state.status !== 'live') return;
		inflight?.abort();
		inflight = new AbortController();
		loading = true;
		lastError = null;
		try {
			const speedKMH = kmh(state.speed);
			const res = await fetchUpcoming({
				lat: state.lat,
				lon: state.lon,
				heading: state.heading ?? 0,
				speed: speedKMH,
				filters: activeFilterKeys(getCurrentFilters()),
				limit: 10,
				signal: inflight.signal
			});
			stops = res.stops;
			road = res.road ?? null;
			reason = res.reason ?? null;
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') return;
			if (err instanceof ApiError) lastError = err.message;
			else lastError = 'Network error';
		} finally {
			loading = false;
		}
	}

	function getCurrentFilters(): Set<FilterKey> {
		let s = new Set<FilterKey>();
		const u = filters.subscribe((v) => (s = v));
		u();
		return s;
	}

	function pollIntervalMS(state: GeoState): number {
		if (state.status !== 'live') return 60_000;
		const v = kmh(state.speed);
		return v > 20 ? 15_000 : 60_000;
	}

	onMount(() => {
		geo.start();

		lastUnsub = geo.subscribe((s) => {
			if (pollTimer) clearInterval(pollTimer);
			if (s.status === 'live') {
				void refresh(s);
				pollTimer = setInterval(() => void refresh(s), pollIntervalMS(s));
			}
		});

		lastFiltersUnsub = filters.subscribe(() => {
			let cur: GeoState = { status: 'idle' };
			const u = geo.subscribe((v) => (cur = v));
			u();
			void refresh(cur);
		});
	});

	onDestroy(() => {
		geo.stop();
		if (pollTimer) clearInterval(pollTimer);
		inflight?.abort();
		lastUnsub?.();
		lastFiltersUnsub?.();
	});
</script>

<section>
	<FilterChips />

	{#if $geo.status === 'idle' || $geo.status === 'pending'}
		<p class="muted">Waiting for location…</p>
	{:else if $geo.status === 'permission-denied'}
		<p class="error">
			Location permission denied. This app needs your location to find stops ahead.
		</p>
	{:else if $geo.status === 'unavailable'}
		<p class="error">Location is unavailable on this device.</p>
	{:else if $geo.status === 'live'}
		{#if road?.ref}
			<p class="road">
				{road.ref}
				{#if road.direction}<span class="dir"> {road.direction}</span>{/if}
				{#if road.name}<span class="muted"> · {road.name}</span>{/if}
			</p>
		{/if}

		{#if lastError}
			<p class="error">{lastError}</p>
		{/if}

		{#if reason === 'outside-supported-area'}
			<p class="muted">Outside the MVP coverage area (DE / AT / SK / CZ).</p>
		{:else if reason === 'off-highway-or-wrong-direction'}
			<p class="muted">No motorway match yet — keep driving.</p>
		{:else if stops.length === 0 && !loading}
			<p class="muted">No upcoming stops match your filters.</p>
		{/if}

		{#each stops as stop (stop.id)}
			<StopCard {stop} />
		{/each}
	{/if}
</section>

<style>
	.muted {
		color: var(--muted);
	}
	.error {
		color: var(--danger);
	}
	.road {
		font-weight: 600;
		font-size: 1.1rem;
		margin: 0.5rem 0;
	}
	.dir {
		color: var(--accent);
	}
</style>
