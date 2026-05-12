package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// fakeFetcher is an inline OverpassFetcher implementation for tests. It
// records the queries it has been asked to run and returns canned responses
// per call index, or a fixed response if all responses share one slot.
type fakeFetcher struct {
	calls    int32
	queries  []string
	response []byte
	err      error
}

func (f *fakeFetcher) Query(_ context.Context, q string) ([]byte, error) {
	atomic.AddInt32(&f.calls, 1)
	f.queries = append(f.queries, q)
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

// slowFetcher blocks until its gate is released, then returns canned data.
type slowFetcher struct {
	gate     chan struct{}
	calls    int32
	response []byte
}

func (f *slowFetcher) Query(ctx context.Context, _ string) ([]byte, error) {
	atomic.AddInt32(&f.calls, 1)
	select {
	case <-f.gate:
		return f.response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func newRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func sampleOverpassResponse() []byte {
	return []byte(`{
		"version": 0.6,
		"generator": "test",
		"elements": [
			{"type":"way","id":1001,
			 "tags":{"highway":"motorway","oneway":"yes","ref":"A8"},
			 "geometry":[
				{"lat":48.0,"lon":11.0},
				{"lat":48.0,"lon":11.005},
				{"lat":48.0,"lon":11.01}
			 ]
			},
			{"type":"node","id":2001,"lat":48.0,"lon":11.0075,
			 "tags":{"highway":"services","name":"Aichen Sud"}},
			{"type":"node","id":3001,"lat":48.0,"lon":11.0080,
			 "tags":{"amenity":"fuel"}}
		]
	}`)
}

func TestTileFor(t *testing.T) {
	tile := TileFor(geo.LatLng{Lat: 48.27, Lon: 11.83})
	if tile.South != 48.0 || tile.West != 11.5 {
		t.Fatalf("TileFor(48.27,11.83) = %+v, want {48.0, 11.5}", tile)
	}

	// Negative coordinate floors away from zero.
	negTile := TileFor(geo.LatLng{Lat: -0.1, Lon: -0.1})
	if negTile.South != -0.5 || negTile.West != -0.5 {
		t.Fatalf("TileFor(-0.1,-0.1) = %+v, want {-0.5, -0.5}", negTile)
	}
}

func TestTilesAroundReturnsSingleTile(t *testing.T) {
	pos := geo.LatLng{Lat: 48.0, Lon: 11.003}
	tiles := TilesAround(pos)
	if len(tiles) != 1 {
		t.Fatalf("TilesAround returned %d tiles, want 1", len(tiles))
	}
	if tiles[0] != TileFor(pos) {
		t.Fatalf("TilesAround[0] = %+v, want %+v", tiles[0], TileFor(pos))
	}
}

func TestTileBBox(t *testing.T) {
	tile := Tile{South: 48.0, West: 11.5}
	bb := tile.BBox()
	if bb.South != 48.0 || bb.West != 11.5 || bb.North != 48.5 || bb.East != 12.0 {
		t.Fatalf("Tile.BBox() = %+v, want {S:48.0 W:11.5 N:48.5 E:12.0}", bb)
	}
}

func TestTileKeyDeterministic(t *testing.T) {
	tile := Tile{South: 48.0, West: 11.5}
	got := tileKey(tile)
	want := "reststops:tile:48.0:11.5"
	if got != want {
		t.Fatalf("tileKey = %q, want %q", got, want)
	}
}

func TestNewTileCacheDefaults(t *testing.T) {
	rdb := newRedis(t)
	c := NewTileCache(rdb, &fakeFetcher{})
	if c.ttl != 7*24*time.Hour {
		t.Fatalf("default ttl = %v, want %v", c.ttl, 7*24*time.Hour)
	}
	if c.missTTL != 24*time.Hour {
		t.Fatalf("default missTTL = %v, want %v", c.missTTL, 24*time.Hour)
	}
}

func TestWithTileTTLAndMissTTL(t *testing.T) {
	rdb := newRedis(t)
	c := NewTileCache(rdb, &fakeFetcher{},
		WithTileTTL(2*time.Hour),
		WithTileMissTTL(15*time.Minute),
	)
	if c.ttl != 2*time.Hour {
		t.Fatalf("WithTileTTL not applied: got %v", c.ttl)
	}
	if c.missTTL != 15*time.Minute {
		t.Fatalf("WithTileMissTTL not applied: got %v", c.missTTL)
	}
}

func TestGetCacheMissThenHit(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)
	tile := Tile{South: 48.0, West: 11.0}
	ctx := context.Background()

	// Cold: triggers fetch + write.
	ds, err := c.Get(ctx, tile)
	if err != nil {
		t.Fatalf("Get cold: %v", err)
	}
	if len(ds.Ways) != 1 {
		t.Fatalf("cold Ways = %d, want 1", len(ds.Ways))
	}
	if len(ds.Stops) != 1 {
		t.Fatalf("cold Stops = %d, want 1", len(ds.Stops))
	}
	if len(ds.Amenities) != 1 {
		t.Fatalf("cold Amenities = %d, want 1", len(ds.Amenities))
	}
	if got := atomic.LoadInt32(&fetcher.calls); got != 1 {
		t.Fatalf("cold fetcher calls = %d, want 1", got)
	}

	// Warm: served from Redis, no extra Overpass call.
	ds2, err := c.Get(ctx, tile)
	if err != nil {
		t.Fatalf("Get warm: %v", err)
	}
	if got := atomic.LoadInt32(&fetcher.calls); got != 1 {
		t.Fatalf("warm fetcher calls = %d, want 1 (no refetch)", got)
	}
	if len(ds2.Ways) != 1 || len(ds2.Stops) != 1 {
		t.Fatalf("warm dataset diverged from cold: %+v", ds2)
	}

	// Verify the query body matches the tile bbox.
	if len(fetcher.queries) == 0 || !strings.Contains(fetcher.queries[0], "48,11,48.5,11.5") {
		t.Fatalf("expected query to include tile bbox 48,11,48.5,11.5, got %q", fetcher.queries[0])
	}
}

func TestGetEmptyResponseUsesMissTTL(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{response: []byte(`{"elements":[]}`)}
	c := NewTileCache(rdb, fetcher,
		WithTileTTL(7*24*time.Hour),
		WithTileMissTTL(11*time.Minute),
	)
	tile := Tile{South: 47.0, West: 9.0}
	ctx := context.Background()

	ds, err := c.Get(ctx, tile)
	if err != nil {
		t.Fatalf("Get empty: %v", err)
	}
	if len(ds.Ways) != 0 || len(ds.Stops) != 0 {
		t.Fatalf("expected empty dataset, got %+v", ds)
	}

	ttl := rdb.TTL(ctx, tileKey(tile)).Val()
	// miniredis honours TTL; allow small drift.
	if ttl <= 0 || ttl > 12*time.Minute {
		t.Fatalf("expected miss TTL ~11m, got %v", ttl)
	}
}

func TestGetReturnsErrorOnOverpassFailure(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{err: errors.New("boom")}
	c := NewTileCache(rdb, fetcher)
	_, err := c.Get(context.Background(), Tile{South: 0, West: 0})
	if err == nil {
		t.Fatal("expected error on overpass failure, got nil")
	}
	if !strings.Contains(err.Error(), "overpass") {
		t.Fatalf("expected error to mention overpass, got %v", err)
	}
}

func TestGetReturnsErrorOnDecodeFailure(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{response: []byte(`not-json`)}
	c := NewTileCache(rdb, fetcher)
	_, err := c.Get(context.Background(), Tile{South: 0, West: 0})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestGetRefetchesOnCorruptCache(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)
	tile := Tile{South: 48.0, West: 11.0}
	ctx := context.Background()

	// Pre-seed an unparseable cache entry.
	if err := rdb.Set(ctx, tileKey(tile), "garbage", time.Hour).Err(); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ds, err := c.Get(ctx, tile)
	if err != nil {
		t.Fatalf("Get after corrupt cache: %v", err)
	}
	if len(ds.Ways) != 1 {
		t.Fatalf("expected refetched dataset, got %+v", ds)
	}
	if got := atomic.LoadInt32(&fetcher.calls); got != 1 {
		t.Fatalf("expected one refetch call, got %d", got)
	}
}

func TestGetMergedSingleTile(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)

	pos := geo.LatLng{Lat: 48.0, Lon: 11.003}
	ds, err := c.GetMerged(context.Background(), pos)
	if err != nil {
		t.Fatalf("GetMerged: %v", err)
	}
	if len(ds.Ways) != 1 || len(ds.Stops) != 1 || len(ds.Amenities) != 1 {
		t.Fatalf("GetMerged dataset = %+v, want 1/1/1", ds)
	}
	// TilesAround currently returns one tile, so GetMerged should issue
	// exactly one Overpass call on a cold cache.
	if got := atomic.LoadInt32(&fetcher.calls); got != 1 {
		t.Fatalf("GetMerged calls = %d, want 1", got)
	}
}

func TestGetMergedPropagatesError(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{err: errors.New("net down")}
	c := NewTileCache(rdb, fetcher)
	pos := geo.LatLng{Lat: 48.0, Lon: 11.003}
	_, err := c.GetMerged(context.Background(), pos)
	if err == nil {
		t.Fatal("expected error from GetMerged, got nil")
	}
}

func TestGetCachedPayloadRoundtrips(t *testing.T) {
	rdb := newRedis(t)
	fetcher := &fakeFetcher{response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)
	tile := Tile{South: 48.0, West: 11.0}
	ctx := context.Background()

	if _, err := c.Get(ctx, tile); err != nil {
		t.Fatalf("prime: %v", err)
	}

	// Confirm payload in Redis is JSON-decodable as overpass.Dataset.
	raw, err := rdb.Get(ctx, tileKey(tile)).Bytes()
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	var ds overpass.Dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		t.Fatalf("payload not a Dataset: %v", err)
	}
	if len(ds.Ways) != 1 {
		t.Fatalf("cached Ways = %d, want 1", len(ds.Ways))
	}
}

