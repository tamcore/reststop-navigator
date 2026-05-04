import { writable } from 'svelte/store';
import { ALL_FILTERS, type FilterKey } from '$lib/types/api';

const STORAGE_KEY = 'reststop:filters';

function load(): Set<FilterKey> {
	if (typeof localStorage === 'undefined') return new Set();
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return new Set();
		const arr = JSON.parse(raw) as string[];
		return new Set(arr.filter((k): k is FilterKey => (ALL_FILTERS as readonly string[]).includes(k)));
	} catch {
		return new Set();
	}
}

function persist(set: Set<FilterKey>) {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(STORAGE_KEY, JSON.stringify([...set]));
}

function createFiltersStore() {
	const inner = writable<Set<FilterKey>>(load());
	return {
		subscribe: inner.subscribe,
		toggle(k: FilterKey) {
			inner.update((s) => {
				const next = new Set(s);
				if (next.has(k)) next.delete(k);
				else next.add(k);
				persist(next);
				return next;
			});
		},
		set(s: Set<FilterKey>) {
			persist(s);
			inner.set(new Set(s));
		},
		clear() {
			const empty = new Set<FilterKey>();
			persist(empty);
			inner.set(empty);
		}
	};
}

export const filters = createFiltersStore();
