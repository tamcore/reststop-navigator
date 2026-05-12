<script lang="ts">
	import { demo } from '$lib/stores/demo';
	import { DEMO_TRIP_DURATION_MS, DEMO_TRIP_POINTS } from '$lib/stores/geo';

	const durationSec = Math.round(DEMO_TRIP_DURATION_MS / 1000);
	const durationLabel =
		durationSec >= 60
			? `${Math.floor(durationSec / 60)}m${durationSec % 60 ? ` ${durationSec % 60}s` : ''}`
			: `${durationSec}s`;
</script>

{#if $demo}
	<div class="demo-banner" role="status" aria-live="polite">
		<span class="demo-label">DEMO MODE</span>
		<span class="demo-sep">—</span>
		<span class="demo-desc">A3 replay · {DEMO_TRIP_POINTS} pts · {durationLabel} loop</span>
		<button type="button" class="demo-exit" onclick={() => demo.disable()}>exit</button>
	</div>
{/if}

<style>
	.demo-banner {
		max-width: 720px;
		margin: 0 auto;
		padding: 0.45rem 1rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-family: var(--font-display);
		font-size: 0.68rem;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		background: color-mix(in srgb, var(--accent) 10%, transparent);
		border-bottom: 1px solid color-mix(in srgb, var(--accent) 25%, transparent);
		position: relative;
		z-index: 10;
	}
	.demo-label {
		color: var(--accent);
		font-weight: 700;
	}
	.demo-sep {
		color: var(--muted-2);
	}
	.demo-desc {
		color: var(--muted);
		flex: 1;
	}
	.demo-exit {
		background: none;
		border: 1px solid color-mix(in srgb, var(--accent) 30%, transparent);
		border-radius: 4px;
		color: var(--accent);
		font-family: inherit;
		font-size: inherit;
		letter-spacing: inherit;
		text-transform: inherit;
		padding: 0.1rem 0.5rem;
		cursor: pointer;
		transition: background 0.15s;
	}
	.demo-exit:hover {
		background: color-mix(in srgb, var(--accent) 15%, transparent);
	}
	.demo-exit:focus-visible {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
	}
</style>
