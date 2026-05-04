<script lang="ts">
	import { filters } from '$lib/stores/filters';
	import { ALL_FILTERS, type FilterKey } from '$lib/types/api';

	const labels: Record<FilterKey, string> = {
		fuel: 'FUEL',
		charging: 'EV',
		food: 'FOOD',
		toilets: 'WC',
		open24h: '24/7',
		dog: 'DOG'
	};
</script>

<div class="chips" role="group" aria-label="amenity filters">
	{#each ALL_FILTERS as key (key)}
		{@const active = $filters.has(key)}
		<button
			type="button"
			class="chip"
			class:active
			aria-pressed={active}
			onclick={() => filters.toggle(key)}
		>
			<span class="chip-label">{labels[key]}</span>
		</button>
	{/each}
</div>

<style>
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin: 1rem 0 0.5rem;
	}
	.chip {
		padding: 0.45rem 0.8rem 0.4rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: linear-gradient(180deg, var(--surface), var(--bg-elev));
		color: var(--muted);
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.78rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		transition:
			color 0.18s,
			border-color 0.18s,
			background 0.18s,
			box-shadow 0.18s;
		position: relative;
	}
	.chip:hover {
		color: var(--text);
		border-color: var(--border-strong);
	}
	.chip.active {
		color: var(--bg);
		background: linear-gradient(180deg, #34f088, var(--accent-strong));
		border-color: var(--accent-strong);
		box-shadow:
			0 0 0 1px rgba(46, 226, 122, 0.35),
			0 6px 18px -8px rgba(46, 226, 122, 0.55);
	}
	.chip.active::after {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: inherit;
		background: linear-gradient(180deg, rgba(255, 255, 255, 0.3), transparent 50%);
		pointer-events: none;
		mix-blend-mode: overlay;
	}
	.chip-label {
		display: inline-block;
	}
</style>
