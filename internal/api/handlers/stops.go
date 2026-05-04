// Package handlers exposes the public REST API for Reststop Navigator.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/stops"
)

// StopsService is the contract handlers depend on. Implemented by *stops.Service
// in production; faked in tests.
type StopsService interface {
	Upcoming(ctx context.Context, req stops.UpcomingRequest) (stops.UpcomingResponse, error)
	Get(ctx context.Context, id string, pos geo.LatLng) (stops.DetailResponse, error)
}

// Stops bundles the upcoming + detail handlers.
type Stops struct {
	svc StopsService
}

// NewStops builds a Stops handler.
func NewStops(svc StopsService) *Stops {
	return &Stops{svc: svc}
}

// Mount registers the routes onto r under /api/stops.
func (h *Stops) Mount(r chi.Router) {
	r.Get("/api/stops/upcoming", h.upcoming)
	r.Get("/api/stops/detail", h.detail)
}

func (h *Stops) upcoming(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lat, err := requiredFloat(q, "lat")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	lon, err := requiredFloat(q, "lon")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	heading, err := optionalFloat(q, "heading", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	speed, err := optionalFloat(q, "speed", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := optionalInt(q, "limit", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req := stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: lat, Lon: lon},
		Heading: heading,
		Speed:   speed,
		Filters: parseFilters(q.Get("filters")),
		Limit:   limit,
	}

	resp, err := h.svc.Upcoming(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Stops) detail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	lat, err := requiredFloat(q, "lat")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	lon, err := requiredFloat(q, "lon")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.svc.Get(r.Context(), id, geo.LatLng{Lat: lat, Lon: lon})
	if err != nil {
		if errors.Is(err, stops.ErrStopNotFound) {
			writeError(w, http.StatusNotFound, "stop not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseFilters(raw string) stops.Filters {
	var f stops.Filters
	for _, part := range strings.Split(raw, ",") {
		switch strings.TrimSpace(part) {
		case "fuel":
			f.Fuel = true
		case "charging":
			f.Charging = true
		case "food":
			f.Food = true
		case "toilets":
			f.Toilets = true
		case "open24h":
			f.Open24h = true
		case "dog":
			f.Dog = true
		}
	}
	return f
}

func requiredFloat(q map[string][]string, key string) (float64, error) {
	v := firstValue(q, key)
	if v == "" {
		return 0, errors.New(key + " is required")
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, errors.New(key + " must be a number")
	}
	return f, nil
}

func optionalFloat(q map[string][]string, key string, def float64) (float64, error) {
	v := firstValue(q, key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, errors.New(key + " must be a number")
	}
	return f, nil
}

func optionalInt(q map[string][]string, key string, def int) (int, error) {
	v := firstValue(q, key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}
	return n, nil
}

func firstValue(q map[string][]string, key string) string {
	vs := q[key]
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
