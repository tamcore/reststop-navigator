<script lang="ts">
	import { theme, type Theme } from '$lib/stores/theme';

	const options: { value: Theme; label: string; aria: string }[] = [
		{ value: 'light', label: 'L', aria: 'Light theme' },
		{ value: 'auto', label: 'A', aria: 'Match system theme' },
		{ value: 'dark', label: 'D', aria: 'Dark theme' }
	];
</script>

<div class="seg" role="group" aria-label="Theme">
	{#each options as o (o.value)}
		<button
			type="button"
			class="opt"
			class:active={$theme === o.value}
			aria-pressed={$theme === o.value}
			aria-label={o.aria}
			title={o.aria}
			onclick={() => theme.set(o.value)}
		>
			{o.label}
		</button>
	{/each}
</div>

<style>
	.seg {
		display: inline-flex;
		padding: 2px;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--bg-elev);
	}
	.opt {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.7rem;
		letter-spacing: 0.18em;
		color: var(--muted);
		padding: 0.2rem 0.55rem;
		border-radius: 999px;
		transition:
			color 0.18s,
			background 0.18s;
	}
	.opt:hover {
		color: var(--text);
	}
	.opt.active {
		background: var(--accent);
		color: var(--bg);
	}
</style>
