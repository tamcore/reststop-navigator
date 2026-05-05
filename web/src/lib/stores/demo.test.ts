import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

const STORAGE_KEY = 'reststop:demo';

describe('demo store', () => {
	beforeEach(() => {
		vi.resetModules();
		localStorage.clear();
	});

	afterEach(() => {
		localStorage.clear();
	});

	it('defaults to false when localStorage is empty', async () => {
		const { demo } = await import('./demo');
		expect(get(demo)).toBe(false);
	});

	it('loads true when localStorage has "1"', async () => {
		localStorage.setItem(STORAGE_KEY, '1');
		const { demo } = await import('./demo');
		expect(get(demo)).toBe(true);
	});

	it('defaults to false when localStorage has an unexpected value', async () => {
		localStorage.setItem(STORAGE_KEY, 'yes');
		const { demo } = await import('./demo');
		expect(get(demo)).toBe(false);
	});

	it('enable sets state to true and persists', async () => {
		const { demo } = await import('./demo');
		demo.enable();
		expect(get(demo)).toBe(true);
		expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
	});

	it('enable is idempotent', async () => {
		const { demo } = await import('./demo');
		demo.enable();
		demo.enable();
		expect(get(demo)).toBe(true);
		expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
	});

	it('disable sets state to false and removes the key', async () => {
		localStorage.setItem(STORAGE_KEY, '1');
		const { demo } = await import('./demo');
		demo.disable();
		expect(get(demo)).toBe(false);
		expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
	});

	it('disable when already false is a no-op', async () => {
		const { demo } = await import('./demo');
		demo.disable();
		expect(get(demo)).toBe(false);
		expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
	});

	it('toggle flips false → true and persists', async () => {
		const { demo } = await import('./demo');
		demo.toggle();
		expect(get(demo)).toBe(true);
		expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
	});

	it('toggle flips true → false and removes key', async () => {
		localStorage.setItem(STORAGE_KEY, '1');
		const { demo } = await import('./demo');
		demo.toggle();
		expect(get(demo)).toBe(false);
		expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
	});
});

describe('demo store — SSR (no localStorage)', () => {
	let saved: Storage;

	beforeEach(() => {
		vi.resetModules();
		saved = globalThis.localStorage;
		// @ts-expect-error simulate SSR environment without localStorage
		delete globalThis.localStorage;
	});

	afterEach(() => {
		Object.defineProperty(globalThis, 'localStorage', { value: saved, configurable: true });
	});

	it('defaults to false when localStorage is unavailable', async () => {
		const { demo } = await import('./demo');
		expect(get(demo)).toBe(false);
	});

	it('enable does not throw when localStorage is unavailable', async () => {
		const { demo } = await import('./demo');
		expect(() => demo.enable()).not.toThrow();
		expect(get(demo)).toBe(true);
	});

	it('disable does not throw when localStorage is unavailable', async () => {
		const { demo } = await import('./demo');
		demo.enable();
		expect(() => demo.disable()).not.toThrow();
		expect(get(demo)).toBe(false);
	});

	it('toggle does not throw when localStorage is unavailable', async () => {
		const { demo } = await import('./demo');
		expect(() => demo.toggle()).not.toThrow();
		expect(get(demo)).toBe(true);
	});
});
