import { writable } from 'svelte/store';

const STORAGE_KEY = 'reststop:demo';

function load(): boolean {
	if (typeof localStorage === 'undefined') return false;
	return localStorage.getItem(STORAGE_KEY) === '1';
}

function persist(active: boolean): void {
	if (typeof localStorage === 'undefined') return;
	if (active) localStorage.setItem(STORAGE_KEY, '1');
	else localStorage.removeItem(STORAGE_KEY);
}

function createDemoStore() {
	const inner = writable<boolean>(load());
	return {
		subscribe: inner.subscribe,
		enable(): void {
			inner.set(true);
			persist(true);
		},
		disable(): void {
			inner.set(false);
			persist(false);
		},
		toggle(): void {
			inner.update((v) => {
				const next = !v;
				persist(next);
				return next;
			});
		}
	};
}

export const demo = createDemoStore();
