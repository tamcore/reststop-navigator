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
	filters?: FilterKey[];
	limit?: number;
	signal?: AbortSignal;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, { ...init, headers: { Accept: 'application/json' } });
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
	if (params.filters && params.filters.length > 0) q.set('filters', params.filters.join(','));
	if (params.limit) q.set('limit', String(params.limit));
	return request<UpcomingResponse>(`/api/stops/upcoming?${q.toString()}`, { signal: params.signal });
}

export function fetchStopDetail(
	id: string,
	pos: { lat: number; lon: number },
	signal?: AbortSignal
): Promise<DetailResponse> {
	const q = new URLSearchParams({ id, lat: String(pos.lat), lon: String(pos.lon) });
	return request<DetailResponse>(`/api/stops/detail?${q.toString()}`, { signal });
}
