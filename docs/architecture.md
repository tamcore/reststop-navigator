# Architecture

A driver opens the PWA. The browser asks for geolocation. The backend identifies which highway and direction they're on, lists the upcoming rest stops filterable by amenity, and hands off to Google Maps / Apple Maps / Waze when one is picked.

This file explains how that works. Pair it with [development.md](development.md) (running it locally) and [deployment.md](deployment.md) (how it ships).

## Components

```
┌────────────────────┐                  ┌──────────────────────┐
│  SvelteKit PWA     │  fetch /api/...  │  Go backend (chi)    │
│  (embedded via     │ ───────────────► │  cmd/server          │
│  //go:embed)       │ ◄─────────────── │                      │
└────────────────────┘                  │  internal/api        │
                                        │    handlers          │
                                        │    middleware        │
                                        │  internal/stops      │
                                        │  internal/geo        │
                                        │  internal/cache      │
                                        │  internal/overpass   │
                                        └──────────┬───────────┘
                                                   │
                                  ┌────────────────┴────────────────┐
                                  │                                  │
                                  ▼                                  ▼
                          ┌─────────────┐               ┌──────────────────────┐
                          │ Redis       │               │ Overpass API         │
                          │ tile cache  │               │ overpass-api.de      │
                          │ 7d TTL      │               │ overpass.kumi.systems│
                          └─────────────┘               └──────────────────────┘
```

- **Frontend** — SvelteKit 2 with the static adapter, Svelte 5 runes. Built once per release and embedded into the Go binary via `//go:embed all:build` (`web/embed_prod.go`), gated on `-tags prodfrontend`.
- **Backend** — Go 1.26, chi router, slog. One binary serves both the embedded SPA and the `/api/...` JSON endpoints.
- **Cache** — Redis only. No Postgres. The cache is rehydratable from Overpass, so persistence is intentionally off in the chart.
- **Data** — OpenStreetMap via the Overpass API. Two endpoints (`overpass-api.de`, `overpass.kumi.systems`) with backoff + failover.

## The single most important data flow: `GET /api/stops/upcoming`

1. **Request** — browser sends `lat`, `lon`, `heading`, `speed`, `filters`, `limit` from `navigator.geolocation.watchPosition`.
2. **Country resolve** — point-in-bbox over four precomputed country bounding boxes (DE / AT / SK / CZ). Off-grid → 204 with reason.
3. **Tile resolve** — geographic tile = `(floor(lat/0.5)*0.5, floor(lon/0.5)*0.5)`. ≈ 55×37 km at 48°N. Single tile per request.
4. **Tile fetch** (`internal/cache/tilecache.go`):
   - Hit Redis at `reststops:tile:{south:.1f}:{west:.1f}` → unmarshal → done.
   - Miss → call Overpass with a bbox query (`internal/overpass/queries.go`), decode (`decode.go`), spatial-join amenities within 350 m of each stop centroid (`enrich.go`), persist with 7-day TTL.
5. **Match the road & direction** (`internal/geo/match.go`):
   - Iterate ways within ~5 km bbox.
   - For each segment: `distancePointToSegment ≤ maxDist` AND `angleDiff(heading, segmentBearing) ≤ 60°` → candidate.
   - `maxDist` is accuracy-aware: `max(80, gpsAccuracy)` capped at 250 m (default 80 m when accuracy not provided). This widens the match radius for early imprecise GPS fixes, tightening as accuracy improves.
   - Pick min-distance candidate. None → `200 { reason: "off-highway-or-wrong-direction" }`.
6. **Stops ahead** (`internal/geo/ahead.go`):
   - Haversine distance from user to each stop on the matched way.
   - Heading-vector dot product filters out stops behind the driver.
   - We deliberately do **not** project onto the way's polyline — stops often live on a different OSM way than the carriageway, and projection clamps to endpoints in that case (this regression is in the git history).
7. **Filter** — drop stops that fail any selected amenity flag (`fuel`, `charging`, `food`, `toilets`, `open24h`, `dog`).
8. **Rank** — by road-distance ascending. Take `limit`.
9. **Respond** — JSON, country + road shield + ranked stops + version + ttl.

