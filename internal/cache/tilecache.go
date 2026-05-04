package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// tileSizeDeg is the side length of one tile in degrees. ~111 km lat × ~74 km
// lon at 48N — large enough to amortize Overpass calls, small enough that each
// query fits comfortably under the server's 180s timeout.
const tileSizeDeg = 1.0

// OverpassFetcher is the subset of *overpass.Client that TileCache needs.
type OverpassFetcher interface {
	Query(ctx context.Context, q string) ([]byte, error)
}

// TileCache stores per-tile datasets in Redis. On miss it fetches from
// Overpass for that tile only and writes the result back.
type TileCache struct {
	rdb     *redis.Client
	client  OverpassFetcher
	ttl     time.Duration
	missTTL time.Duration
}

// TileOption configures NewTileCache.
type TileOption func(*TileCache)

// WithTileTTL overrides the per-tile TTL (default 7 days).
func WithTileTTL(d time.Duration) TileOption { return func(c *TileCache) { c.ttl = d } }

// WithTileMissTTL sets a shorter TTL for empty tiles (default 1 day).
func WithTileMissTTL(d time.Duration) TileOption { return func(c *TileCache) { c.missTTL = d } }

// NewTileCache builds a TileCache.
func NewTileCache(rdb *redis.Client, client OverpassFetcher, opts ...TileOption) *TileCache {
	c := &TileCache{
		rdb:     rdb,
		client:  client,
		ttl:     7 * 24 * time.Hour,
		missTTL: 24 * time.Hour,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Tile is a 1° × 1° geographic cell, identified by its south-west corner.
type Tile struct{ South, West float64 }

// TileFor returns the tile containing pos.
func TileFor(pos geo.LatLng) Tile {
	return Tile{
		South: math.Floor(pos.Lat/tileSizeDeg) * tileSizeDeg,
		West:  math.Floor(pos.Lon/tileSizeDeg) * tileSizeDeg,
	}
}

// TilesAround returns the 4 tiles closest to pos (the tile pos is in plus its
// 3 neighbours toward whichever quadrant pos lies in). This way the matcher
// always has data for the next ~80–100 km of motorway regardless of which
// way the user is heading.
func TilesAround(pos geo.LatLng) []Tile {
	t := TileFor(pos)
	dLat, dLon := tileSizeDeg, tileSizeDeg
	if pos.Lat-t.South < tileSizeDeg/2 {
		dLat = -tileSizeDeg
	}
	if pos.Lon-t.West < tileSizeDeg/2 {
		dLon = -tileSizeDeg
	}
	return []Tile{
		t,
		{South: t.South + dLat, West: t.West},
		{South: t.South, West: t.West + dLon},
		{South: t.South + dLat, West: t.West + dLon},
	}
}

// BBox converts the tile to an Overpass bbox.
func (t Tile) BBox() overpass.BBox {
	return overpass.BBox{
		South: t.South,
		West:  t.West,
		North: t.South + tileSizeDeg,
		East:  t.West + tileSizeDeg,
	}
}

// Get returns the cached or freshly-fetched dataset for tile.
func (c *TileCache) Get(ctx context.Context, t Tile) (overpass.Dataset, error) {
	key := tileKey(t)
	payload, err := c.rdb.Get(ctx, key).Bytes()
	switch {
	case err == nil:
		var ds overpass.Dataset
		if jsonErr := json.Unmarshal(payload, &ds); jsonErr == nil {
			return ds, nil
		}
		// fall through and refetch on decode errors.
	case errors.Is(err, redis.Nil):
		// fall through to fetch.
	default:
		return overpass.Dataset{}, fmt.Errorf("tilecache: redis get: %w", err)
	}

	raw, err := c.client.Query(ctx, overpass.BBoxQuery(t.BBox()))
	if err != nil {
		return overpass.Dataset{}, fmt.Errorf("tilecache: overpass: %w", err)
	}
	ds, err := overpass.Decode(raw)
	if err != nil {
		return overpass.Dataset{}, fmt.Errorf("tilecache: decode: %w", err)
	}
	overpass.EnrichDataset(&ds)

	ttl := c.ttl
	if len(ds.Ways) == 0 && len(ds.Stops) == 0 {
		ttl = c.missTTL
	}
	if buf, marshalErr := json.Marshal(ds); marshalErr == nil {
		_ = c.rdb.Set(ctx, key, buf, ttl).Err()
	}
	return ds, nil
}

// GetMerged returns the union of ds.{Ways,Stops,Amenities} across the 4 tiles
// that pos's neighbourhood spans. Tile fetches run concurrently so a cold
// 4-tile request takes the latency of one Overpass call, not four.
func (c *TileCache) GetMerged(ctx context.Context, pos geo.LatLng) (overpass.Dataset, error) {
	tiles := TilesAround(pos)
	results := make([]overpass.Dataset, len(tiles))
	errs := make([]error, len(tiles))

	var wg sync.WaitGroup
	for i, t := range tiles {
		i, t := i, t
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = c.Get(ctx, t)
		}()
	}
	wg.Wait()

	merged := overpass.Dataset{}
	gotData := false
	var lastErr error
	for i := range tiles {
		if errs[i] != nil {
			lastErr = errs[i]
			continue
		}
		gotData = true
		merged.Ways = append(merged.Ways, results[i].Ways...)
		merged.Stops = append(merged.Stops, results[i].Stops...)
		merged.Amenities = append(merged.Amenities, results[i].Amenities...)
	}
	if !gotData && lastErr != nil {
		return overpass.Dataset{}, lastErr
	}
	return merged, nil
}

func tileKey(t Tile) string {
	return fmt.Sprintf("reststops:tile:%.1f:%.1f", t.South, t.West)
}
