// Package api wires HTTP routes for the Reststop Navigator backend.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/tamcore/reststop-navigator/internal/api/handlers"
)

// NewRouter returns the public HTTP handler. Routes live under /api.
//
// stopsSvc may be nil — in that case only /api/healthz is mounted, useful for
// the bootstrap/init phase before the cache is hydrated.
func NewRouter(stopsSvc handlers.StopsService) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/healthz", healthz)

	if stopsSvc != nil {
		handlers.NewStops(stopsSvc).Mount(r)
	}
	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
