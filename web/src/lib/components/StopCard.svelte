<script lang="ts">
	import type { StopInfo } from '$lib/types/api';

	type Props = { stop: StopInfo };
	let { stop }: Props = $props();

	function distanceLabel(m: number): string {
		if (m < 1000) return `${m} m`;
		return `${(m / 1000).toFixed(1)} km`;
	}

	function etaLabel(s: number): string {
		if (s < 60) return `${s} s`;
		const min = Math.round(s / 60);
		return `${min} min`;
	}

	const amenityIcons: { key: keyof typeof stop.amenities; label: string }[] = [
		{ key: 'fuel', label: '⛽' },
		{ key: 'charging', label: '🔌' },
		{ key: 'food', label: '🍴' },
		{ key: 'toilets', label: '🚻' },
		{ key: 'open24h', label: '24h' },
		{ key: 'dog', label: '🐕' }
	];
</script>

<a
	class="card"
	href="/stop/{encodeURIComponent(stop.id)}?lat={stop.lat}&lon={stop.lon}"
>
	<div class="row1">
		<span class="name">{stop.name || 'Rest area'}</span>
		<span class="dist">{distanceLabel(stop.distance_m)}</span>
	</div>
	<div class="row2">
		<span class="kind">{stop.kind}</span>
		<span class="eta">{etaLabel(stop.eta_seconds)}</span>
	</div>
	<div class="amenities">
		{#each amenityIcons as a (a.key)}
			{#if stop.amenities[a.key]}
				<span class="badge">{a.label}</span>
			{/if}
		{/each}
	</div>
</a>

<style>
	.card {
		display: block;
		padding: 0.85rem 1rem;
		margin-bottom: 0.5rem;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 12px;
		color: var(--text);
		text-decoration: none;
	}
	.row1,
	.row2 {
		display: flex;
		justify-content: space-between;
		gap: 0.5rem;
	}
	.name {
		font-weight: 600;
	}
	.dist {
		color: var(--accent);
		font-variant-numeric: tabular-nums;
	}
	.kind,
	.eta {
		color: var(--muted);
		font-size: 0.85rem;
	}
	.amenities {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		margin-top: 0.4rem;
	}
	.badge {
		padding: 0.15rem 0.5rem;
		background: rgba(34, 197, 94, 0.15);
		border-radius: 999px;
		font-size: 0.8rem;
	}
</style>
