<script lang="ts">
	import { kmh, type GeoState } from '$lib/stores/geo';

	type Props = {
		geo: GeoState & { status: 'live' };
		reason: string | null;
	};

	let { geo, reason }: Props = $props();

	function formatCoord(val: number, pos: string, neg: string): string {
		const dir = val >= 0 ? pos : neg;
		return `${Math.abs(val).toFixed(4)}° ${dir}`;
	}

	function headingToCardinal(deg: number): string {
		const dirs = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW'];
		return dirs[Math.round(deg / 45) % 8];
	}

	function accuracyClass(acc: number): string {
		if (acc <= 30) return 'acc-good';
		if (acc <= 100) return 'acc-fair';
		return 'acc-poor';
	}
</script>

<div class="geo-status" role="status" aria-live="polite">
	<div class="status-header">
		<span class="status-label">GPS Active</span>
		{#if reason === 'off-highway-or-wrong-direction' || !reason}
			<span class="search-pulse">Searching for motorway…</span>
		{:else if reason === 'outside-supported-area'}
			<span class="search-info">Outside coverage area</span>
		{/if}
	</div>

	<div class="telemetry">
		<div class="tele-row">
			<span class="tele-label">Position</span>
			<span class="tele-value mono">
				{formatCoord(geo.lat, 'N', 'S')}, {formatCoord(geo.lon, 'E', 'W')}
			</span>
		</div>

		<div class="tele-row">
			<span class="tele-label">Accuracy</span>
			<span class="tele-value mono {accuracyClass(geo.accuracy)}">
				±{Math.round(geo.accuracy)} m
			</span>
		</div>

		<div class="tele-row">
			<span class="tele-label">Heading</span>
			<span class="tele-value mono">
				{#if geo.heading !== null}
					{headingToCardinal(geo.heading)} · {Math.round(geo.heading)}°
				{:else}
					—
				{/if}
			</span>
		</div>

		<div class="tele-row">
			<span class="tele-label">Speed</span>
			<span class="tele-value mono">{kmh(geo.speed).toFixed(0)} km/h</span>
		</div>
	</div>
</div>

<style>
	.geo-status {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		padding: 0.25rem 0;
		width: 100%;
	}
	.status-header {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.status-label {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.78rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--accent);
	}
	.search-pulse {
		font-size: 0.8rem;
		color: var(--muted);
		animation: pulse 2s ease-in-out infinite;
	}
	.search-info {
		font-size: 0.8rem;
		color: var(--cool);
	}
	@keyframes pulse {
		0%, 100% { opacity: 0.5; }
		50% { opacity: 1; }
	}

	.telemetry {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}
	.tele-row {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 0.75rem;
	}
	.tele-label {
		font-family: var(--font-display);
		font-size: 0.68rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--muted);
		flex-shrink: 0;
	}
	.tele-value {
		font-size: 0.82rem;
		color: var(--text);
		text-align: right;
	}

	.acc-good { color: var(--accent); }
	.acc-fair { color: var(--warn, #f59e0b); }
	.acc-poor { color: var(--danger); }
</style>
