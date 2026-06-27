<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import type { Map as LeafletMap, LayerGroup } from 'leaflet';
	import {
		fetchAdminPositions,
		fetchAdminStats,
		fetchAdminTileStops,
		fetchAdminTiles
	} from '$lib/api/admin';
	import { ApiError } from '$lib/api/client';
	import type {
		AdminClient,
		AdminStatsResponse,
		AdminTileInfo,
		AdminTileStop
	} from '$lib/types/admin';

	type LeafletNS = typeof import('leaflet');

	const POSITIONS_POLL_MS = 5_000;
	const TILES_POLL_MS = 30_000;

	let L: LeafletNS | null = null;
	let map: LeafletMap | null = null;
	let mapEl: HTMLDivElement | null = null;
	let clientLayer: LayerGroup | null = null;
	let tileLayer: LayerGroup | null = null;

	let clients: AdminClient[] = [];
	let tiles: AdminTileInfo[] = [];
	let stats: AdminStatsResponse | null = null;
	let selectedTile: AdminTileInfo | null = null;
	let tileStops: AdminTileStop[] = [];
	let tileStopsLoading = false;
	let error = '';
	let unauthorized = false;

	let positionsTimer: ReturnType<typeof setInterval> | null = null;
	let tilesTimer: ReturnType<typeof setInterval> | null = null;
	let hasFittedMap = false;

	function handleError(err: unknown) {
		if (err instanceof ApiError && err.status === 401) {
			unauthorized = true;
			stopPolling();
			return;
		}
		error = err instanceof ApiError ? err.message : 'Network error';
	}

	async function refreshLive() {
		try {
			const [pos, st] = await Promise.all([fetchAdminPositions(), fetchAdminStats()]);
			clients = pos.clients;
			stats = st;
			error = '';
			renderClients();
		} catch (err) {
			handleError(err);
		}
	}

	async function refreshTiles() {
		try {
			const res = await fetchAdminTiles();
			tiles = res.tiles;
			renderTiles();
		} catch (err) {
			handleError(err);
		}
	}

	async function selectTile(tile: AdminTileInfo) {
		selectedTile = tile;
		tileStops = [];
		tileStopsLoading = true;
		try {
			const res = await fetchAdminTileStops(tile.south, tile.west);
			tileStops = res.stops;
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) tileStops = [];
			else handleError(err);
		} finally {
			tileStopsLoading = false;
		}
		if (map) map.fitBounds(tileBounds(tile));
	}

	function tileBounds(t: AdminTileInfo): [[number, number], [number, number]] {
		return [
			[t.south, t.west],
			[t.south + t.size_deg, t.west + t.size_deg]
		];
	}

	function renderClients() {
		if (!L || !map || !clientLayer) return;
		clientLayer.clearLayers();
		for (const c of clients) {
			const marker = L.circleMarker([c.lat, c.lon], {
				radius: 8,
				color: '#2ee27a',
				fillColor: '#2ee27a',
				fillOpacity: 0.5,
				weight: 2
			});
			marker.bindPopup(
				`<strong>${c.client_id.slice(0, 8)}…</strong><br>` +
					`heading ${Math.round(c.heading)}° · ${Math.round(c.speed)} km/h<br>` +
					`±${Math.round(c.accuracy)} m · seen ${formatAgo(c.last_seen)}`
			);
			marker.addTo(clientLayer);
		}
		if (!hasFittedMap && clients.length > 0) {
			map.setView([clients[0].lat, clients[0].lon], 9);
			hasFittedMap = true;
		}
	}

	function renderTiles() {
		if (!L || !map || !tileLayer) return;
		tileLayer.clearLayers();
		for (const t of tiles) {
			const rect = L.rectangle(tileBounds(t), {
				color: t.stops > 0 ? '#4d7cff' : '#5b6685',
				weight: 1,
				fillOpacity: Math.min(0.08 + t.stops * 0.01, 0.3)
			});
			rect.on('click', () => selectTile(t));
			rect.bindTooltip(`${t.stops} stops · ${formatBytes(t.bytes)} · TTL ${formatTTL(t.ttl_seconds)}`);
			rect.addTo(tileLayer);
		}
		if (!hasFittedMap && clients.length === 0 && tiles.length > 0) {
			map.fitBounds(tileBounds(tiles[0]));
			hasFittedMap = true;
		}
	}

	function formatAgo(iso: string): string {
		const sec = Math.max(0, Math.round((Date.now() - Date.parse(iso)) / 1000));
		if (sec < 60) return `${sec}s ago`;
		return `${Math.round(sec / 60)}m ago`;
	}

	function formatBytes(n: number): string {
		if (n >= 1_048_576) return `${(n / 1_048_576).toFixed(1)} MiB`;
		if (n >= 1024) return `${(n / 1024).toFixed(1)} KiB`;
		return `${n} B`;
	}

	function formatTTL(sec: number): string {
		if (sec <= 0) return 'n/a';
		if (sec >= 86_400) return `${(sec / 86_400).toFixed(1)}d`;
		if (sec >= 3600) return `${(sec / 3600).toFixed(1)}h`;
		return `${Math.round(sec / 60)}m`;
	}

	function formatUptime(sec: number): string {
		const d = Math.floor(sec / 86_400);
		const h = Math.floor((sec % 86_400) / 3600);
		const m = Math.floor((sec % 3600) / 60);
		if (d > 0) return `${d}d ${h}h`;
		if (h > 0) return `${h}h ${m}m`;
		return `${m}m`;
	}

	function stopPolling() {
		if (positionsTimer) clearInterval(positionsTimer);
		if (tilesTimer) clearInterval(tilesTimer);
		positionsTimer = null;
		tilesTimer = null;
	}

	function startPolling() {
		stopPolling();
		positionsTimer = setInterval(refreshLive, POSITIONS_POLL_MS);
		tilesTimer = setInterval(refreshTiles, TILES_POLL_MS);
	}

	function onVisibilityChange() {
		if (document.hidden) {
			stopPolling();
		} else if (!unauthorized) {
			void refreshLive();
			void refreshTiles();
			startPolling();
		}
	}

	onMount(async () => {
		L = (await import('leaflet')).default;
		await import('leaflet/dist/leaflet.css');
		await tick();
		if (!mapEl) return;

		map = L.map(mapEl, { zoomControl: true, attributionControl: false }).setView([49.5, 13.0], 5);
		L.tileLayer(
			'https://sgx.geodatenzentrum.de/wmts_topplus_open/tile/1.0.0/web_grau/default/WEBMERCATOR/{z}/{y}/{x}.png',
			{ maxZoom: 18, attribution: '© BKG (TopPlusOpen)' }
		).addTo(map);
		L.control.attribution({ prefix: false }).addAttribution('© BKG').addTo(map);
		tileLayer = L.layerGroup().addTo(map);
		clientLayer = L.layerGroup().addTo(map);

		await Promise.all([refreshLive(), refreshTiles()]);
		startPolling();
		document.addEventListener('visibilitychange', onVisibilityChange);
	});

	onDestroy(() => {
		stopPolling();
		if (typeof document !== 'undefined') {
			document.removeEventListener('visibilitychange', onVisibilityChange);
		}
		if (map) map.remove();
	});
