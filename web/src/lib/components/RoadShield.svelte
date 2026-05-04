<script lang="ts">
	type Props = { ref: string; direction: string; name?: string };
	let { ref, direction, name = '' }: Props = $props();

	const cardinalToBearing: Record<string, number> = {
		N: 0,
		NE: 45,
		E: 90,
		SE: 135,
		S: 180,
		SW: 225,
		W: 270,
		NW: 315
	};
	let arrowAngle = $derived(cardinalToBearing[direction] ?? 0);
</script>

<div class="shield">
	<span class="kategorie">AUTOBAHN</span>
	<div class="badge">
		<span class="ref">{ref}</span>
	</div>
	<div class="meta">
		<div class="heading">
			<svg
				viewBox="0 0 24 24"
				width="14"
				height="14"
				style="transform: rotate({arrowAngle}deg)"
				aria-hidden="true"
			>
				<path d="M12 2 L18 14 L12 11 L6 14 Z" fill="currentColor" />
			</svg>
			<span>{direction || '?'}</span>
		</div>
		{#if name}
			<div class="route" title={name}>{name}</div>
		{/if}
	</div>
</div>

<style>
	.shield {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.4rem;
		position: relative;
	}
	.kategorie {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.65rem;
		letter-spacing: 0.32em;
		color: #79c2ff;
		text-transform: uppercase;
	}
	.badge {
		--blue: #003c7c;
		--blue-edge: #1466bf;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 92px;
		padding: 0.45rem 0.95rem 0.5rem;
		background: linear-gradient(180deg, var(--blue-edge), var(--blue));
		border-radius: 6px;
		border: 2px solid #ffffff;
		box-shadow:
			inset 0 1px 0 rgba(255, 255, 255, 0.35),
			inset 0 -2px 0 rgba(0, 0, 0, 0.35),
			0 6px 14px -4px rgba(0, 60, 124, 0.6);
	}
	.ref {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 1.55rem;
		letter-spacing: 0.05em;
		color: #ffffff;
		line-height: 1;
		text-shadow: 0 1px 0 rgba(0, 0, 0, 0.35);
	}
	.meta {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.heading {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.85rem;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--accent);
		animation: pulse-chevron 1.6s infinite var(--ease-out);
	}
	.route {
		font-size: 0.78rem;
		color: var(--muted);
		max-width: 22ch;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
