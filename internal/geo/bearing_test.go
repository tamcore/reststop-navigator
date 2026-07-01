package geo_test

import (
	"math"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

func TestBearing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b geo.LatLng
		want float64
		tol  float64
	}{
		{"north from equator", geo.LatLng{Lat: 0, Lon: 0}, geo.LatLng{Lat: 1, Lon: 0}, 0, 0.01},
		{"east at equator", geo.LatLng{Lat: 0, Lon: 0}, geo.LatLng{Lat: 0, Lon: 1}, 90, 0.01},
		{"south to equator", geo.LatLng{Lat: 1, Lon: 0}, geo.LatLng{Lat: 0, Lon: 0}, 180, 0.01},
		{"west at equator", geo.LatLng{Lat: 0, Lon: 1}, geo.LatLng{Lat: 0, Lon: 0}, 270, 0.01},
		// A8 Stuttgart -> Munich initial great-circle bearing.
		{"a8 stuttgart-munich", geo.LatLng{Lat: 48.78, Lon: 9.18}, geo.LatLng{Lat: 48.14, Lon: 11.58}, 111.0, 0.5},
		// A1 Vienna -> Salzburg initial great-circle bearing (WSW).
		{"a1 vienna-salzburg", geo.LatLng{Lat: 48.21, Lon: 16.37}, geo.LatLng{Lat: 47.81, Lon: 13.05}, 260.4, 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := geo.Bearing(tc.a, tc.b)
			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("Bearing(%+v,%+v) = %.4f, want %.4f +/- %.4f", tc.a, tc.b, got, tc.want, tc.tol)
			}
		})
	}
}

func TestBearingNormalisedToZero360(t *testing.T) {
	t.Parallel()

	got := geo.Bearing(geo.LatLng{Lat: 48, Lon: 11}, geo.LatLng{Lat: 47, Lon: 10})
	if got < 0 || got >= 360 {
		t.Fatalf("Bearing out of [0,360): %v", got)
	}
}

func TestAngleDiff(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b float64
		want float64
	}{
		{"identical", 90, 90, 0},
		{"opposite", 0, 180, 180},
		{"wrap small +", 350, 10, 20},
		{"wrap small -", 10, 350, 20},
		{"east vs ese", 90, 110, 20},
		{"clamp 360 == 0", 360, 0, 0},
		{"negative input", -10, 10, 20},
		{"large input", 720 + 30, 30, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := geo.AngleDiff(tc.a, tc.b)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("AngleDiff(%v,%v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
