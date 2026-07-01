package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/overpass"
	"github.com/tamcore/reststop-navigator/internal/presence"
)

// PresenceLister lists live clients. Implemented by *presence.Tracker.
type PresenceLister interface {
	List(ctx context.Context) ([]presence.Client, error)
}

// TileAdmin is the read-only tile cache surface the admin API needs.
// Implemented by *cache.TileCache.
type TileAdmin interface {
	Snapshot(ctx context.Context) ([]cache.TileInfo, error)
	GetCached(ctx context.Context, t cache.Tile) (overpass.Dataset, bool, error)
	Stats() cache.CacheStats
}

// Admin bundles the admin API handlers: live client positions, cached tiles,
// per-tile stops, and runtime stats.
type Admin struct {
	presence PresenceLister
	tiles    TileAdmin
	rdb      *redis.Client // optional; nil omits Redis stats
	start    time.Time
}

// NewAdmin builds the admin handlers. rdb may be nil — Redis stats are then
// omitted from the stats response.
func NewAdmin(p PresenceLister, t TileAdmin, rdb *redis.Client) *Admin {
	return &Admin{presence: p, tiles: t, rdb: rdb, start: time.Now()}
}

// Mount registers the admin routes onto r. Callers must wrap r (or the group
// this is mounted into) with middleware.AdminAuth.
func (a *Admin) Mount(r chi.Router) {
	r.Get("/api/admin/positions", a.positions)
	r.Get("/api/admin/tiles", a.tilesList)
	r.Get("/api/admin/tiles/stops", a.tileStops)
	r.Get("/api/admin/stats", a.stats)
}

func (a *Admin) positions(w http.ResponseWriter, r *http.Request) {
	clients, err := a.presence.List(r.Context())
	if err != nil {
		slog.Error("admin: presence list failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clients": clients,
		"count":   len(clients),
	})
}

func (a *Admin) tilesList(w http.ResponseWriter, r *http.Request) {
	infos, err := a.tiles.Snapshot(r.Context())
	if err != nil {
		slog.Error("admin: tile snapshot failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tiles": infos})
}

func (a *Admin) tileStops(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	south, err := requiredFloat(q, "south")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	west, err := requiredFloat(q, "west")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ds, ok, err := a.tiles.GetCached(r.Context(), cache.Tile{South: south, West: west})
	if err != nil {
		slog.Error("admin: tile read failed", "error", err, "south", south, "west", west)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "tile not cached")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stops": ds.Stops})
}

// usedMemoryRe extracts used_memory from a Redis INFO memory section.
var usedMemoryRe = regexp.MustCompile(`used_memory:(\d+)`)

func (a *Admin) stats(w http.ResponseWriter, r *http.Request) {
	clients, err := a.presence.List(r.Context())
	if err != nil {
		slog.Error("admin: presence list failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := map[string]any{
		"uptime_seconds": int64(time.Since(a.start).Seconds()),
		"cache":          a.tiles.Stats(),
		"presence_count": len(clients),
	}
	if a.rdb != nil {
		out["redis"] = a.redisStats(r.Context())
	}
	writeJSON(w, http.StatusOK, out)
}

// redisStats is best-effort: failures log and yield zero values rather than
// failing the whole stats response.
func (a *Admin) redisStats(ctx context.Context) map[string]int64 {
	stats := map[string]int64{"keys": 0, "used_memory_bytes": 0}
	if keys, err := a.rdb.DBSize(ctx).Result(); err == nil {
		stats["keys"] = keys
	} else {
		slog.Warn("admin: redis dbsize failed", "error", err)
	}
	if info, err := a.rdb.Info(ctx, "memory").Result(); err == nil {
		if m := usedMemoryRe.FindStringSubmatch(info); m != nil {
			if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				stats["used_memory_bytes"] = v
			}
		}
	} else {
		slog.Warn("admin: redis info failed", "error", err)
	}
	return stats
}
