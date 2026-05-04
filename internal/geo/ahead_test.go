package geo_test

import (
	"math"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

// straightEastA8 is a 4-segment east-bound way at 48N. Each segment is
// 0.005 deg longitude (~371 m at this latitude), total ~1485 m.
func straightEastA8() geo.Way {
	return geo.Way{
		ID:     "way/east",
		Ref:    "A8",
		Oneway: true,
		Coords: []geo.LatLng{
			{Lat: 48.000, Lon: 11.000},
			{Lat: 48.000, Lon: 11.005},
			{Lat: 48.000, Lon: 11.010},
			{Lat: 48.000, Lon: 11.015},
		},
	}
}

func TestProjectOntoWay_OnFirstSegment(t *testing.T) {
	t.Parallel()
	w := straightEastA8()
	p, ok := geo.ProjectOntoWay(geo.LatLng{Lat: 48.000, Lon: 11.0025}, &w)
	if !ok {
		t.Fatal("expected ok")
	}
	if p.SegmentIndex != 0 {
		t.Errorf("segment = %d, want 0", p.SegmentIndex)
	}
	// Halfway through segment 0 (~371 m / 2 ≈ 186 m).
	if math.Abs(p.Cumulative-186) > 5 {
		t.Errorf("cumulative = %.1f, want ~186", p.Cumulative)
	}
	if p.Distance > 1 {
		t.Errorf("expected ~0 distance from on-line point, got %.4f", p.Distance)
	}
}

func TestProjectOntoWay_OnSecondSegment(t *testing.T) {
	t.Parallel()
	w := straightEastA8()
	p, ok := geo.ProjectOntoWay(geo.LatLng{Lat: 48.000, Lon: 11.0075}, &w)
	if !ok {
		t.Fatal("expected ok")
	}
	if p.SegmentIndex != 1 {
		t.Errorf("segment = %d, want 1", p.SegmentIndex)
	}
	if math.Abs(p.Cumulative-557) > 10 {
		t.Errorf("cumulative = %.1f, want ~557", p.Cumulative)
	}
}

func TestProjectOntoWay_OffAxisProjects(t *testing.T) {
	t.Parallel()
	w := straightEastA8()
	p, ok := geo.ProjectOntoWay(geo.LatLng{Lat: 48.0009, Lon: 11.0075}, &w)
	if !ok {
		t.Fatal("expected ok")
	}
	if p.SegmentIndex != 1 {
		t.Errorf("segment = %d, want 1", p.SegmentIndex)
	}
	if math.Abs(p.Distance-100) > 2 {
		t.Errorf("perpendicular distance = %.2f, want ~100", p.Distance)
	}
	if math.Abs(p.Cumulative-557) > 10 {
		t.Errorf("cumulative = %.1f, want ~557 even when off-axis", p.Cumulative)
	}
}

func TestProjectOntoWay_DegenerateWay(t *testing.T) {
	t.Parallel()
	w := geo.Way{Coords: []geo.LatLng{{Lat: 48, Lon: 11}}}
	if _, ok := geo.ProjectOntoWay(geo.LatLng{Lat: 48, Lon: 11}, &w); ok {
		t.Error("single-coord way should not project")
	}
}

func TestFilterAhead_Forward(t *testing.T) {
	t.Parallel()
	w := straightEastA8()
	m := geo.Match{Way: &w, Forward: true}
	user := geo.LatLng{Lat: 48.000, Lon: 11.002}

	stops := []geo.StopOnWay{
		{StopID: "behind", Pos: geo.LatLng{Lat: 48.000, Lon: 11.000}},
		{StopID: "near", Pos: geo.LatLng{Lat: 48.000, Lon: 11.0075}},
		{StopID: "far", Pos: geo.LatLng{Lat: 48.000, Lon: 11.0125}},
		{StopID: "way-end", Pos: geo.LatLng{Lat: 48.000, Lon: 11.015}},
	}

	got := geo.FilterAhead(m, user, stops)
	gotIDs := []string{}
	for _, s := range got {
		gotIDs = append(gotIDs, s.StopID)
	}
	wantIDs := []string{"near", "far", "way-end"}
	if !equalSlices(gotIDs, wantIDs) {
		t.Errorf("ahead IDs = %v, want %v", gotIDs, wantIDs)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Distance < got[i-1].Distance {
			t.Errorf("not sorted ascending: %v then %v", got[i-1], got[i])
		}
	}
}

func TestFilterAhead_Backward(t *testing.T) {
	t.Parallel()
	w := straightEastA8()
	m := geo.Match{Way: &w, Forward: false}
	user := geo.LatLng{Lat: 48.000, Lon: 11.0125}
	candidates := []geo.StopOnWay{
		{StopID: "ahead-back", Pos: geo.LatLng{Lat: 48.000, Lon: 11.005}},
		{StopID: "behind-back", Pos: geo.LatLng{Lat: 48.000, Lon: 11.015}},
	}
	got := geo.FilterAhead(m, user, candidates)
	if len(got) != 1 || got[0].StopID != "ahead-back" {
		t.Fatalf("backward ahead = %v, want only ahead-back", got)
	}
}

func TestFilterAhead_Empty(t *testing.T) {
	t.Parallel()
	w := straightEastA8()
	m := geo.Match{Way: &w, Forward: true}
	got := geo.FilterAhead(m, geo.LatLng{Lat: 48, Lon: 11.005}, nil)
	if len(got) != 0 {
		t.Errorf("empty input -> empty output, got %v", got)
	}
}

func TestFilterAhead_NilWayReturnsNil(t *testing.T) {
	t.Parallel()
	got := geo.FilterAhead(geo.Match{Way: nil, Forward: true}, geo.LatLng{}, []geo.StopOnWay{
		{StopID: "x", Pos: geo.LatLng{Lat: 48, Lon: 11.005}},
	})
	if got != nil {
		t.Errorf("nil way -> nil result, got %v", got)
	}
}

func TestFilterAhead_DegenerateUserProjectionReturnsNil(t *testing.T) {
	t.Parallel()
	// Way with a single coord can't be projected onto.
	w := geo.Way{Coords: []geo.LatLng{{Lat: 48, Lon: 11}}}
	got := geo.FilterAhead(geo.Match{Way: &w, Forward: true}, geo.LatLng{Lat: 48, Lon: 11}, []geo.StopOnWay{
		{StopID: "x", Pos: geo.LatLng{Lat: 48, Lon: 11}},
	})
	if got != nil {
		t.Errorf("degenerate way -> nil result, got %v", got)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