func TestGetCompletesAfterCallerContextCancel(t *testing.T) {
	rdb := newRedis(t)
	gate := make(chan struct{})
	fetcher := &slowFetcher{gate: gate, response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)
	tile := Tile{South: 48.0, West: 11.0}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := c.Get(ctx, tile)
		done <- err
	}()

	// Wait for the fetcher to be called, then cancel the caller's context.
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Release the fetcher — tile should still be cached.
	close(gate)

	// The caller gets the result despite the cancelled context.
	if err := <-done; err != nil {
		t.Fatalf("Get returned error after ctx cancel: %v", err)
	}

	// Verify the tile landed in Redis.
	payload, err := rdb.Get(context.Background(), tileKey(tile)).Bytes()
	if err != nil {
		t.Fatalf("expected tile in cache after ctx cancel, redis error: %v", err)
	}
	var ds overpass.Dataset
	if err := json.Unmarshal(payload, &ds); err != nil {
		t.Fatalf("cached payload not valid: %v", err)
	}
	if len(ds.Ways) != 1 {
		t.Fatalf("cached Ways = %d, want 1", len(ds.Ways))
	}
}

func TestGetSingleflightCoalescesConcurrentFetches(t *testing.T) {
	rdb := newRedis(t)
	gate := make(chan struct{})
	fetcher := &slowFetcher{gate: gate, response: sampleOverpassResponse()}
	c := NewTileCache(rdb, fetcher)
	tile := Tile{South: 48.0, West: 11.0}

	const n = 5
	errs := make([]error, n)
	datasets := make([]overpass.Dataset, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			datasets[i], errs[i] = c.Get(context.Background(), tile)
		}()
	}

	// Give goroutines time to enter singleflight, then release.
	time.Sleep(50 * time.Millisecond)
	close(gate)
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if len(datasets[i].Ways) != 1 {
			t.Fatalf("goroutine %d: Ways = %d, want 1", i, len(datasets[i].Ways))
		}
	}

	// Only one Overpass call should have been made.
	if got := atomic.LoadInt32(&fetcher.calls); got != 1 {
		t.Fatalf("singleflight calls = %d, want 1", got)
	}
}
