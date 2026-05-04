package cache_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

const overpassFixture = `{
  "elements": [
    {
      "type":"way","id":1,
      "tags":{"highway":"motorway","oneway":"yes","ref":"A8"},
      "geometry":[{"lat":48,"lon":11},{"lat":48,"lon":11.01}]
    },
    {
      "type":"node","id":42,
      "lat":48,"lon":11.005,
      "tags":{"highway":"services","name":"Aichen"}
    }
  ]
}`

func setupHydratorEnv(t *testing.T, handler http.HandlerFunc) (*cache.Redis, *cache.Hydrator) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := cache.NewRedis(rdb, cache.WithTTL(time.Hour))

	client := overpass.NewClient([]string{srv.URL},
		overpass.WithBackoff(time.Millisecond, 2*time.Millisecond),
		overpass.WithMaxRetries(1),
	)
	h := cache.NewHydrator(client, c)
	return c, h
}

func TestHydrateCountry_WritesDataset(t *testing.T) {
	t.Parallel()

	c, h := setupHydratorEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(overpassFixture))
	})
	ctx := context.Background()

	if err := h.HydrateCountry(ctx, overpass.DE); err != nil {
		t.Fatalf("HydrateCountry: %v", err)
	}

	got, err := c.ReadDataset(ctx, overpass.DE)
	if err != nil {
		t.Fatalf("ReadDataset: %v", err)
	}
	if got.Country != overpass.DE {
		t.Errorf("country = %q", got.Country)
	}
	if got.Version == "" {
		t.Error("version should be set after hydrate")
	}
	// HydrateCountry now fans out to 4 quadrant sub-bboxes; the stub returns
	// the same fixture for every query, so the merged result holds 4x the
	// elements. Assert non-empty + correct shape rather than exact counts.
	if len(got.Ways) == 0 || got.Ways[0].Ref != "A8" {
		t.Errorf("ways: %+v", got.Ways)
	}
	if len(got.Stops) == 0 || got.Stops[0].Name != "Aichen" {
		t.Errorf("stops: %+v", got.Stops)
	}
}

func TestHydrateCountry_OverpassFailureDoesNotWipeCache(t *testing.T) {
	t.Parallel()

	var status atomic.Int32
	status.Store(http.StatusOK)
	c, h := setupHydratorEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		s := int(status.Load())
		if s != http.StatusOK {
			w.WriteHeader(s)
			return
		}
		_, _ = w.Write([]byte(overpassFixture))
	})
	ctx := context.Background()

	if err := h.HydrateCountry(ctx, overpass.DE); err != nil {
		t.Fatalf("first hydrate: %v", err)
	}
	first, err := c.ReadDataset(ctx, overpass.DE)
	if err != nil {
		t.Fatal(err)
	}

	status.Store(http.StatusInternalServerError)
	if err := h.HydrateCountry(ctx, overpass.DE); err == nil {
		t.Fatal("expected error from failing overpass")
	}

	stillThere, err := c.ReadDataset(ctx, overpass.DE)
	if err != nil {
		t.Fatalf("cache lost prior version: %v", err)
	}
	if stillThere.Version != first.Version {
		t.Errorf("version changed after failed refresh: %q -> %q", first.Version, stillThere.Version)
	}
}

func TestHydrateAll_PerCountryFailureDoesNotAbortOthers(t *testing.T) {
	t.Parallel()

	c, h := setupHydratorEnv(t, func(w http.ResponseWriter, r *http.Request) {
		body := r.PostFormValue("data")
		// AT is uniquely identifiable by its southern latitude (46.37);
		// DE/SK/CZ all start north of 47.27.
		if strings.Contains(body, "46.37") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(overpassFixture))
	})
	ctx := context.Background()

	err := h.HydrateAll(ctx)
	if err == nil {
		t.Fatal("expected aggregate error containing the AT failure")
	}
	if !strings.Contains(err.Error(), "AT") {
		t.Errorf("expected error mentioning AT, got %v", err)
	}

	for _, country := range []overpass.CountryISO{overpass.DE, overpass.SK, overpass.CZ} {
		if _, rerr := c.ReadDataset(ctx, country); rerr != nil {
			t.Errorf("country %q missing after partial failure: %v", country, rerr)
		}
	}
}

func TestHydrateCountry_RejectsUnsupported(t *testing.T) {
	t.Parallel()
	_, h := setupHydratorEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Overpass should not be hit for unsupported country")
		w.WriteHeader(http.StatusOK)
	})
	if err := h.HydrateCountry(context.Background(), overpass.CountryISO("XX")); err == nil {
		t.Fatal("expected error for unsupported country")
	}
}
