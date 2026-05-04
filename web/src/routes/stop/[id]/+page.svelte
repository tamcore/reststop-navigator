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
		const speedMS = Math.max(state.speed ?? 0, 16.7);
		liveETASeconds = Math.round(dist / speedMS);

		if (!map || !L) return;
		const userLatLng = L.latLng(state.lat, state.lon);
		if (userMarker) {
			userMarker.setLatLng(userLatLng);
		} else {
			userMarker = L.circleMarker(userLatLng, {
				radius: 8,
				color: '#2ee27a',
				fillColor: '#2ee27a',
				fillOpacity: 0.95,
				weight: 3
			}).addTo(map);
		}
		const stopLatLng = L.latLng(stop.lat, stop.lon);
		if (line) line.remove();
		line = L.polyline([userLatLng, stopLatLng], {
			color: '#2ee27a',
			weight: 3,
			opacity: 0.7,
			dashArray: '6 6'
		}).addTo(map);
		map.fitBounds(L.latLngBounds(userLatLng, stopLatLng), { padding: [40, 40], maxZoom: 13 });
	}

	function fmtDistance(m: number): { num: string; unit: string } {
		if (m < 1000) return { num: String(Math.round(m)), unit: 'm' };
		return { num: (m / 1000).toFixed(1), unit: 'km' };
	}
	function fmtETA(s: number): { num: string; unit: string } {
		if (s < 60) return { num: String(s), unit: 's' };
		const min = Math.round(s / 60);
		if (min < 60) return { num: String(min), unit: 'min' };
		const h = Math.floor(min / 60);
		return { num: `${h}:${String(min % 60).padStart(2, '0')}`, unit: 'h' };
	}

	const amenities: {
		key: keyof NonNullable<DetailResponse['stop']>['amenities'];
		label: string;
		icon: string;
	}[] = [
		{ key: 'fuel', label: 'FUEL', icon: '⛽' },
		{ key: 'charging', label: 'EV', icon: '⚡' },
		{ key: 'food', label: 'FOOD', icon: '🍴' },
		{ key: 'toilets', label: 'WC', icon: '🚻' },
		{ key: 'open24h', label: '24/7', icon: '🕐' },
		{ key: 'dog', label: 'DOG', icon: '🐕' }
	];

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

		L = (await import('leaflet')).default;
		await import('leaflet/dist/leaflet.css');
		await tick();
		if (!mapEl) return;

		map = L.map(mapEl, { zoomControl: true, attributionControl: false }).setView(
			[detail.stop.lat, detail.stop.lon],
			11
		);
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			attribution: '© OpenStreetMap',
			maxZoom: 19
		}).addTo(map);
		L.control.attribution({ prefix: false }).addAttribution('© OSM').addTo(map);
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
	<div class="loader">
		<div class="loader-bar"></div>
	</div>
{:else if error}
	<p class="error">{error}</p>
	<p><a href="/" class="back">← Back to list</a></p>
{:else if detail}
	{@const s = detail.stop}
	<a class="back" href="/">
		<svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
			<path d="M15 5 L8 12 L15 19" fill="none" stroke="currentColor" stroke-width="2.4" />
		</svg>
		<span>Back</span>
	</a>

	<div class="kind">{s.kind === 'services' ? 'SERVICES' : 'REST AREA'} · {detail.country}</div>
	<h1>{s.name || 'Rest area'}</h1>
	{#if s.operator}
		<p class="operator">{s.operator}</p>
	{/if}

	<div bind:this={mapEl} class="map"></div>

	<div class="live">
		{#if liveDistanceM !== null && liveETASeconds !== null}
			{@const d = fmtDistance(liveDistanceM)}
			{@const e = fmtETA(liveETASeconds)}
			<div class="tile">
				<div class="tile-num mono">
					{d.num}<span class="tile-unit">{d.unit}</span>
				</div>
				<div class="tile-label">Remaining</div>
			</div>
			<div class="tile">
				<div class="tile-num mono">
					{e.num}<span class="tile-unit">{e.unit}</span>
				</div>
				<div class="tile-label">
					ETA
					{#if $geo.status === 'live'}
						<span class="dim">@ {kmh($geo.speed).toFixed(0)} km/h</span>
					{/if}
				</div>
			</div>
		{:else}
			<div class="tile">
				<div class="tile-num mono">—</div>
				<div class="tile-label">Acquiring GPS…</div>
			</div>
		{/if}
	</div>

	<div class="section-label">Amenities</div>
	<div class="amen-grid">
		{#each amenities as a (a.key)}
			{@const on = !!s.amenities[a.key]}
			<div class="amen-tile" class:on>
				<span class="amen-icon">{a.icon}</span>
				<span class="amen-label">{a.label}</span>
				{#if on}
					<span class="amen-dot" aria-hidden="true">✓</span>
				{/if}
			</div>
		{/each}
	</div>

	{#if s.opening_hours}
		<div class="kv">
			<span class="kv-k">Opening hours</span>
			<span class="kv-v">{s.opening_hours}</span>
		</div>
	{/if}

	<div class="section-label">Send to navigation</div>
	<div class="links">
		<a class="btn primary" href={detail.deep_links.google} target="_blank" rel="noopener">
			<span class="btn-label">Google Maps</span>
			<span class="btn-hint">Add as waypoint</span>
		</a>
		<a class="btn" href={detail.deep_links.apple} target="_blank" rel="noopener">
			<span class="btn-label">Apple Maps</span>
			<span class="btn-hint">Open destination</span>
		</a>
		<a class="btn" href={detail.deep_links.waze} target="_blank" rel="noopener">
			<span class="btn-label">Waze</span>
			<span class="btn-hint">Replaces active route</span>
		</a>
	</div>
{/if}

<style>
	.loader {
		height: 3px;
		background: var(--surface);
		overflow: hidden;
		margin: 0 0 2rem;
		border-radius: 2px;
	}
	.loader-bar {
		width: 30%;
		height: 100%;
		background: linear-gradient(90deg, transparent, var(--accent), transparent);
		animation: slide 1.2s linear infinite;
	}
	@keyframes slide {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(400%);
		}
	}

	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		color: var(--accent);
		text-decoration: none;
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.78rem;
		letter-spacing: 0.2em;
		text-transform: uppercase;
		margin: 0.25rem 0 0.75rem;
	}
	.back:hover {
		filter: brightness(1.15);
	}

	.kind {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.7rem;
		letter-spacing: 0.28em;
		color: var(--muted);
		text-transform: uppercase;
	}
	h1 {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: clamp(1.6rem, 5.5vw, 2.1rem);
		line-height: 1.05;
		letter-spacing: 0.005em;
		margin: 0.2rem 0 0.4rem;
		color: var(--text-strong);
	}
	.operator {
		color: var(--muted);
		font-size: 0.85rem;
		margin: 0 0 0.5rem;
	}
	.error {
		color: var(--danger);
	}

	.map {
		width: 100%;
		height: 220px;
		margin: 0.75rem 0 1rem;
		border-radius: var(--radius-card);
		overflow: hidden;
		border: 1px solid var(--border);
		background: var(--bg-elev);
	}
	.map :global(.leaflet-container) {
		font: inherit;
		background: var(--bg-elev);
	}
	.map :global(.leaflet-control-attribution) {
		background: rgba(6, 9, 18, 0.7);
		color: var(--muted);
		backdrop-filter: blur(4px);
		font-size: 10px;
	}

	.live {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.6rem;
		margin: 0.5rem 0 0.5rem;
	}
	.tile {
		padding: 0.85rem 1rem;
		border-radius: var(--radius-card);
		background:
			radial-gradient(120% 100% at 0% 0%, rgba(46, 226, 122, 0.08), transparent 60%),
			var(--surface);
		border: 1px solid var(--border);
	}
	.tile-num {
		font-size: 1.9rem;
		font-weight: 700;
		color: var(--text-strong);
		line-height: 1;
		display: inline-flex;
		align-items: baseline;
	}
	.tile-unit {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.75rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--muted);
		margin-left: 0.3rem;
	}
	.tile-label {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.7rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--muted);
		margin-top: 0.5rem;
	}
	.dim {
		color: var(--muted-2);
	}

	.amen-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.5rem;
	}
	.amen-tile {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.3rem;
		padding: 0.85rem 0.5rem 0.7rem;
		border-radius: var(--radius-card);
		background: var(--surface);
		border: 1px solid var(--border);
		text-align: center;
		opacity: 0.4;
		transition: opacity 0.2s, border-color 0.2s, background 0.2s;
	}
	.amen-tile.on {
		opacity: 1;
		border-color: rgba(46, 226, 122, 0.4);
		background:
			radial-gradient(80% 100% at 50% 0%, rgba(46, 226, 122, 0.08), transparent 60%),
			var(--surface);
	}
	.amen-icon {
		font-size: 1.4rem;
		filter: grayscale(0.6);
	}
	.amen-tile.on .amen-icon {
		filter: none;
	}
	.amen-label {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.7rem;
		letter-spacing: 0.22em;
		color: var(--muted);
	}
	.amen-tile.on .amen-label {
		color: var(--accent);
	}
	.amen-dot {
		position: absolute;
		top: 0.4rem;
		right: 0.5rem;
		font-size: 0.7rem;
		color: var(--accent);
	}

	.kv {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		padding: 0.7rem 0;
		border-bottom: 1px dashed var(--border);
	}
	.kv-k {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.72rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.kv-v {
		font-family: var(--font-mono);
		color: var(--text-strong);
	}

	.links {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin: 0.4rem 0 0.5rem;
	}
	.btn {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.95rem 1.1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		background: var(--surface);
		color: var(--text);
		text-decoration: none;
		transition: transform 0.15s var(--ease-spring), border-color 0.2s, box-shadow 0.2s;
	}
	.btn:hover {
		transform: translateY(-1px);
		border-color: var(--border-strong);
	}
	.btn:active {
		transform: translateY(0);
	}
	.btn-label {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.95rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--text-strong);
	}
	.btn-hint {
		font-size: 0.78rem;
		color: var(--muted);
	}
	.btn.primary {
		background: linear-gradient(180deg, #34f088, var(--accent-strong));
		border-color: var(--accent-strong);
		box-shadow: 0 12px 28px -16px rgba(46, 226, 122, 0.6);
	}
	.btn.primary .btn-label,
	.btn.primary .btn-hint {
		color: var(--bg);
	}
	.btn.primary .btn-hint {
		opacity: 0.7;
	}
</style>
