// Mirrors the /api/admin JSON shapes from the Go backend
// (internal/api/handlers/admin.go, internal/cache/snapshot.go,
// internal/presence/tracker.go).

export type AdminClient = {
	client_id: string;
	lat: number;
	lon: number;
	heading: number;
	speed: number;
	accuracy: number;
	last_seen: string;
};

export type AdminPositionsResponse = {
	clients: AdminClient[];
	count: number;
};

export type AdminTileInfo = {
	key: string;
	south: number;
	west: number;
	size_deg: number;
	stops: number;
	ways: number;
	amenities: number;
	bytes: number;
	ttl_seconds: number;
};

export type AdminTilesResponse = {
	tiles: AdminTileInfo[];
};

export type AdminTileStop = {
	osm_type: string;
	osm_id: number;
	kind: string;
	pos: { lat: number; lon: number };
	name?: string;
	amenities: {
		fuel: boolean;
		charging: boolean;
		food: boolean;
		toilets: boolean;
		open24h: boolean;
		dog: boolean;
	};
	highway_ref?: string;
};

export type AdminTileStopsResponse = {
	stops: AdminTileStop[];
};

export type AdminStatsResponse = {
	uptime_seconds: number;
	cache: { hits: number; misses: number };
	presence_count: number;
	redis?: { keys: number; used_memory_bytes: number };
};
