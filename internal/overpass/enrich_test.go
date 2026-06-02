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

// snapTestDataset has two parallel A8 carriageways ~110m apart at lat=48.000
// (east-bound) and lat=48.001 (west-bound). Stops are placed clearly on one
// side so distance to the nearest way is unambiguous.
func snapTestDataset() overpass.Dataset {
	return overpass.Dataset{
		Ways: []geo.Way{
			{
				ID: "way/1001", Ref: "A8", Oneway: true,
				Coords: []geo.LatLng{
					{Lat: 48.000, Lon: 11.000},
					{Lat: 48.000, Lon: 11.010},
					{Lat: 48.000, Lon: 11.020},
				},
			},
			{
				ID: "way/1002", Ref: "A8", Oneway: true,
				Coords: []geo.LatLng{
					{Lat: 48.001, Lon: 11.020},
					{Lat: 48.001, Lon: 11.010},
					{Lat: 48.001, Lon: 11.000},
				},
			},
		},
	}
}

func TestSnapStopsToMotorways_EastboundCarriageway(t *testing.T) {
	t.Parallel()
	ds := snapTestDataset()
	// Stop south of the east-bound way (48.000) — clearly closer to east way.
	ds.Stops = []overpass.Stop{
		{OSMID: 1, Kind: "rest_area", Pos: geo.LatLng{Lat: 47.999, Lon: 11.010}},
	}
	overpass.EnrichDataset(&ds)

	s := ds.Stops[0]
	if s.HighwayRef != "A8" {
		t.Errorf("HighwayRef = %q, want A8", s.HighwayRef)
	}
	if geo.AngleDiff(s.HighwayBearing, 90) > 10 {
		t.Errorf("HighwayBearing = %.1f, want ≈90° (east-bound)", s.HighwayBearing)
	}
}

func TestSnapStopsToMotorways_WestboundCarriageway(t *testing.T) {
	t.Parallel()
	ds := snapTestDataset()
	// Stop north of the west-bound way (48.001) — clearly closer to west way.
	ds.Stops = []overpass.Stop{
		{OSMID: 2, Kind: "rest_area", Pos: geo.LatLng{Lat: 48.002, Lon: 11.010}},
	}
	overpass.EnrichDataset(&ds)

	s := ds.Stops[0]
	if s.HighwayRef != "A8" {
		t.Errorf("HighwayRef = %q, want A8", s.HighwayRef)
	}
	if geo.AngleDiff(s.HighwayBearing, 270) > 10 {
		t.Errorf("HighwayBearing = %.1f, want ≈270° (west-bound)", s.HighwayBearing)
	}
}

func TestSnapStopsToMotorways_TooFarFromAnyWay(t *testing.T) {
	t.Parallel()
	ds := snapTestDataset()
	// Stop far north — more than 350 m from either way.
	ds.Stops = []overpass.Stop{
		{OSMID: 3, Kind: "rest_area", Pos: geo.LatLng{Lat: 49.000, Lon: 11.010}},
	}
	overpass.EnrichDataset(&ds)

	if ds.Stops[0].HighwayRef != "" {
		t.Errorf("HighwayRef = %q, want empty (stop beyond snap radius)", ds.Stops[0].HighwayRef)
	}
}

func TestSnapStopsToMotorways_EmptyRefWaySkipped(t *testing.T) {
	t.Parallel()
	ds := overpass.Dataset{
		Ways: []geo.Way{
			{
				ID: "way/9999", Ref: "", Oneway: true,
				Coords: []geo.LatLng{
					{Lat: 48.000, Lon: 11.000},
					{Lat: 48.000, Lon: 11.010},
				},
			},
		},
		Stops: []overpass.Stop{
			{OSMID: 4, Kind: "rest_area", Pos: geo.LatLng{Lat: 48.000, Lon: 11.005}},
		},
	}
	overpass.EnrichDataset(&ds)

	if ds.Stops[0].HighwayRef != "" {
		t.Errorf("HighwayRef = %q, want empty (way has no ref)", ds.Stops[0].HighwayRef)
	}
}
