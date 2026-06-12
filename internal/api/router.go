// Package api wires HTTP routes for the Reststop Navigator backend.
package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/tamcore/reststop-navigator/internal/api/handlers"
	"github.com/tamcore/reststop-navigator/internal/api/middleware"
	"github.com/tamcore/reststop-navigator/web"
)

// NewRouter returns the public HTTP handler. Routes live under /api.
//
// stopsSvc may be nil — in that case only /api/healthz is mounted, useful for
// the bootstrap/init phase before the cache is hydrated.
func NewRouter(stopsSvc handlers.StopsService) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.AccessLog)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)

	r.Get("/api/healthz", healthz)

	if stopsSvc != nil {
		handlers.NewStops(stopsSvc).Mount(r)
	}

	// Mount the embedded frontend at root, when present. Built without the
	// prodfrontend tag, web.FS is nil and we just serve a friendly note.
	if web.Available() {
		fs := http.FileServer(http.FS(web.FS))
		// Real static assets (JS bundles, CSS, fonts, favicons) are served
		// as-is. Unknown non-/api paths fall back to index.html so that
		// client-side routes like /stop/<id> work on direct load / refresh.
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			if hasEmbeddedFile(req.URL.Path) {
				fs.ServeHTTP(w, req)
				return
			}
			req2 := req.Clone(req.Context())
			req2.URL.Path = "/"
			req2.URL.RawPath = ""
			fs.ServeHTTP(w, req2)
		})
	} else {
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><meta charset=utf-8><title>reststop-navigator</title>` +
				`<p>Backend running. Frontend not embedded (build without -tags prodfrontend).</p>`))
		})
	}
	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// hasEmbeddedFile reports whether urlPath maps to an existing file inside the
// embedded frontend FS. Used by the SPA fallback to distinguish "real asset
// 404'd by chi for some other reason" from "unknown client-side route".
func hasEmbeddedFile(urlPath string) bool {
	if web.FS == nil {
		return false
	}
	p := strings.TrimPrefix(urlPath, "/")
	if p == "" {
		return false
	}
	f, err := web.FS.Open(p)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
