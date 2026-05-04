<script lang="ts">
	import '../styles/global.css';
	let { children } = $props();
</script>

<header class="app-header">
	<a href="/" class="brand">
		<span class="brand-mark">▣</span>
		<span class="brand-name">RESTSTOP NAVIGATOR</span>
	</a>
</header>

<main>
	{@render children?.()}
</main>

<footer class="app-footer">
	<span>OSM data</span>
	<span class="dot">·</span>
	<span>DE / AT / SK / CZ</span>
</footer>

<style>
	.app-header {
		position: sticky;
		top: 0;
		padding: 0.75rem 1rem;
		background: linear-gradient(
			180deg,
			rgba(6, 9, 18, 0.95) 0%,
			rgba(6, 9, 18, 0.85) 70%,
			rgba(6, 9, 18, 0)
		);
		backdrop-filter: blur(8px) saturate(140%);
		-webkit-backdrop-filter: blur(8px) saturate(140%);
		z-index: 20;
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--text);
		text-decoration: none;
	}
	.brand-mark {
		color: var(--accent);
		font-size: 1rem;
		line-height: 1;
		filter: drop-shadow(0 0 6px var(--accent-glow));
	}
	.brand-name {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.85rem;
		letter-spacing: 0.28em;
		color: var(--text-strong);
	}
	main {
		max-width: 720px;
		margin: 0 auto;
		padding: 0.5rem 1rem 4rem;
		position: relative;
		z-index: 1;
	}
	.app-footer {
		max-width: 720px;
		margin: 0 auto;
		padding: 1.5rem 1rem 2rem;
		font-family: var(--font-display);
		font-size: 0.7rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--muted-2);
		display: flex;
		gap: 0.5rem;
		justify-content: center;
		position: relative;
		z-index: 1;
	}
	.dot {
		color: var(--accent);
	}
</style>
