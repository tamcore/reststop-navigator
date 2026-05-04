<script lang="ts">
	import { page } from '$app/stores';
	import { onDestroy, onMount, tick } from 'svelte';
	import { ApiError, fetchStopDetail } from '$lib/api/client';
	import { geo, kmh, type GeoState } from '$lib/stores/geo';
	import type { DetailResponse } from '$lib/types/api';

	let detail = $state<DetailResponse | null>(null);
	let error = $state<string | null>(null);
	let loading = $state(true);
	let liveDistanceM = $state<number | null>(null);
	let liveETASeconds = $state<number | null>(null);
	let mapEl = $state<HTMLDivElement | null>(null);

	type LeafletNS = typeof import('leaflet');
	let L: LeafletNS | null = null;
	type LMap = ReturnType<LeafletNS['map']>;
	let map: LMap | null = null;
	let userMarker: ReturnType<LeafletNS['circleMarker']> | null = null;
	let stopMarker: ReturnType<LeafletNS['marker']> | null = null;
	let line: ReturnType<LeafletNS['polyline']> | null = null;
	let geoUnsub: (() => void) | null = null;

	function distanceM(a: { lat: number; lon: number }, b: { lat: number; lon: number }): number {
		const R = 6371008.8;
		const toRad = (d: number) => (d * Math.PI) / 180;
		const phi1 = toRad(a.lat);
		const phi2 = toRad(b.lat);
		const dPhi = toRad(b.lat - a.lat);
		const dLam = toRad(b.lon - a.lon);
		const h =
			Math.sin(dPhi / 2) ** 2 + Math.cos(phi1) * Math.cos(phi2) * Math.sin(dLam / 2) ** 2;
		return 2 * R * Math.asin(Math.sqrt(h));
	}

	function updateLive(state: GeoState) {
		if (!detail || state.status !== 'live') {
			liveDistanceM = null;
			liveETASeconds = null;
			return;
		}
		const stop = { lat: detail.stop.lat, lon: detail.stop.lon };
		const dist = distanceM({ lat: state.lat, lon: state.lon }, stop);
		liveDistanceM = dist;

		// ETA = remaining metres / current speed. Speed comes from the
		// Geolocation API in m/s; clamp to 60 km/h (≈16.7 m/s) so a parked
		// user still sees a meaningful number.
		const speedMS = Math.max(state.speed ?? 0, 16.7);
		liveETASeconds = Math.round(dist / speedMS);

		if (!map || !L) return;
		const userLatLng = L.latLng(state.lat, state.lon);
		if (userMarker) {
			userMarker.setLatLng(userLatLng);
		} else {
			userMarker = L.circleMarker(userLatLng, {
				radius: 8,
				color: '#22c55e',
				fillColor: '#22c55e',
				fillOpacity: 0.9
			}).addTo(map);
		}
		const stopLatLng = L.latLng(stop.lat, stop.lon);
		if (line) line.remove();
		line = L.polyline([userLatLng, stopLatLng], {
			color: '#22c55e',
			weight: 3,
			opacity: 0.7,
			dashArray: '6 6'
		}).addTo(map);
		map.fitBounds(L.latLngBounds(userLatLng, stopLatLng), { padding: [40, 40], maxZoom: 13 });
	}

	function fmtDistance(m: number): string {
		if (m < 1000) return `${Math.round(m)} m`;
		return `${(m / 1000).toFixed(1)} km`;
	}
	function fmtETA(s: number): string {
		if (s < 60) return `${s} s`;
		const min = Math.round(s / 60);
		if (min < 60) return `${min} min`;
		const h = Math.floor(min / 60);
		return `${h} h ${min % 60} min`;
	}

	onMount(async () => {
		const raw = $page.params.id;
		if (!raw) {
			error = 'Stop id is missing.';
			loading = false;
			return;
		}
		const id = decodeURIComponent(raw);
		const lat = parseFloat($page.url.searchParams.get('lat') ?? '');
		const lon = parseFloat($page.url.searchParams.get('lon') ?? '');
		if (!Number.isFinite(lat) || !Number.isFinite(lon)) {
			error = 'Stop link is missing location.';
			loading = false;
			return;
		}
		try {
			detail = await fetchStopDetail(id, { lat, lon });
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) error = 'Stop not found.';
			else if (err instanceof ApiError) error = err.message;
			else error = 'Network error';
		} finally {
			loading = false;
		}

		if (!detail) return;

		// Dynamic import so leaflet only loads on the detail page.
		L = (await import('leaflet')).default;
		await import('leaflet/dist/leaflet.css');
		await tick();
		if (!mapEl) return;

		map = L.map(mapEl, { zoomControl: true }).setView([detail.stop.lat, detail.stop.lon], 11);
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			attribution: '© OpenStreetMap',
			maxZoom: 19
		}).addTo(map);
		stopMarker = L.marker([detail.stop.lat, detail.stop.lon]).addTo(map);

		geo.start();
		geoUnsub = geo.subscribe((s) => updateLive(s));
	});

	onDestroy(() => {
		geoUnsub?.();
		if (map) map.remove();
		geo.stop();
	});
