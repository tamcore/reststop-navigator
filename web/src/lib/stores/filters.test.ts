import { beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

describe('filters store', () => {
	beforeEach(() => {
		vi.resetModules();
		try {
			localStorage.removeItem('reststop:filters');
		} catch {
			// some test environments restrict localStorage; ignore
		}
	});

	it('toggles values and persists to localStorage', async () => {
		const { filters } = await import('./filters');
		filters.clear();
		expect([...get(filters)]).toEqual([]);

		filters.toggle('fuel');
		expect(get(filters).has('fuel')).toBe(true);
		expect(JSON.parse(localStorage.getItem('reststop:filters') ?? '[]')).toEqual(['fuel']);

		filters.toggle('charging');
		expect(new Set(JSON.parse(localStorage.getItem('reststop:filters') ?? '[]'))).toEqual(
			new Set(['fuel', 'charging'])
		);

		filters.toggle('fuel');
		expect(get(filters).has('fuel')).toBe(false);
		expect(JSON.parse(localStorage.getItem('reststop:filters') ?? '[]')).toEqual(['charging']);
	});

	it('clear empties the store and storage', async () => {
		const { filters } = await import('./filters');
		filters.toggle('fuel');
		filters.toggle('food');
		filters.clear();
		expect([...get(filters)]).toEqual([]);
		expect(JSON.parse(localStorage.getItem('reststop:filters') ?? '[]')).toEqual([]);
	});

	it('set replaces the entire selection', async () => {
		const { filters } = await import('./filters');
		filters.set(new Set(['toilets', 'open24h']));
		expect(new Set(get(filters))).toEqual(new Set(['toilets', 'open24h']));
	});
});