## Why no Postgres?

A country's worth of motorway geometry + amenity tags fits comfortably in a few MB of Redis JSON. The data is *fetchable on demand* from Overpass — losing the cache costs ~5 s per tile, not user data. So we ditched Postgres and migrations entirely; the only stateful dependency is Redis, the chart can run with `persistence: false`, and recovery is "wait for the next request to repopulate."

## Direction detection (the non-trivial bit)

OSM models dual-carriageway motorways as **two separate ways**, each `oneway=yes`, traced in the direction of travel. So when we match a `oneway` way, the segment bearing equals the lane direction — heading alignment alone tells us which side of the road the user is on. No "left/right of carriageway" math.

Single-carriageway segments (some rural SK/CZ, AT B-roads) carry both directions on one way. The geo layer handles this by classifying via the dot product of user-velocity and segment-bearing, and could mark the result `direction_uncertain: true`. Stationary/low-speed users skip direction matching entirely and get nearest stops in any direction.

## Frontend at a glance

- **Two routes:** `/` (live list) and `/stop/[id]` (detail + Leaflet map + nav handoff).
- **Stores:** `geo.ts` wraps `watchPosition`, `filters.ts` is localStorage-backed, `theme.ts` is cookie-backed (10-year TTL, three states: `auto` / `light` / `dark`).
- **Polling cadence:** `/api/stops/upcoming` with burst + normal modes:
  - **Burst mode** (first 30 s after GPS goes `live`): 3 s interval for fast initial feedback. Exits early when a highway match is found.
  - **Normal mode**: 15 s while moving (>20 km/h), 60 s otherwise.
  - GPS accuracy is forwarded to the backend so the match radius can widen for imprecise early fixes.
- **GeoStatusPanel:** Shown in the hero area when GPS is live but no highway is matched yet. Displays real-time telemetry (coordinates, accuracy with green/amber/red colour coding, heading as cardinal + degrees, speed) and highway search status. Works identically in demo mode.
- **Detail map:** Leaflet 1.9, OSM tiles, dynamic import for code-splitting. The stop marker is an inline-SVG `divIcon` (no asset dependency — Vite-bundled Leaflet can't resolve `marker-icon.png`).
- **PWA shell:** `manifest.webmanifest` + `display: standalone`. API responses are network-first; last response shown stale-with-banner offline.
- **Deep links:** Google Maps (waypoint-add when a route is active), Apple Maps (open destination), Waze (replaces active route — labelled).

## Design language

- **Type stack:** Antonio (display, condensed for road-shield aesthetic), Geist (body), JetBrains Mono (numerics, distances, ETAs). All self-hosted via `@fontsource`.
- **Palette:** "Verkehrszeichen instrument cluster" — autobahn-green accent (`#2ee27a` / `#16a34a` light), midnight-blue surfaces (dark) or warm bone (light), signage-blue road shield.
- **Theme:** `data-theme` attribute on `<html>` set by an inline script in `app.html` *before* hydration to avoid FOUC. Defaults to OS preference.

## Observability

- All HTTP requests log via `internal/api/middleware/access_log.go` (slog JSON in prod, text in dev): method, path, query, status, bytes, duration, request_id, remote, ua, referer.
- `/api/healthz` returns `{"status":"ok"}`. No `/readyz` yet; pods are ready once they pass the listener probe in the Helm chart.

## Security posture

- **No accounts, no per-user storage, no third-party trackers.**
- CSP: `default-src 'self'` + `script-src 'self' 'unsafe-inline'` (SvelteKit hydration scripts) + `img-src` allows OSM tile servers + `font-src 'self' data:` (@fontsource subsets).
- HSTS only on TLS-backed requests (so plain-HTTP local dev still works).
- Rate-limited at the ingress layer.
- Geolocation is the only browser permission requested.

## Where to read next

- [development.md](development.md) — running the stack locally, tests, the GPX replay tool.
- [deployment.md](deployment.md) — release pipeline, GitOps notes, manual dev deploys.
- The upstream plan lives at `~/.claude/plans/the-problem-on-long-fuzzy-eich.md` — frozen at design time, useful as historical context.