</script>

{#if loading}
	<p class="muted">Loading…</p>
{:else if error}
	<p class="error">{error}</p>
	<p><a href="/">Back to list</a></p>
{:else if detail}
	{@const s = detail.stop}
	<section>
		<a class="back" href="/">← Back</a>

		<h1>{s.name || 'Rest area'}</h1>
		<p class="muted">{s.kind} · {detail.country}</p>

		<div bind:this={mapEl} class="map"></div>

		<div class="live">
			{#if liveDistanceM !== null && liveETASeconds !== null}
				<div>
					<span class="live-num">{fmtDistance(liveDistanceM)}</span>
					<span class="live-label">remaining</span>
				</div>
				<div>
					<span class="live-num">{fmtETA(liveETASeconds)}</span>
					<span class="live-label">
						ETA
						{#if $geo.status === 'live'}
							@ {kmh($geo.speed).toFixed(0)} km/h
						{/if}
					</span>
				</div>
			{:else}
				<span class="muted">Waiting for live location…</span>
			{/if}
		</div>

		<div class="amenities">
			{#if s.amenities.fuel}<span class="badge">⛽ Fuel</span>{/if}
			{#if s.amenities.charging}<span class="badge">🔌 EV</span>{/if}
			{#if s.amenities.food}<span class="badge">🍴 Food</span>{/if}
			{#if s.amenities.toilets}<span class="badge">🚻 Toilets</span>{/if}
			{#if s.amenities.open24h}<span class="badge">⏰ 24/7</span>{/if}
			{#if s.amenities.dog}<span class="badge">🐕 Dog</span>{/if}
		</div>

		{#if s.opening_hours}
			<p><strong>Opening hours:</strong> {s.opening_hours}</p>
		{/if}
		{#if s.operator}
			<p><strong>Operator:</strong> {s.operator}</p>
		{/if}

		<div class="links">
			<a class="btn primary" href={detail.deep_links.google} target="_blank" rel="noopener">
				Google Maps
			</a>
			<a class="btn" href={detail.deep_links.apple} target="_blank" rel="noopener">
				Apple Maps
			</a>
			<a class="btn" href={detail.deep_links.waze} target="_blank" rel="noopener">
				Waze
			</a>
		</div>

		<p class="muted small">
			Google Maps adds the stop as a destination. Waze replaces any active route. Apple Maps
			depends on iOS version.
		</p>
	</section>
{/if}

<style>
	.back {
		color: var(--accent);
		text-decoration: none;
		font-size: 0.9rem;
	}
	h1 {
		font-size: 1.4rem;
		margin: 0.5rem 0 0.25rem;
	}
	.muted {
		color: var(--muted);
	}
	.small {
		font-size: 0.8rem;
	}
	.error {
		color: var(--danger);
	}
	.map {
		width: 100%;
		height: 240px;
		margin: 0.75rem 0;
		border-radius: 12px;
		overflow: hidden;
		border: 1px solid var(--border);
		background: #0b1220;
	}
	.live {
		display: flex;
		gap: 1.5rem;
		align-items: baseline;
		margin: 0.5rem 0 1rem;
	}
	.live-num {
		font-size: 1.4rem;
		font-weight: 700;
		color: var(--accent);
		font-variant-numeric: tabular-nums;
	}
	.live-label {
		display: block;
		color: var(--muted);
		font-size: 0.8rem;
	}
	.amenities {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin: 0.75rem 0;
	}
	.badge {
		padding: 0.2rem 0.55rem;
		background: rgba(34, 197, 94, 0.15);
		border-radius: 999px;
		font-size: 0.85rem;
	}
	.links {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	.btn {
		display: block;
		padding: 0.85rem 1rem;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--surface);
		color: var(--text);
		text-decoration: none;
		text-align: center;
		font-weight: 600;
	}
	.btn.primary {
		background: var(--accent-strong);
		border-color: var(--accent);
		color: #fff;
	}
</style>
