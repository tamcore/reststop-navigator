package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

func newCache(t *testing.T) (*cache.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := cache.NewRedis(rdb, cache.WithTTL(time.Hour))
	return c, mr
}

func sampleDataset() overpass.Dataset {
	return overpass.Dataset{
		Country: overpass.DE,
		Version: "1714824000",
		Ways: []geo.Way{
			{
				ID:     "way/1",
				Coords: []geo.LatLng{{Lat: 48, Lon: 11}, {Lat: 48, Lon: 11.01}},
				Oneway: true,
				Ref:    "A8",
			},
		},
		Stops: []overpass.Stop{
			{OSMID: 100, Kind: "services", Pos: geo.LatLng{Lat: 48, Lon: 11.005}, Name: "Aichen"},
		},
	}
}

func TestRedis_WriteAndReadRoundTrip(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t)
	ctx := context.Background()

	ds := sampleDataset()
	if err := c.WriteDataset(ctx, ds); err != nil {
		t.Fatalf("WriteDataset: %v", err)
	}

	got, err := c.ReadDataset(ctx, overpass.DE)
	if err != nil {
		t.Fatalf("ReadDataset: %v", err)
	}
	if got.Country != ds.Country {
		t.Errorf("country = %q, want %q", got.Country, ds.Country)
	}
	if got.Version != ds.Version {
		t.Errorf("version = %q, want %q", got.Version, ds.Version)
	}
	if len(got.Ways) != 1 || got.Ways[0].Ref != "A8" {
		t.Errorf("ways: %+v", got.Ways)
	}
	if len(got.Stops) != 1 || got.Stops[0].Name != "Aichen" {
		t.Errorf("stops: %+v", got.Stops)
	}
}

func TestRedis_TwoKeySwap(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t)
	ctx := context.Background()

	v1 := sampleDataset()
	v1.Version = "1"
	if err := c.WriteDataset(ctx, v1); err != nil {
		t.Fatal(err)
	}

	v2 := sampleDataset()
	v2.Version = "2"
	v2.Stops[0].Name = "Holledau"
	if err := c.WriteDataset(ctx, v2); err != nil {
		t.Fatal(err)
	}

	got, err := c.ReadDataset(ctx, overpass.DE)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "2" {
		t.Errorf("version after swap = %q, want %q", got.Version, "2")
	}
	if got.Stops[0].Name != "Holledau" {
		t.Errorf("stop name after swap = %q, want %q", got.Stops[0].Name, "Holledau")
	}
}

func TestRedis_ReadMissing(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t)
	if _, err := c.ReadDataset(context.Background(), overpass.AT); err == nil {
		t.Fatal("expected error for missing dataset")
	}
}

func TestRedis_ReadCorruptVersionedKey(t *testing.T) {
	t.Parallel()
	c, mr := newCache(t)
	ctx := context.Background()

	if err := c.WriteDataset(ctx, sampleDataset()); err != nil {
		t.Fatal(err)
	}

	versionPtr, err := mr.Get("reststops:country:DE:current")
	if err != nil {
		t.Fatal(err)
	}
	if err := mr.Set("reststops:country:DE:v"+versionPtr, "not-json"); err != nil {
		t.Fatal(err)
	}

	if _, err := c.ReadDataset(ctx, overpass.DE); err == nil {
		t.Fatal("expected error decoding corrupt payload")
	}
}

func TestRedis_RejectsUnsupportedCountry(t *testing.T) {
	t.Parallel()
	c, _ := newCache(t)
	bad := sampleDataset()
	bad.Country = overpass.CountryISO("XX")
	if err := c.WriteDataset(context.Background(), bad); err == nil {
		t.Fatal("expected WriteDataset to reject unsupported country")
	}
	if _, err := c.ReadDataset(context.Background(), overpass.CountryISO("XX")); err == nil {
		t.Fatal("expected ReadDataset to reject unsupported country")
	}
}
