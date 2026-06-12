package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/tamcore/reststop-navigator/internal/api"
	"github.com/tamcore/reststop-navigator/internal/api/handlers"
	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/overpass"
	"github.com/tamcore/reststop-navigator/internal/presence"
	"github.com/tamcore/reststop-navigator/web"
)

func TestHealthzReturnsOK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(api.NewRouter(nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("GET /api/healthz: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got, want := resp.Header.Get("Content-Type"), "application/json"; got != want {
		t.Errorf("content-type = %q, want %q", got, want)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body %q: %v", body, err)
	}
	if payload.Status != "ok" {
		t.Errorf("status field = %q, want %q", payload.Status, "ok")
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(api.NewRouter(nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/does-not-exist")
	if err != nil {
		t.Fatalf("GET unknown: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

type stubPresence struct{}

func (stubPresence) List(_ context.Context) ([]presence.Client, error) {
	return []presence.Client{}, nil
}

type stubTiles struct{}

func (stubTiles) Snapshot(_ context.Context) ([]cache.TileInfo, error) {
	return []cache.TileInfo{}, nil
}

func (stubTiles) GetCached(_ context.Context, _ cache.Tile) (overpass.Dataset, bool, error) {
	return overpass.Dataset{}, false, nil
}

func (stubTiles) Stats() cache.CacheStats { return cache.CacheStats{} }

func TestAdminRoutesNotMountedWithoutPassword(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(api.NewRouter(nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/admin/stats")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (admin disabled)", resp.StatusCode)
	}
}

func TestAdminRoutesRequireAuth(t *testing.T) {
	t.Parallel()

	admin := handlers.NewAdmin(stubPresence{}, stubTiles{}, nil)
	srv := httptest.NewServer(api.NewRouter(nil, api.WithAdmin("s3cret", admin)))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/admin/stats")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/admin/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("admin", "s3cret")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp2.Body.Close() })
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("authed status = %d, want 200", resp2.StatusCode)
	}
}

// TestSPAFallbackServesIndexForUnknownPath ensures that when the embedded
// frontend is present, any non-/api path which does not resolve to a static
// file falls back to index.html. Without this, refreshing/navigating to a
// client-side route like /stop/<id> returns a hard 404.
func TestSPAFallbackServesIndexForUnknownPath(t *testing.T) {
	const indexBody = "<!doctype html><body>spa-shell</body>"
	web.FS = fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(indexBody)},
	}
	t.Cleanup(func() { web.FS = nil })

	srv := httptest.NewServer(api.NewRouter(nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/stop/node%2F1487561134?lat=50&lon=8")
	if err != nil {
		t.Fatalf("GET /stop/...: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (SPA fallback)", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != indexBody {
		t.Errorf("body = %q, want %q", body, indexBody)
	}
}

// TestSPAFallbackPreservesRealAssets guards against a regression where the
// SPA fallback rewrote *every* path to "/", which would break JS/CSS bundles.
// A real asset under the FS must be served verbatim, not the index.html.
func TestSPAFallbackPreservesRealAssets(t *testing.T) {
	const assetBody = "console.log('hydrate');"
	web.FS = fstest.MapFS{
		"index.html":          &fstest.MapFile{Data: []byte("<html>shell</html>")},
		"_app/immutable/x.js": &fstest.MapFile{Data: []byte(assetBody)},
	}
	t.Cleanup(func() { web.FS = nil })

	srv := httptest.NewServer(api.NewRouter(nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/_app/immutable/x.js")
	if err != nil {
		t.Fatalf("GET asset: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != assetBody {
		t.Errorf("body = %q, want raw asset %q (rewrite leaked into asset paths)", body, assetBody)
	}
}
