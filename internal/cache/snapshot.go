package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// TileInfo describes one cached tile for the admin view.
type TileInfo struct {
	Key        string  `json:"key"`
	South      float64 `json:"south"`
	West       float64 `json:"west"`
	SizeDeg    float64 `json:"size_deg"`
	Stops      int     `json:"stops"`
	Ways       int     `json:"ways"`
	Amenities  int     `json:"amenities"`
	Bytes      int64   `json:"bytes"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

// CacheStats are process-lifetime tile cache counters.
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
}

// Stats returns the hit/miss counters accumulated since process start.
func (c *TileCache) Stats() CacheStats {
	return CacheStats{Hits: c.hits.Load(), Misses: c.misses.Load()}
}

// Snapshot lists all cached tiles with their contents summary. Corrupt or
// concurrently-expired entries are skipped so the admin view stays usable.
func (c *TileCache) Snapshot(ctx context.Context) ([]TileInfo, error) {
	infos := []TileInfo{}
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, "reststops:tile:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("tilecache: snapshot scan: %w", err)
		}
		for _, key := range keys {
			info, err := c.tileInfo(ctx, key)
			if err != nil {
				slog.Warn("tilecache: snapshot skipping entry", "key", key, "error", err)
				continue
			}
			infos = append(infos, info)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return infos, nil
}

func (c *TileCache) tileInfo(ctx context.Context, key string) (TileInfo, error) {
	payload, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return TileInfo{}, fmt.Errorf("get: %w", err)
	}
	var ds overpass.Dataset
	if err := json.Unmarshal(payload, &ds); err != nil {
		return TileInfo{}, fmt.Errorf("decode: %w", err)
	}
	ttl, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		return TileInfo{}, fmt.Errorf("ttl: %w", err)
	}
	south, west := parseTileKey(key)
	return TileInfo{
		Key:        key,
		South:      south,
		West:       west,
		SizeDeg:    tileSizeDeg,
		Stops:      len(ds.Stops),
		Ways:       len(ds.Ways),
		Amenities:  len(ds.Amenities),
		Bytes:      int64(len(payload)),
		TTLSeconds: int64(ttl.Seconds()),
	}, nil
}

// parseTileKey extracts the south/west corner from a tile key. Unknown key
// layouts yield zeros, which the admin UI renders as "unknown".
func parseTileKey(key string) (south, west float64) {
	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return 0, 0
	}
	s, err1 := strconv.ParseFloat(parts[len(parts)-2], 64)
	w, err2 := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return s, w
}

// GetCached returns the dataset for tile if (and only if) it is already in
// Redis. It never falls through to Overpass — admin reads must not trigger
// external fetches. ok is false on a cache miss.
func (c *TileCache) GetCached(ctx context.Context, t Tile) (overpass.Dataset, bool, error) {
	payload, err := c.rdb.Get(ctx, tileKey(t)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return overpass.Dataset{}, false, nil
		}
		return overpass.Dataset{}, false, fmt.Errorf("tilecache: redis get: %w", err)
	}
	var ds overpass.Dataset
	if err := json.Unmarshal(payload, &ds); err != nil {
		return overpass.Dataset{}, false, fmt.Errorf("tilecache: decode: %w", err)
	}
	return ds, true, nil
}
