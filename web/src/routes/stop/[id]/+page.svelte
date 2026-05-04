<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { ApiError, fetchStopDetail } from '$lib/api/client';
	import type { DetailResponse } from '$lib/types/api';

	let detail = $state<DetailResponse | null>(null);
	let error = $state<string | null>(null);
	let loading = $state(true);

	onMount(async () => {
		const id = decodeURIComponent($page.params.id);
		try {
			detail = await fetchStopDetail(id);
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) error = 'Stop not found.';
			else if (err instanceof ApiError) error = err.message;
			else error = 'Network error';
		} finally {
			loading = false;
		}
	});
</script>

{#if loading}
	<p class="muted">Loading…</p>
{:else if error}
	<p class="error">{error}</p>
	<p><a href="/">Back to list</a></p>
{:else if detail}
	{@const s = detail.stop}
	<section>
		<a class="back" href="/">← Back</a>

		<h1>{s.name || 'Rest area'}</h1>
		<p class="muted">{s.kind} · {detail.country}</p>

		<div class="amenities">
			{#if s.amenities.fuel}<span class="badge">⛽ Fuel</span>{/if}
			{#if s.amenities.charging}<span class="badge">🔌 EV</span>{/if}
			{#if s.amenities.food}<span class="badge">🍴 Food</span>{/if}
			{#if s.amenities.toilets}<span class="badge">🚻 Toilets</span>{/if}
			{#if s.amenities.open24h}<span class="badge">⏰ 24/7</span>{/if}
			{#if s.amenities.dog}<span class="badge">🐕 Dog</span>{/if}
		</div>

		{#if s.opening_hours}
			<p><strong>Opening hours:</strong> {s.opening_hours}</p>
		{/if}
		{#if s.operator}
			<p><strong>Operator:</strong> {s.operator}</p>
		{/if}

		<div class="links">
			<a class="btn primary" href={detail.deep_links.google} target="_blank" rel="noopener">
				Google Maps
			</a>
			<a class="btn" href={detail.deep_links.apple} target="_blank" rel="noopener">
				Apple Maps
			</a>
			<a class="btn" href={detail.deep_links.waze} target="_blank" rel="noopener">
				Waze
			</a>
		</div>

		<p class="muted small">
			Google Maps adds the stop as a destination. Waze replaces any active route. Apple Maps
			depends on iOS version.
		</p>
	</section>
{/if}

<style>
	.back {
		color: var(--accent);
		text-decoration: none;
		font-size: 0.9rem;
	}
	h1 {
		font-size: 1.4rem;
		margin: 0.5rem 0 0.25rem;
	}
	.muted {
		color: var(--muted);
	}
	.small {
		font-size: 0.8rem;
	}
	.error {
		color: var(--danger);
	}
	.amenities {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin: 0.75rem 0;
	}
	.badge {
		padding: 0.2rem 0.55rem;
		background: rgba(34, 197, 94, 0.15);
		border-radius: 999px;
		font-size: 0.85rem;
	}
	.links {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	.btn {
		display: block;
		padding: 0.85rem 1rem;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--surface);
		color: var(--text);
		text-decoration: none;
		text-align: center;
		font-weight: 600;
	}
	.btn.primary {
		background: var(--accent-strong);
		border-color: var(--accent);
		color: #fff;
	}
</style>
