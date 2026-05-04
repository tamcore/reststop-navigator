// Package api wires HTTP routes for the Reststop Navigator backend.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter returns the public HTTP handler. Routes live under /api.
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/api/healthz", healthz)
	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
