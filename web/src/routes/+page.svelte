<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { ApiError, fetchUpcoming } from '$lib/api/client';
	import { filters } from '$lib/stores/filters';
	import { geo, kmh, type GeoState } from '$lib/stores/geo';
	import FilterChips from '$lib/components/FilterChips.svelte';
	import StopCard from '$lib/components/StopCard.svelte';
	import RoadShield from '$lib/components/RoadShield.svelte';
	import { ALL_FILTERS, type FilterKey, type StopInfo, type Road } from '$lib/types/api';

	let stops: StopInfo[] = $state([]);
	let road: Road | null = $state(null);
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

<section class="hero">
	{#if $geo.status === 'live' && road}
		<RoadShield ref={road.ref ?? ''} direction={road.direction ?? ''} name={road.name ?? ''} />
		<div class="speed">
			<span class="speed-num mono">{kmh($geo.speed).toFixed(0)}</span>
			<span class="speed-unit">km/h</span>
		</div>
	{:else if $geo.status === 'live' && !road}
		<div class="hero-empty">
			<span class="hero-empty-label">Standby</span>
			<span class="hero-empty-msg">No motorway match yet.</span>
		</div>
	{:else if $geo.status === 'idle' || $geo.status === 'pending'}
		<div class="hero-empty">
			<span class="hero-empty-label">Acquiring GPS</span>
			<div class="dots"><i></i><i></i><i></i></div>
		</div>
	{:else if $geo.status === 'permission-denied'}
		<div class="hero-empty error">
			<span class="hero-empty-label">Location denied</span>
			<span class="hero-empty-msg">
				This app needs your location to find stops ahead. Allow it in your browser.
			</span>
		</div>
	{:else if $geo.status === 'unavailable'}
		<div class="hero-empty error">
			<span class="hero-empty-label">Location unavailable</span>
			<span class="hero-empty-msg">No geolocation on this device.</span>
		</div>
	{/if}
</section>

<FilterChips />

<div class="section-label">Next stops</div>

{#if lastError}
	<p class="error">{lastError}</p>
{/if}

{#if reason === 'outside-supported-area'}
	<div class="info-panel">
		<div class="info-title">Outside coverage</div>
		<p>This MVP only tracks rest stops in 🇩🇪 Germany, 🇦🇹 Austria, 🇸🇰 Slovakia and 🇨🇿 Czechia.</p>
	</div>
{:else if reason === 'off-highway-or-wrong-direction'}
	<div class="info-panel">
		<div class="info-title">Waiting for motorway match</div>
		<p>We only track motorways (Autobahnen). Keep driving until you're on one of these:</p>
		<ul class="road-list">
			<li>
				<span class="flag">🇩🇪</span>
				<span class="cat">Bundesautobahn</span>
				<span class="prefix">A&nbsp;1 – A&nbsp;995</span>
			</li>
			<li>
				<span class="flag">🇦🇹</span>
				<span class="cat">Autobahn</span>
				<span class="prefix">A&nbsp;1 – A&nbsp;26</span>
			</li>
			<li>
				<span class="flag">🇸🇰</span>
				<span class="cat">Diaľnica</span>
				<span class="prefix">D&nbsp;1 – D&nbsp;4</span>
			</li>
			<li>
				<span class="flag">🇨🇿</span>
				<span class="cat">Dálnice</span>
				<span class="prefix">D&nbsp;1 – D&nbsp;56</span>
			</li>
		</ul>
		<p class="hint">Schnellstraßen / Bundesstraßen / city expressways aren't covered yet.</p>
	</div>
{:else if stops.length === 0 && !loading && $geo.status === 'live'}
	<p class="muted">No upcoming stops match your filters.</p>
{/if}

<ol class="stop-list">
	{#each stops as stop, i (stop.id)}
		<li style="--i: {i}">
			<StopCard {stop} />
		</li>
	{/each}
</ol>

<style>
	.hero {
		position: relative;
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1rem;
		padding: 1.25rem 1rem 1.5rem;
		margin: 0.5rem 0 0.5rem;
		border-radius: 20px;
		background:
			radial-gradient(120% 100% at 0% 0%, rgba(46, 226, 122, 0.08), transparent 60%),
			linear-gradient(180deg, var(--bg-elev) 0%, var(--surface) 100%);
		border: 1px solid var(--border);
		box-shadow: 0 24px 60px -30px rgba(46, 226, 122, 0.25);
		overflow: hidden;
	}
	.hero::before {
		content: '';
		position: absolute;
		inset: 0;
		background-image: repeating-linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.025) 0,
			rgba(255, 255, 255, 0.025) 1px,
			transparent 1px,
			transparent 22px
		);
		pointer-events: none;
		mask-image: linear-gradient(180deg, black 30%, transparent 100%);
	}
	.speed {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		line-height: 1;
		position: relative;
	}
	.speed-num {
		font-size: 2.4rem;
		font-weight: 700;
		color: var(--text-strong);
		letter-spacing: -0.02em;
	}
	.speed-unit {
		font-family: var(--font-display);
		font-size: 0.75rem;
		letter-spacing: 0.2em;
		text-transform: uppercase;
		color: var(--muted);
		margin-top: 0.25rem;
	}

	.hero-empty {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		padding: 0.5rem 0;
	}
	.hero-empty-label {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.95rem;
		letter-spacing: 0.24em;
		color: var(--accent);
		text-transform: uppercase;
	}
	.hero-empty.error .hero-empty-label {
		color: var(--danger);
	}
	.hero-empty-msg {
		color: var(--muted);
		font-size: 0.9rem;
		max-width: 28ch;
	}

	.dots {
		display: inline-flex;
		gap: 6px;
		margin-top: 0.25rem;
	}
	.dots i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--accent);
		opacity: 0.4;
		animation: blink 1.2s infinite var(--ease-out);
	}
	.dots i:nth-child(2) {
		animation-delay: 0.15s;
	}
	.dots i:nth-child(3) {
		animation-delay: 0.3s;
	}
	@keyframes blink {
		0%,
		100% {
			opacity: 0.25;
		}
		40% {
			opacity: 1;
		}
	}

	.muted {
		color: var(--muted);
		padding: 0.25rem 0.25rem 0.5rem;
	}
	.error {
		color: var(--danger);
		padding: 0.25rem 0.25rem 0.5rem;
	}

	.info-panel {
		padding: 1rem 1.1rem 1.1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		background:
			radial-gradient(120% 100% at 0% 0%, rgba(77, 124, 255, 0.08), transparent 60%),
			var(--surface);
		margin-bottom: 0.75rem;
	}
	.info-title {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.78rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--cool);
		margin-bottom: 0.5rem;
	}
	.info-panel p {
		margin: 0 0 0.5rem;
		color: var(--text);
		font-size: 0.9rem;
		line-height: 1.45;
	}
	.info-panel .hint {
		color: var(--muted);
		font-size: 0.8rem;
		margin-top: 0.6rem;
		margin-bottom: 0;
	}
	.road-list {
		list-style: none;
		padding: 0;
		margin: 0.4rem 0 0;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.4rem 0.75rem;
	}
	.road-list li {
		display: grid;
		grid-template-columns: 1.5rem 1fr;
		grid-template-rows: auto auto;
		column-gap: 0.5rem;
		align-items: center;
		padding: 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--bg-elev);
	}
	.flag {
		grid-row: 1 / span 2;
		font-size: 1.15rem;
		line-height: 1;
	}
	.cat {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.78rem;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--text-strong);
	}
	.prefix {
		font-family: var(--font-mono);
		font-size: 0.72rem;
		color: var(--accent);
	}
	.stop-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.stop-list li {
		opacity: 0;
		animation: rise 0.36s var(--ease-spring) forwards;
		animation-delay: calc(var(--i) * 50ms);
	}
</style>
