package overpass_test

import (
	"testing"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

func TestEnrichDataset_NearbyAmenitiesSetFlags(t *testing.T) {
	t.Parallel()
	ds := overpass.Dataset{
		Stops: []overpass.Stop{
			{OSMID: 1, Kind: "rest_area", Pos: geo.LatLng{Lat: 48, Lon: 11}, Name: "Holledau"},
		},
		Amenities: []overpass.AmenityNode{
			{Kind: "fuel", Pos: geo.LatLng{Lat: 48, Lon: 11.001}},
			{Kind: "charging_station", Pos: geo.LatLng{Lat: 48.001, Lon: 11}},
			{Kind: "restaurant", Pos: geo.LatLng{Lat: 48, Lon: 11.0005}},
			{Kind: "fuel", Pos: geo.LatLng{Lat: 48, Lon: 11.5}, Tags: map[string]string{"name": "far"}},
		},
	}
	overpass.EnrichDataset(&ds)

	got := ds.Stops[0].Amenities
	if !got.Fuel || !got.Charging || !got.Food {
		t.Errorf("flags = %+v, want fuel+charging+food", got)
	}
	if got.Open24h || got.DogFriendly {
		t.Errorf("flags = %+v, want no open24h/dog without supporting tags", got)
	}
}

func TestEnrichDataset_TagsOnStop(t *testing.T) {
	t.Parallel()
	ds := overpass.Dataset{
		Stops: []overpass.Stop{
			{
				OSMID: 1, Kind: "services", Pos: geo.LatLng{Lat: 48, Lon: 11},
				Tags: map[string]string{"opening_hours": "24/7", "dog": "yes"},
			},
		},
	}
	overpass.EnrichDataset(&ds)

	flags := ds.Stops[0].Amenities
	if !flags.Open24h {
		t.Error("expected Open24h from opening_hours=24/7 tag")
	}
	if !flags.DogFriendly {
		t.Error("expected DogFriendly from dog=yes tag")
	}
	if !flags.Food {
		t.Error("services kind should default Food=true")
	}
	if !flags.Toilets {
		t.Error("services kind should default Toilets=true")
	}
}

func TestEnrichDataset_FarAmenitiesIgnored(t *testing.T) {
	t.Parallel()
	ds := overpass.Dataset{
		Stops: []overpass.Stop{
			{OSMID: 1, Kind: "rest_area", Pos: geo.LatLng{Lat: 48, Lon: 11}},
		},
		Amenities: []overpass.AmenityNode{
			{Kind: "fuel", Pos: geo.LatLng{Lat: 48, Lon: 11.5}},
		},
	}
	overpass.EnrichDataset(&ds)
	if ds.Stops[0].Amenities.Fuel {
		t.Error("far amenity should not set fuel flag")
	}
}

func TestEnrichDataset_RestAreaWithoutAmenitiesGetsNothing(t *testing.T) {
	t.Parallel()
	ds := overpass.Dataset{
		Stops: []overpass.Stop{
			{OSMID: 1, Kind: "rest_area", Pos: geo.LatLng{Lat: 48, Lon: 11}},
		},
	}
	overpass.EnrichDataset(&ds)
	flags := ds.Stops[0].Amenities
	if flags.Fuel || flags.Charging || flags.Food || flags.Toilets {
		t.Errorf("rest_area without amenities should have no flags, got %+v", flags)
	}
}

func TestEnrichDataset_NilSafe(t *testing.T) {
	t.Parallel()
	overpass.EnrichDataset(nil)
}
