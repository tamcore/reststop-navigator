package geo_test

import (
	"math"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

func TestDistance(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b geo.LatLng
		want float64 // meters
		tol  float64
	}{
		{"identical", geo.LatLng{Lat: 0, Lon: 0}, geo.LatLng{Lat: 0, Lon: 0}, 0, 1e-3},
		// 1 deg latitude ~= 111.20 km along any meridian.
		{"1 deg lat", geo.LatLng{Lat: 0, Lon: 0}, geo.LatLng{Lat: 1, Lon: 0}, 111195, 100},
		// 1 deg longitude at the equator ~= 111.32 km.
		{"1 deg lon at equator", geo.LatLng{Lat: 0, Lon: 0}, geo.LatLng{Lat: 0, Lon: 1}, 111195, 100},
		// 1 deg longitude at 48 deg N ~= 74.4 km.
		{"1 deg lon at 48N", geo.LatLng{Lat: 48, Lon: 0}, geo.LatLng{Lat: 48, Lon: 1}, 74400, 200},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := geo.Distance(tc.a, tc.b)
			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("Distance(%+v,%+v) = %.2f, want %.2f +/- %.2f", tc.a, tc.b, got, tc.want, tc.tol)
			}
		})
	}
}

func TestDistancePointToSegment(t *testing.T) {
	t.Parallel()

	a := geo.LatLng{Lat: 48, Lon: 11}
	b := geo.LatLng{Lat: 48, Lon: 11.01} // ~745 m east at 48N

	cases := []struct {
		name    string
		p, a, b geo.LatLng
		want    float64 // meters
		tol     float64
	}{
		{"on endpoint a", a, a, b, 0, 1e-6},
		{"on endpoint b", b, a, b, 0, 1e-6},
		{"on midpoint", geo.LatLng{Lat: 48, Lon: 11.005}, a, b, 0, 1},
		// Point ~111 m north of midpoint (0.001 deg lat at this latitude).
		{"perpendicular north", geo.LatLng{Lat: 48.001, Lon: 11.005}, a, b, 111.2, 1},
		{"perpendicular south", geo.LatLng{Lat: 47.999, Lon: 11.005}, a, b, 111.2, 1},
		// Point past b (east beyond segment) — clamps to b.
		{"past b east", geo.LatLng{Lat: 48, Lon: 11.02}, a, b, 745, 5},
		// Point before a (west of segment) — clamps to a.
		{"before a west", geo.LatLng{Lat: 48, Lon: 10.99}, a, b, 745, 5},
		// Degenerate: a == b → falls back to point-to-point distance.
		{"degenerate a==b", geo.LatLng{Lat: 48.001, Lon: 11}, a, a, 111.2, 1},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := geo.DistancePointToSegment(tc.p, tc.a, tc.b)
			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("DistancePointToSegment(%+v,%+v,%+v) = %.4f, want %.4f +/- %.4f",
					tc.p, tc.a, tc.b, got, tc.want, tc.tol)
			}
		})
	}
}
