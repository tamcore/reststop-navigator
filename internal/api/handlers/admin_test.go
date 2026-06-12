package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/tamcore/reststop-navigator/internal/api/handlers"
	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
	"github.com/tamcore/reststop-navigator/internal/presence"
)

type fakePresence struct {
	clients []presence.Client
	err     error
}

func (f *fakePresence) List(_ context.Context) ([]presence.Client, error) {
	return f.clients, f.err
}

type fakeTileAdmin struct {
	infos   []cache.TileInfo
	dataset overpass.Dataset
	cached  bool
	stats   cache.CacheStats
	err     error
	gotTile *cache.Tile
}

func (f *fakeTileAdmin) Snapshot(_ context.Context) ([]cache.TileInfo, error) {
	return f.infos, f.err
}

func (f *fakeTileAdmin) GetCached(_ context.Context, t cache.Tile) (overpass.Dataset, bool, error) {
	f.gotTile = &t
	return f.dataset, f.cached, f.err
}

func (f *fakeTileAdmin) Stats() cache.CacheStats { return f.stats }

func mountAdmin(t *testing.T, a *handlers.Admin) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	a.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func getJSON(t *testing.T, url string, v interface{}) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if v != nil {
		mustDecode(t, resp.Body, v)
	}
	return resp
}

func TestAdminPositions(t *testing.T) {
	t.Parallel()
	fp := &fakePresence{clients: []presence.Client{{
		ClientID: "b3a4c1d2-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
		Position: presence.Position{Lat: 48.1, Lon: 11.5, LastSeen: time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)},
	}}}
	srv := mountAdmin(t, handlers.NewAdmin(fp, &fakeTileAdmin{}, nil))

	var out struct {
		Clients []presence.Client `json:"clients"`
		Count   int               `json:"count"`
	}
	resp := getJSON(t, srv.URL+"/api/admin/positions", &out)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if out.Count != 1 || len(out.Clients) != 1 {
		t.Fatalf("count = %d, clients = %d", out.Count, len(out.Clients))
	}
	if out.Clients[0].ClientID != "b3a4c1d2-5e6f-4a7b-8c9d-0e1f2a3b4c5d" {
		t.Errorf("ClientID = %q", out.Clients[0].ClientID)
	}
}

func TestAdminPositions_Error500(t *testing.T) {
	t.Parallel()
	srv := mountAdmin(t, handlers.NewAdmin(&fakePresence{err: errors.New("boom")}, &fakeTileAdmin{}, nil))

	resp := getJSON(t, srv.URL+"/api/admin/positions", nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestAdminTiles(t *testing.T) {
	t.Parallel()
	ft := &fakeTileAdmin{infos: []cache.TileInfo{{Key: "reststops:tile:v2:48.0:11.0", South: 48, West: 11, Stops: 3}}}
	srv := mountAdmin(t, handlers.NewAdmin(&fakePresence{}, ft, nil))

	var out struct {
		Tiles []cache.TileInfo `json:"tiles"`
	}
	resp := getJSON(t, srv.URL+"/api/admin/tiles", &out)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if len(out.Tiles) != 1 || out.Tiles[0].Stops != 3 {
		t.Fatalf("tiles = %+v", out.Tiles)
	}
}

func TestAdminTileStops(t *testing.T) {
	t.Parallel()
	ft := &fakeTileAdmin{
		cached: true,
		dataset: overpass.Dataset{Stops: []overpass.Stop{{
			OSMType: "node", OSMID: 42, Kind: "services", Name: "Aichen Sud",
			Pos: geo.LatLng{Lat: 48.0, Lon: 11.0075},
		}}},
	}
	srv := mountAdmin(t, handlers.NewAdmin(&fakePresence{}, ft, nil))

	var out struct {
		Stops []overpass.Stop `json:"stops"`
	}
	resp := getJSON(t, srv.URL+"/api/admin/tiles/stops?south=48.0&west=11.0", &out)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if len(out.Stops) != 1 || out.Stops[0].Name != "Aichen Sud" {
		t.Fatalf("stops = %+v", out.Stops)
	}
	if ft.gotTile == nil || ft.gotTile.South != 48.0 || ft.gotTile.West != 11.0 {
		t.Errorf("tile = %+v", ft.gotTile)
	}
}

func TestAdminTileStops_404WhenNotCached(t *testing.T) {
	t.Parallel()
	srv := mountAdmin(t, handlers.NewAdmin(&fakePresence{}, &fakeTileAdmin{cached: false}, nil))

	resp := getJSON(t, srv.URL+"/api/admin/tiles/stops?south=48.0&west=11.0", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestAdminTileStops_400OnBadParams(t *testing.T) {
	t.Parallel()
	srv := mountAdmin(t, handlers.NewAdmin(&fakePresence{}, &fakeTileAdmin{}, nil))

	for _, qs := range []string{"", "south=48.0", "south=abc&west=11.0"} {
		resp := getJSON(t, srv.URL+"/api/admin/tiles/stops?"+qs, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("query %q: status = %d, want 400", qs, resp.StatusCode)
		}
	}
}

func TestAdminStats(t *testing.T) {
	t.Parallel()
	ft := &fakeTileAdmin{stats: cache.CacheStats{Hits: 7, Misses: 2}}
	fp := &fakePresence{clients: []presence.Client{{ClientID: "a"}, {ClientID: "b"}}}
	srv := mountAdmin(t, handlers.NewAdmin(fp, ft, nil))

	var out struct {
		UptimeSeconds int64            `json:"uptime_seconds"`
		Cache         cache.CacheStats `json:"cache"`
		PresenceCount int              `json:"presence_count"`
	}
	resp := getJSON(t, srv.URL+"/api/admin/stats", &out)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if out.Cache.Hits != 7 || out.Cache.Misses != 2 {
		t.Errorf("cache = %+v", out.Cache)
	}
	if out.PresenceCount != 2 {
		t.Errorf("presence_count = %d, want 2", out.PresenceCount)
	}
	if out.UptimeSeconds < 0 {
		t.Errorf("uptime_seconds = %d", out.UptimeSeconds)
	}
}
