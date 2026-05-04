package geo_test

import (
	"testing"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

// onewayEastA8 is a small synthetic motorway segment heading east at 48N
// (lat 48.000), modelling one carriageway of the A8.
func onewayEastA8() geo.Way {
	return geo.Way{
		ID:  "way/east",
		Ref: "A8",
		Coords: []geo.LatLng{
			{Lat: 48.000, Lon: 11.000},
			{Lat: 48.000, Lon: 11.010},
			{Lat: 48.000, Lon: 11.020},
		},
		Oneway: true,
	}
}

// onewayWestA8 is the opposite carriageway: traced east to west at lat 48.001
// (~111 m north of the east carriageway).
func onewayWestA8() geo.Way {
	return geo.Way{
		ID:  "way/west",
		Ref: "A8",
		Coords: []geo.LatLng{
			{Lat: 48.001, Lon: 11.020},
			{Lat: 48.001, Lon: 11.010},
			{Lat: 48.001, Lon: 11.000},
		},
		Oneway: true,
	}
}

// twoWayBundesstrasse is a small bidirectional B-road segment heading north.
func twoWayBundesstrasse() geo.Way {
	return geo.Way{
		ID:  "way/btwo",
		Ref: "B12",
		Coords: []geo.LatLng{
			{Lat: 48.000, Lon: 11.500},
			{Lat: 48.005, Lon: 11.500},
		},
		Oneway: false,
	}
}

func TestNearestForwardWay_OnewayMatchesAlignedHeading(t *testing.T) {
	t.Parallel()

	ways := []geo.Way{onewayEastA8(), onewayWestA8()}
	pos := geo.LatLng{Lat: 48.000, Lon: 11.005}
	heading := 90.0

	m, ok := geo.NearestForwardWay(pos, heading, ways, geo.MatchOpts{})
	if !ok {
		t.Fatal("expected a match, got none")
	}
	if m.Way.ID != "way/east" {
		t.Errorf("matched way = %q, want way/east", m.Way.ID)
	}
	if m.SegmentIndex != 0 {
		t.Errorf("segment index = %d, want 0", m.SegmentIndex)
	}
	if !m.Forward {
		t.Error("expected Forward=true on aligned oneway match")
	}
}

func TestNearestForwardWay_OnewayRejectsOppositeHeading(t *testing.T) {
	t.Parallel()

	ways := []geo.Way{onewayEastA8()}
	pos := geo.LatLng{Lat: 48.000, Lon: 11.005}
	heading := 270.0

	if _, ok := geo.NearestForwardWay(pos, heading, ways, geo.MatchOpts{}); ok {
		t.Fatal("expected no match (oneway, opposite direction)")
	}
}

func TestNearestForwardWay_PicksCorrectCarriageway(t *testing.T) {
	t.Parallel()

	ways := []geo.Way{onewayEastA8(), onewayWestA8()}

	pos := geo.LatLng{Lat: 48.0001, Lon: 11.005}
	m, ok := geo.NearestForwardWay(pos, 90, ways, geo.MatchOpts{})
	if !ok || m.Way.ID != "way/east" {
		t.Fatalf("east-bound match = %v ok=%v, want way/east", m.Way.ID, ok)
	}

	pos = geo.LatLng{Lat: 48.0009, Lon: 11.005}
	m, ok = geo.NearestForwardWay(pos, 270, ways, geo.MatchOpts{})
	if !ok || m.Way.ID != "way/west" {
		t.Fatalf("west-bound match = %v ok=%v, want way/west", m.Way.ID, ok)
	}
}

func TestNearestForwardWay_TwoWayAcceptsBothDirections(t *testing.T) {
	t.Parallel()

	ways := []geo.Way{twoWayBundesstrasse()}
	pos := geo.LatLng{Lat: 48.0025, Lon: 11.500}

	mNorth, ok := geo.NearestForwardWay(pos, 0, ways, geo.MatchOpts{})
	if !ok || mNorth.Way.ID != "way/btwo" {
		t.Fatalf("northbound match = %v ok=%v", mNorth.Way.ID, ok)
	}
	if !mNorth.Forward {
		t.Error("northbound on north-traced way should be Forward=true")
	}

	mSouth, ok := geo.NearestForwardWay(pos, 180, ways, geo.MatchOpts{})
	if !ok || mSouth.Way.ID != "way/btwo" {
		t.Fatalf("southbound match = %v ok=%v", mSouth.Way.ID, ok)
	}
	if mSouth.Forward {
		t.Error("southbound on north-traced way should be Forward=false")
	}
}

func TestNearestForwardWay_OutOfRangeReturnsNoMatch(t *testing.T) {
	t.Parallel()

	ways := []geo.Way{onewayEastA8()}
	pos := geo.LatLng{Lat: 48.1, Lon: 11.005}
	if _, ok := geo.NearestForwardWay(pos, 90, ways, geo.MatchOpts{}); ok {
		t.Fatal("expected no match for far-away position")
	}
}

func TestNearestForwardWay_EmptyAndDegenerate(t *testing.T) {
	t.Parallel()

	if _, ok := geo.NearestForwardWay(geo.LatLng{}, 0, nil, geo.MatchOpts{}); ok {
		t.Error("nil ways: expected no match")
	}

	degenerate := []geo.Way{{ID: "stub", Coords: []geo.LatLng{{Lat: 48, Lon: 11}}, Oneway: true}}
	if _, ok := geo.NearestForwardWay(geo.LatLng{Lat: 48, Lon: 11}, 0, degenerate, geo.MatchOpts{}); ok {
		t.Error("single-coord way: expected no match")
	}
}

func TestNearestForwardWay_CustomOpts(t *testing.T) {
	t.Parallel()

	ways := []geo.Way{onewayEastA8()}
	pos := geo.LatLng{Lat: 48.0009, Lon: 11.005}

	if _, ok := geo.NearestForwardWay(pos, 90, ways, geo.MatchOpts{MaxDistanceMeters: 80}); ok {
		t.Error("100m north should be outside 80m cap")
	}
	if _, ok := geo.NearestForwardWay(pos, 90, ways, geo.MatchOpts{MaxDistanceMeters: 200}); !ok {
		t.Error("100m north should be inside 200m cap")
	}
}
