package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

func newPrimedCache(t *testing.T) *TileCache {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewTileCache(rdb, &fakeFetcher{response: sampleOverpassResponse()})
}

func TestSnapshot_ReturnsCachedTiles(t *testing.T) {
	t.Parallel()
	c := newPrimedCache(t)
	ctx := context.Background()

	if _, err := c.Get(ctx, Tile{South: 48.0, West: 11.0}); err != nil {
		t.Fatalf("Get: %v", err)
	}

	infos, err := c.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("Snapshot returned %d tiles, want 1", len(infos))
	}
	info := infos[0]
	if info.South != 48.0 || info.West != 11.0 {
		t.Errorf("tile coords = %v,%v", info.South, info.West)
	}
	if info.Stops != 1 || info.Ways != 1 || info.Amenities != 1 {
		t.Errorf("counts = stops:%d ways:%d amenities:%d, want 1 each", info.Stops, info.Ways, info.Amenities)
	}
	if info.Bytes <= 0 {
		t.Errorf("Bytes = %d, want > 0", info.Bytes)
	}
	if info.TTLSeconds <= 0 || info.TTLSeconds > 7*24*3600 {
		t.Errorf("TTLSeconds = %d", info.TTLSeconds)
	}
	if info.Key == "" {
		t.Error("Key should be set")
	}
}

func TestSnapshot_EmptyCache(t *testing.T) {
	t.Parallel()
	c := newPrimedCache(t)

	infos, err := c.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(infos) != 0 {
		t.Fatalf("Snapshot returned %d tiles, want 0", len(infos))
	}
}

func TestGetCached_ReturnsTileWithoutFetching(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	fetcher := &fakeFetcher{response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)
	ctx := context.Background()

	tile := Tile{South: 48.0, West: 11.0}
	if _, err := c.Get(ctx, tile); err != nil {
		t.Fatalf("Get: %v", err)
	}
	callsAfterPrime := fetcher.calls

	ds, ok, err := c.GetCached(ctx, tile)
	if err != nil {
		t.Fatalf("GetCached: %v", err)
	}
	if !ok {
		t.Fatal("GetCached: ok = false for cached tile")
	}
	if len(ds.Stops) != 1 {
		t.Errorf("Stops = %d, want 1", len(ds.Stops))
	}

	_, ok, err = c.GetCached(ctx, Tile{South: 50.0, West: 9.0})
	if err != nil {
		t.Fatalf("GetCached(miss): %v", err)
	}
	if ok {
		t.Error("GetCached: ok = true for uncached tile")
	}
	if fetcher.calls != callsAfterPrime {
		t.Errorf("GetCached must never hit Overpass (calls %d -> %d)", callsAfterPrime, fetcher.calls)
	}
}

func TestStats_CountsHitsAndMisses(t *testing.T) {
	t.Parallel()
	c := newPrimedCache(t)
	ctx := context.Background()
	tile := Tile{South: 48.0, West: 11.0}

	for i := 0; i < 3; i++ { // 1 miss, then 2 hits
		if _, err := c.Get(ctx, tile); err != nil {
			t.Fatalf("Get #%d: %v", i, err)
		}
	}

	st := c.Stats()
	if st.Hits != 2 {
		t.Errorf("Hits = %d, want 2", st.Hits)
	}
	if st.Misses != 1 {
		t.Errorf("Misses = %d, want 1", st.Misses)
	}
}

func TestReportStats_AfterSnapshotRefactor(t *testing.T) {
	t.Parallel()
	c := newPrimedCache(t)
	ctx := context.Background()
	if _, err := c.GetMerged(ctx, geo.LatLng{Lat: 48.1, Lon: 11.2}); err != nil {
		t.Fatalf("GetMerged: %v", err)
	}
	// Smoke: ReportStats must keep working on top of Snapshot.
	c.ReportStats(ctx)
}
