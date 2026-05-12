<script lang="ts">
	import { geo, kmh } from '$lib/stores/geo';
	import { stopsPoller } from '$lib/stores/stops';
	import FilterChips from '$lib/components/FilterChips.svelte';
	import StopCard from '$lib/components/StopCard.svelte';
	import RoadShield from '$lib/components/RoadShield.svelte';
	import GeoStatusPanel from '$lib/components/GeoStatusPanel.svelte';
</script>

<section class="hero">
	{#if $geo.status === 'live' && $stopsPoller.road}
		<RoadShield ref={$stopsPoller.road.ref ?? ''} direction={$stopsPoller.road.direction ?? ''} name={$stopsPoller.road.name ?? ''} />
		<div class="speed">
			<span class="speed-num mono">{kmh($geo.speed).toFixed(0)}</span>
			<span class="speed-unit">km/h</span>
		</div>
	{:else if $geo.status === 'live' && !$stopsPoller.road}
		<GeoStatusPanel geo={$geo} reason={$stopsPoller.reason} />
	{:else if $geo.status === 'idle' || $geo.status === 'pending'}
		<div class="hero-empty">
			<span class="hero-empty-label">Acquiring GPS</span>
			<span class="hero-empty-msg">Waiting for satellite fix…</span>
			<div class="dots"><i></i><i></i><i></i></div>
		</div>
	{:else if $geo.status === 'permission-denied'}
		<div class="hero-empty error">
			<span class="hero-empty-label">Location denied</span>
			<span class="hero-empty-msg">
				This app needs your location to find stops ahead. Allow it in your browser.
			</span>
		</div>
	{:else if $geo.status === 'unavailable'}
		<div class="hero-empty error">
			<span class="hero-empty-label">Location unavailable</span>
			<span class="hero-empty-msg">No geolocation on this device.</span>
		</div>
	{/if}
</section>

<FilterChips />

<div class="section-label">Next stops</div>

{#if $stopsPoller.lastError}
	<div class="error-panel" role="alert">
		<div class="error-icon">⚠</div>
		<div class="error-body">
			<div class="error-title">Connection problem</div>
			<p class="error-msg">{$stopsPoller.lastError}</p>
			{#if $stopsPoller.errorCount > 1}
				<p class="error-count">{$stopsPoller.errorCount} consecutive failures — retrying automatically…</p>
			{:else}
				<p class="error-count">Retrying automatically…</p>
			{/if}
		</div>
	</div>
{/if}

{#if $stopsPoller.reason === 'outside-supported-area'}
	<div class="info-panel">
		<div class="info-title">Outside coverage</div>
		<p>This MVP only tracks rest stops in 🇩🇪 Germany, 🇦🇹 Austria, 🇸🇰 Slovakia and 🇨🇿 Czechia.</p>
	</div>
{:else if $stopsPoller.reason === 'off-highway-or-wrong-direction'}
	<div class="info-panel">
		<div class="info-title">Waiting for motorway match</div>
		<p>We only track motorways (Autobahnen). Keep driving until you're on one of these:</p>
		<ul class="road-list">
			<li>
				<span class="flag">🇩🇪</span>
				<span class="cat">Bundesautobahn</span>
				<span class="prefix">A&nbsp;1 – A&nbsp;995</span>
			</li>
			<li>
				<span class="flag">🇦🇹</span>
				<span class="cat">Autobahn</span>
				<span class="prefix">A&nbsp;1 – A&nbsp;26</span>
			</li>
			<li>
				<span class="flag">🇸🇰</span>
				<span class="cat">Diaľnica</span>
				<span class="prefix">D&nbsp;1 – D&nbsp;4</span>
			</li>
			<li>
				<span class="flag">🇨🇿</span>
				<span class="cat">Dálnice</span>
				<span class="prefix">D&nbsp;1 – D&nbsp;56</span>
			</li>
		</ul>
		<p class="hint">Schnellstraßen / Bundesstraßen / city expressways aren't covered yet.</p>
	</div>
{:else if $stopsPoller.stops.length === 0 && !$stopsPoller.loading && $geo.status === 'live'}
	<p class="muted">No upcoming stops match your filters.</p>
{/if}

<ol class="stop-list">
	{#each $stopsPoller.stops as stop, i (stop.id)}
		<li style="--i: {i}">
			<StopCard {stop} />
		</li>
	{/each}
</ol>

<style>
	.hero {
		position: relative;
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1rem;
		padding: 1.25rem 1rem 1.5rem;
		margin: 0.5rem 0 0.5rem;
		border-radius: 20px;
		background:
			radial-gradient(120% 100% at 0% 0%, rgba(46, 226, 122, 0.08), transparent 60%),
			linear-gradient(180deg, var(--bg-elev) 0%, var(--surface) 100%);
		border: 1px solid var(--border);
		box-shadow: 0 24px 60px -30px rgba(46, 226, 122, 0.25);
		overflow: hidden;
	}
	.hero::before {
		content: '';
		position: absolute;
		inset: 0;
		background-image: repeating-linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.025) 0,
			rgba(255, 255, 255, 0.025) 1px,
			transparent 1px,
			transparent 22px
		);
		pointer-events: none;
		mask-image: linear-gradient(180deg, black 30%, transparent 100%);
	}
	.speed {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		line-height: 1;
		position: relative;
	}
	.speed-num {
		font-size: 2.4rem;
		font-weight: 700;
		color: var(--text-strong);
		letter-spacing: -0.02em;
	}
	.speed-unit {
		font-family: var(--font-display);
		font-size: 0.75rem;
		letter-spacing: 0.2em;
		text-transform: uppercase;
		color: var(--muted);
		margin-top: 0.25rem;
	}

	.hero-empty {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		padding: 0.5rem 0;
	}
	.hero-empty-label {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.95rem;
		letter-spacing: 0.24em;
		color: var(--accent);
		text-transform: uppercase;
	}
	.hero-empty.error .hero-empty-label {
		color: var(--danger);
	}
	.hero-empty-msg {
		color: var(--muted);
		font-size: 0.9rem;
		max-width: 28ch;
	}

	.dots {
		display: inline-flex;
		gap: 6px;
		margin-top: 0.25rem;
	}
	.dots i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--accent);
		opacity: 0.4;
		animation: blink 1.2s infinite var(--ease-out);
	}
	.dots i:nth-child(2) {
		animation-delay: 0.15s;
	}
	.dots i:nth-child(3) {
		animation-delay: 0.3s;
	}
	@keyframes blink {
		0%,
		100% {
			opacity: 0.25;
		}
		40% {
			opacity: 1;
		}
	}

	.muted {
		color: var(--muted);
		padding: 0.25rem 0.25rem 0.5rem;
	}
	.error {
		color: var(--danger);
		padding: 0.25rem 0.25rem 0.5rem;
	}

	.info-panel {
		padding: 1rem 1.1rem 1.1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		background:
			radial-gradient(120% 100% at 0% 0%, rgba(77, 124, 255, 0.08), transparent 60%),
			var(--surface);
		margin-bottom: 0.75rem;
	}
	.info-title {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.78rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--cool);
		margin-bottom: 0.5rem;
	}
	.info-panel p {
		margin: 0 0 0.5rem;
		color: var(--text);
		font-size: 0.9rem;
		line-height: 1.45;
	}
	.info-panel .hint {
		color: var(--muted);
		font-size: 0.8rem;
		margin-top: 0.6rem;
		margin-bottom: 0;
	}
	.road-list {
		list-style: none;
		padding: 0;
		margin: 0.4rem 0 0;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.4rem 0.75rem;
	}
	.road-list li {
		display: grid;
		grid-template-columns: 1.5rem 1fr;
		grid-template-rows: auto auto;
		column-gap: 0.5rem;
		align-items: center;
		padding: 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--bg-elev);
	}
	.flag {
		grid-row: 1 / span 2;
		font-size: 1.15rem;
		line-height: 1;
	}
	.cat {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.78rem;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--text-strong);
	}
	.prefix {
		font-family: var(--font-mono);
		font-size: 0.72rem;
		color: var(--accent);
	}
	.stop-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.stop-list li {
		opacity: 0;
		animation: rise 0.36s var(--ease-spring) forwards;
		animation-delay: calc(var(--i) * 50ms);
	}

	.error-panel {
		display: flex;
		gap: 0.75rem;
		padding: 1rem 1.1rem;
		border: 1px solid color-mix(in srgb, var(--danger) 40%, transparent);
		border-radius: var(--radius-card);
		background:
			radial-gradient(120% 100% at 0% 0%, rgba(239, 68, 68, 0.1), transparent 60%),
			var(--surface);
		margin-bottom: 0.75rem;
		animation: shake 0.4s ease-out;
	}
	.error-icon {
		font-size: 1.3rem;
		line-height: 1;
		flex-shrink: 0;
	}
	.error-body {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.error-title {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.78rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--danger);
	}
	.error-msg {
		margin: 0;
		color: var(--text);
		font-size: 0.85rem;
		line-height: 1.4;
	}
	.error-count {
		margin: 0;
		color: var(--muted);
		font-size: 0.78rem;
	}
	@keyframes shake {
		0%, 100% { transform: translateX(0); }
		20% { transform: translateX(-4px); }
		40% { transform: translateX(4px); }
		60% { transform: translateX(-2px); }
		80% { transform: translateX(2px); }
	}
</style>
