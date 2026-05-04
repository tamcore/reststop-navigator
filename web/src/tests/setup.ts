import '@testing-library/jest-dom/vitest';

// Node 22+ ships an experimental localStorage global without setItem unless
// --localstorage-file is set. Replace it with a simple in-memory polyfill so
// jsdom-environment tests behave like a browser.
class InMemoryStorage {
	private store = new Map<string, string>();
	get length() {
		return this.store.size;
	}
	clear() {
		this.store.clear();
	}
	getItem(k: string) {
		return this.store.has(k) ? (this.store.get(k) as string) : null;
	}
	setItem(k: string, v: string) {
		this.store.set(k, String(v));
	}
	removeItem(k: string) {
		this.store.delete(k);
	}
	key(i: number) {
		return Array.from(this.store.keys())[i] ?? null;
	}
}

Object.defineProperty(globalThis, 'localStorage', {
	value: new InMemoryStorage(),
	configurable: true,
	writable: true
});