</script>

<svelte:head>
	<title>Admin · Reststop Navigator</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="admin">
	<header>
		<h1>Live view</h1>
		{#if stats}
			<div class="stats">
				<span class="stat"><strong>{stats.presence_count}</strong> live</span>
				<span class="stat"><strong>{tiles.length}</strong> tiles</span>
				<span class="stat">
					cache <strong>{stats.cache.hits}</strong>/<strong class="muted"
						>{stats.cache.hits + stats.cache.misses}</strong
					>
				</span>
				{#if stats.redis}
					<span class="stat">redis <strong>{formatBytes(stats.redis.used_memory_bytes)}</strong></span>
				{/if}
				<span class="stat">up <strong>{formatUptime(stats.uptime_seconds)}</strong></span>
			</div>
		{/if}
	</header>

	{#if unauthorized}
		<p class="error">Unauthorized — reload and enter the admin password.</p>
	{:else}
		{#if error}
			<p class="error">{error}</p>
		{/if}

		<div class="map" bind:this={mapEl}></div>

		<section>
			<h2>Live clients ({clients.length})</h2>
			{#if clients.length === 0}
				<p class="muted">No clients in the last 15 minutes.</p>
			{:else}
				<table>
					<thead>
						<tr><th>Client</th><th>Position</th><th>Heading</th><th>Speed</th><th>Seen</th></tr>
					</thead>
					<tbody>
						{#each clients as c (c.client_id)}
							<tr>
								<td class="mono">{c.client_id.slice(0, 8)}…</td>
								<td class="mono">{c.lat.toFixed(4)}, {c.lon.toFixed(4)}</td>
								<td>{Math.round(c.heading)}°</td>
								<td>{Math.round(c.speed)} km/h</td>
								<td>{formatAgo(c.last_seen)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</section>

		<section class="tiles">
			<h2>Cached tiles ({tiles.length})</h2>
			{#if tiles.length === 0}
				<p class="muted">Tile cache is empty.</p>
			{:else}
				<table>
					<thead>
						<tr><th>Tile</th><th>Stops</th><th>Ways</th><th>Size</th><th>TTL</th></tr>
					</thead>
					<tbody>
						{#each tiles as t (t.key)}
							<tr class:selected={selectedTile?.key === t.key}>
								<td class="mono">
									<button class="linklike" on:click={() => selectTile(t)}>
										{t.south.toFixed(1)}, {t.west.toFixed(1)}
									</button>
								</td>
								<td>{t.stops}</td>
								<td>{t.ways}</td>
								<td>{formatBytes(t.bytes)}</td>
								<td>{formatTTL(t.ttl_seconds)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</section>

		{#if selectedTile}
			<section>
				<h2>
					Stops in tile {selectedTile.south.toFixed(1)}, {selectedTile.west.toFixed(1)}
				</h2>
				{#if tileStopsLoading}
					<p class="muted">Loading…</p>
				{:else if tileStops.length === 0}
					<p class="muted">No stops in this tile.</p>
				{:else}
					<table>
						<thead>
							<tr><th>Name</th><th>Kind</th><th>Road</th><th>Amenities</th></tr>
						</thead>
						<tbody>
							{#each tileStops as s (s.osm_type + s.osm_id)}
								<tr>
									<td>{s.name || '—'}</td>
									<td>{s.kind}</td>
									<td class="mono">{s.highway_ref || '—'}</td>
									<td class="mono">
										{[
											s.amenities.fuel && 'fuel',
											s.amenities.charging && 'ev',
											s.amenities.food && 'food',
											s.amenities.toilets && 'wc',
											s.amenities.open24h && '24/7',
											s.amenities.dog && 'dog'
										]
											.filter(Boolean)
											.join(' ') || '—'}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</section>
		{/if}
	{/if}
</div>

<style>
	.admin {
		max-width: 960px;
		margin: 0 auto;
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	header {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 12px;
	}
	h1 {
		font-size: 1.4rem;
		margin: 0;
	}
	h2 {
		font-size: 1rem;
		margin: 0 0 8px;
		color: var(--muted);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}
	.stats {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		font-size: 0.85rem;
		color: var(--muted);
	}
	.stat strong {
		color: var(--text-strong);
	}
	.map {
		height: 380px;
		border-radius: 12px;
		border: 1px solid var(--border);
		overflow: hidden;
		background: var(--surface);
	}
	.map :global(.leaflet-container) {
		height: 100%;
		background: var(--bg-elev);
	}
	section {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 12px;
		padding: 12px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.85rem;
	}
	th {
		text-align: left;
		color: var(--muted-2);
		font-weight: 500;
		padding: 4px 8px;
		border-bottom: 1px solid var(--border);
	}
	td {
		padding: 6px 8px;
		border-bottom: 1px solid var(--border);
	}
	tbody tr:last-child td {
		border-bottom: none;
	}
	tr.selected td {
		background: var(--accent-glow);
	}
	.linklike {
		background: none;
		border: none;
		padding: 0;
		font: inherit;
		color: var(--cool);
		cursor: pointer;
		text-decoration: underline;
	}
	.mono {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8rem;
	}
	.muted {
		color: var(--muted);
	}
	.error {
		color: var(--danger);
		background: var(--surface);
		border: 1px solid var(--danger);
		border-radius: 8px;
		padding: 8px 12px;
	}
</style>
