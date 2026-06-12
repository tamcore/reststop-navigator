import type {
	AdminPositionsResponse,
	AdminStatsResponse,
	AdminTileStopsResponse,
	AdminTilesResponse
} from '$lib/types/admin';
import { ApiError } from './client';

// The browser handles Basic Auth: the first request triggers the native
// credential prompt and the browser re-sends credentials on every call.
async function adminRequest<T>(path: string, signal?: AbortSignal): Promise<T> {
	const res = await fetch(path, { headers: { Accept: 'application/json' }, signal });
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

export function fetchAdminPositions(signal?: AbortSignal): Promise<AdminPositionsResponse> {
	return adminRequest<AdminPositionsResponse>('/api/admin/positions', signal);
}

export function fetchAdminTiles(signal?: AbortSignal): Promise<AdminTilesResponse> {
	return adminRequest<AdminTilesResponse>('/api/admin/tiles', signal);
}

export function fetchAdminTileStops(
	south: number,
	west: number,
	signal?: AbortSignal
): Promise<AdminTileStopsResponse> {
	const q = new URLSearchParams({ south: String(south), west: String(west) });
	return adminRequest<AdminTileStopsResponse>(`/api/admin/tiles/stops?${q.toString()}`, signal);
}

export function fetchAdminStats(signal?: AbortSignal): Promise<AdminStatsResponse> {
	return adminRequest<AdminStatsResponse>('/api/admin/stats', signal);
}
