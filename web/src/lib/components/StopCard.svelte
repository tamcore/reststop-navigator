<script lang="ts">
	import type { StopInfo } from '$lib/types/api';

	type Props = { stop: StopInfo };
	let { stop }: Props = $props();

	function distanceLabel(m: number): { num: string; unit: string } {
		if (m < 1000) return { num: String(m), unit: 'm' };
		return { num: (m / 1000).toFixed(1), unit: 'km' };
	}
	function etaLabel(s: number): string {
		if (s < 60) return `${s} s`;
		const min = Math.round(s / 60);
		if (min < 60) return `${min} min`;
		const h = Math.floor(min / 60);
		const r = min % 60;
		return r ? `${h}h ${r}m` : `${h} h`;
	}

	const amenityChips: { key: keyof StopInfo['amenities']; label: string }[] = [
		{ key: 'fuel', label: 'FUEL' },
		{ key: 'charging', label: 'EV' },
		{ key: 'food', label: 'FOOD' },
		{ key: 'toilets', label: 'WC' },
		{ key: 'open24h', label: '24/7' },
		{ key: 'dog', label: 'DOG' }
	];

	let dist = $derived(distanceLabel(stop.distance_m));
	let kindLabel = $derived(stop.kind === 'services' ? 'Services' : 'Rast');
</script>

<a class="card" href="/stop/{encodeURIComponent(stop.id)}?lat={stop.lat}&lon={stop.lon}">
	<div class="rail" aria-hidden="true"></div>
	<div class="left">
		<div class="kind">{kindLabel}</div>
		<div class="name">{stop.name || 'Rest area'}</div>
		<div class="amen">
			{#each amenityChips as a (a.key)}
				{#if stop.amenities[a.key]}
					<span class="chip-active">{a.label}</span>
				{/if}
			{/each}
		</div>
	</div>
	<div class="right">
		<div class="dist mono">
			<span class="dist-num">{dist.num}</span>
			<span class="dist-unit">{dist.unit}</span>
		</div>
		<div class="eta mono">{etaLabel(stop.eta_seconds)}</div>
	</div>
	<svg class="arrow" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
		<path d="M9 5 L16 12 L9 19" fill="none" stroke="currentColor" stroke-width="2.4" />
	</svg>
</a>

<style>
	.card {
		position: relative;
		display: grid;
		grid-template-columns: 4px 1fr auto;
		gap: 0.85rem;
		align-items: center;
		padding: 0.9rem 1rem 0.9rem 0.5rem;
		margin-bottom: 0.6rem;
		background:
			linear-gradient(180deg, rgba(255, 255, 255, 0.012), transparent 60%),
			var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		color: var(--text);
		text-decoration: none;
		overflow: hidden;
		transition: transform 0.18s var(--ease-spring), border-color 0.2s, box-shadow 0.2s;
	}
	.card:hover,
	.card:focus-visible {
		transform: translateY(-1px);
		border-color: var(--border-strong);
		box-shadow: 0 16px 30px -20px rgba(46, 226, 122, 0.35);
		outline: none;
	}
	.card:active {
		transform: translateY(0);
	}
	.rail {
		grid-column: 1;
		align-self: stretch;
		width: 4px;
		background: linear-gradient(180deg, var(--accent), var(--accent-strong));
		border-radius: 4px;
		opacity: 0.85;
	}
	.left {
		grid-column: 2;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		min-width: 0;
	}
	.kind {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.65rem;
		letter-spacing: 0.28em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.name {
		font-weight: 600;
		font-size: 1rem;
		color: var(--text-strong);
		line-height: 1.2;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.amen {
		display: flex;
		flex-wrap: wrap;
		gap: 0.3rem;
		margin-top: 0.25rem;
	}
	.chip-active {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.62rem;
		letter-spacing: 0.18em;
		padding: 0.18rem 0.45rem;
		border-radius: 4px;
		background: rgba(46, 226, 122, 0.12);
		color: var(--accent);
		border: 1px solid rgba(46, 226, 122, 0.3);
	}
	.right {
		grid-column: 3;
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.15rem;
		text-align: right;
		padding-right: 1.1rem;
	}
	.dist {
		display: inline-flex;
		align-items: baseline;
		gap: 0.18rem;
		color: var(--text-strong);
	}
	.dist-num {
		font-size: 1.5rem;
		font-weight: 700;
		line-height: 1;
	}
	.dist-unit {
		font-family: var(--font-display);
		font-size: 0.7rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.eta {
		font-size: 0.78rem;
		color: var(--accent);
		letter-spacing: 0.04em;
	}
	.arrow {
		position: absolute;
		right: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		color: var(--muted-2);
		transition: color 0.2s, transform 0.2s var(--ease-spring);
	}
	.card:hover .arrow {
		color: var(--accent);
		transform: translateY(-50%) translateX(2px);
	}
</style>
