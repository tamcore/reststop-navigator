// Mirrors internal/stops JSON shapes from the Go backend.

export type AmenityFlags = {
	fuel: boolean;
	charging: boolean;
	food: boolean;
	toilets: boolean;
	open24h: boolean;
	dog: boolean;
};

export type StopInfo = {
	id: string;
	name?: string;
	kind: 'services' | 'rest_area' | string;
	lat: number;
	lon: number;
	distance_m: number;
	eta_seconds: number;
	amenities: AmenityFlags;
	opening_hours?: string;
	operator?: string;
};

export type Road = {
	ref?: string;
	name?: string;
	direction?: string;
};

export type UpcomingResponse = {
	country?: string;
	road?: Road;
	stops: StopInfo[];
	version?: string;
	reason?: string;
};

export type DeepLinks = {
	google: string;
	apple: string;
	waze: string;
};

export type DetailResponse = {
	country: string;
	stop: StopInfo;
	deep_links: DeepLinks;
	tags?: Record<string, string>;
};

export type FilterKey = 'fuel' | 'charging' | 'food' | 'toilets' | 'open24h' | 'dog';

export const ALL_FILTERS: FilterKey[] = ['fuel', 'charging', 'food', 'toilets', 'open24h', 'dog'];
