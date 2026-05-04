<script lang="ts">
	import { filters } from '$lib/stores/filters';
	import { ALL_FILTERS, type FilterKey } from '$lib/types/api';

	const labels: Record<FilterKey, string> = {
		fuel: 'Fuel',
		charging: 'EV',
		food: 'Food',
		toilets: 'WC',
		open24h: '24/7',
		dog: 'Dog'
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
			{labels[key]}
		</button>
	{/each}
</div>

<style>
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin: 0.5rem 0 1rem;
	}
	.chip {
		padding: 0.4rem 0.75rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--muted);
		font-size: 0.85rem;
	}
	.chip.active {
		border-color: var(--accent);
		background: var(--accent-strong);
		color: #fff;
	}
</style>
