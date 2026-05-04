import { writable } from 'svelte/store';

export type Theme = 'auto' | 'light' | 'dark';

const COOKIE_NAME = 'reststop-theme';
const TEN_YEARS = 60 * 60 * 24 * 365 * 10;

function readCookie(): Theme {
	if (typeof document === 'undefined') return 'auto';
	const m = document.cookie.match(/(?:^|;\s*)reststop-theme=(auto|light|dark)/);
	return (m?.[1] as Theme) ?? 'auto';
}

function writeCookie(t: Theme) {
	if (typeof document === 'undefined') return;
	const secure = location.protocol === 'https:' ? '; Secure' : '';
	document.cookie = `${COOKIE_NAME}=${t}; max-age=${TEN_YEARS}; path=/; SameSite=Lax${secure}`;
}

function applyAttribute(t: Theme) {
	if (typeof document === 'undefined') return;
	const root = document.documentElement;
	if (t === 'auto') {
		root.removeAttribute('data-theme');
	} else {
		root.setAttribute('data-theme', t);
	}
}

function createThemeStore() {
	const initial = readCookie();
	const inner = writable<Theme>(initial);
	if (typeof document !== 'undefined') applyAttribute(initial);

	return {
		subscribe: inner.subscribe,
		set(t: Theme) {
			writeCookie(t);
			applyAttribute(t);
			inner.set(t);
		}
	};
}

export const theme = createThemeStore();
