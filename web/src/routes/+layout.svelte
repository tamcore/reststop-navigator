<script lang="ts">
	import '../styles/global.css';
	let { children } = $props();
</script>

<header class="app-header">
	<a href="/" class="brand">Reststop Navigator</a>
</header>

<main>
	{@render children?.()}
</main>

<style>
	:global(html, body) {
		margin: 0;
		padding: 0;
		font-family: system-ui, -apple-system, 'Segoe UI', sans-serif;
		background: #0b1220;
		color: #e6edf3;
		min-height: 100dvh;
	}

	.app-header {
		position: sticky;
		top: 0;
		padding: 0.75rem 1rem;
		background: #0b1220;
		border-bottom: 1px solid #1f2937;
		z-index: 10;
	}

	.brand {
		color: #e6edf3;
		font-weight: 700;
		text-decoration: none;
		font-size: 1.05rem;
	}

	main {
		max-width: 720px;
		margin: 0 auto;
		padding: 1rem;
	}
</style>
