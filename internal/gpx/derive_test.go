package gpx_test

import (
	"math"
	"testing"
	"time"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/gpx"
)

func TestToDemoPoints_TwoPoints(t *testing.T) {
	t0 := time.Unix(0, 0).UTC()
	t1 := time.Unix(10, 0).UTC()
	p0 := gpx.Point{Lat: 48.0, Lon: 11.0, Time: t0}
	p1 := gpx.Point{Lat: 48.0, Lon: 11.01, Time: t1}
	tr := gpx.Track{Points: []gpx.Point{p0, p1}}

	pts := gpx.ToDemoPoints(tr, 15.0)

	if len(pts) != 2 {
		t.Fatalf("got %d points, want 2", len(pts))
	}

	if pts[0].DelayMS != 0 {
		t.Errorf("pts[0].DelayMS = %d, want 0", pts[0].DelayMS)
	}
	if pts[1].DelayMS != 10000 {
		t.Errorf("pts[1].DelayMS = %d, want 10000", pts[1].DelayMS)
	}

	// Both points share the same bearing on a 2-point track.
	wantBearing := geo.Bearing(
		geo.LatLng{Lat: p0.Lat, Lon: p0.Lon},
		geo.LatLng{Lat: p1.Lat, Lon: p1.Lon},
	)
	if math.Abs(pts[0].Heading-wantBearing) > 0.5 {
		t.Errorf("pts[0].Heading = %.2f, want ~%.2f", pts[0].Heading, wantBearing)
	}
	if math.Abs(pts[1].Heading-wantBearing) > 0.5 {
		t.Errorf("pts[1].Heading = %.2f, want ~%.2f", pts[1].Heading, wantBearing)
	}

	// First point: speed=0 (no prior segment).
	if pts[0].Speed != 0 {
		t.Errorf("pts[0].Speed = %.2f, want 0", pts[0].Speed)
	}

	// Second point: speed in km/h from haversine distance / elapsed seconds.
	dist := geo.Distance(
		geo.LatLng{Lat: p0.Lat, Lon: p0.Lon},
		geo.LatLng{Lat: p1.Lat, Lon: p1.Lon},
	)
	wantSpeed := dist / 10.0 * 3.6
	if math.Abs(pts[1].Speed-wantSpeed) > 0.5 {
		t.Errorf("pts[1].Speed = %.2f, want ~%.2f", pts[1].Speed, wantSpeed)
	}

	if pts[0].Accuracy != 15.0 || pts[1].Accuracy != 15.0 {
		t.Errorf("accuracy = %.1f/%.1f, want 15.0/15.0", pts[0].Accuracy, pts[1].Accuracy)
	}
}
