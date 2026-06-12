import type { DetailResponse, FilterKey, UpcomingResponse } from '$lib/types/api';

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.status = status;
		this.name = 'ApiError';
	}
}

export type UpcomingParams = {
	lat: number;
	lon: number;
	heading?: number;
	speed?: number;
	accuracy?: number;
	filters?: FilterKey[];
	limit?: number;
	signal?: AbortSignal;
};

const CLIENT_ID_STORAGE_KEY = 'reststop:client-id';

let ephemeralClientId: string | null = null;

// crypto.randomUUID only exists in secure contexts (HTTPS/localhost); fall
// back to getRandomValues so plain-HTTP deployments keep working.
function generateUUID(): string {
	if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
	const bytes = new Uint8Array(16);
	crypto.getRandomValues(bytes);
	bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 10
	const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
	return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

/**
 * Returns this browser's anonymous client id (random UUID), persisted in
 * localStorage. Falls back to an in-memory id when storage is unavailable
 * (e.g. private mode). Sent as X-Client-Id so the admin live view can show
 * active clients; it carries no identity.
 */
export function getClientId(): string {
	try {
		const stored = localStorage.getItem(CLIENT_ID_STORAGE_KEY);
		if (stored) return stored;
		const id = generateUUID();
		localStorage.setItem(CLIENT_ID_STORAGE_KEY, id);
		return id;
	} catch {
		if (!ephemeralClientId) ephemeralClientId = generateUUID();
		return ephemeralClientId;
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, { ...init, headers: { Accept: 'application/json', ...init?.headers } });
	if (!res.ok) {
		let msg = res.statusText;
		try {
			const body = (await res.json()) as { error?: string };
			if (body?.error) msg = body.error;
		} catch {
			// body wasn't JSON; keep statusText
		}
		throw new ApiError(res.status, msg);
	}
	return (await res.json()) as T;
}

export function fetchUpcoming(params: UpcomingParams): Promise<UpcomingResponse> {
	const q = new URLSearchParams();
	q.set('lat', String(params.lat));
	q.set('lon', String(params.lon));
	if (params.heading !== undefined) q.set('heading', String(params.heading));
	if (params.speed !== undefined) q.set('speed', String(params.speed));
	if (params.accuracy !== undefined && params.accuracy > 0) q.set('accuracy', String(params.accuracy));
	if (params.filters && params.filters.length > 0) q.set('filters', params.filters.join(','));
	if (params.limit) q.set('limit', String(params.limit));
	return request<UpcomingResponse>(`/api/stops/upcoming?${q.toString()}`, {
		signal: params.signal,
		headers: { 'X-Client-Id': getClientId() }
	});
}

export function fetchStopDetail(
	id: string,
	pos: { lat: number; lon: number },
	signal?: AbortSignal
): Promise<DetailResponse> {
	const q = new URLSearchParams({ id, lat: String(pos.lat), lon: String(pos.lon) });
	return request<DetailResponse>(`/api/stops/detail?${q.toString()}`, { signal });
}
