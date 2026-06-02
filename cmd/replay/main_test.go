package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/gpx"
)

func TestLoadGPX_Synthetic(t *testing.T) {
	f, err := os.Open("../../testdata/gpx/synthetic-de-a8.gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	track, err := gpx.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	pts := track.Points
	if len(pts) != 11 {
		t.Fatalf("expected 11 trkpts, got %d", len(pts))
	}
	if pts[0].Lat != 48.000 || pts[0].Lon != 11.000 {
		t.Errorf("first point = %+v", pts[0])
	}
	if pts[len(pts)-1].Lon != 11.020 {
		t.Errorf("last point = %+v", pts[len(pts)-1])
	}
}

func TestCallAPI_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"country":"DE",
			"road":{"ref":"A8","direction":"E"},
			"stops":[
				{"name":"Aichen","distance_m":4200}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	out := callAPI(srv.Client(), srv.URL, 48.0, 11.005, 90, 120)
	if !strings.Contains(out, "A8") || !strings.Contains(out, "Aichen") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestCallAPI_OffHighway(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"reason":"off-highway-or-wrong-direction","stops":[]}`))
	}))
	t.Cleanup(srv.Close)

	out := callAPI(srv.Client(), srv.URL, 0, 0, 0, 0)
	if out != "off-highway-or-wrong-direction" {
		t.Errorf("got %q", out)
	}
}

// TestReplayAgainstUserGPX is gated on RESTSTOP_GPX_FIXTURE so CI never reaches
// for the user's personal-data file. Set the env var locally to exercise the
// loader against ~/Downloads/route-1.gpx.
func TestReplayAgainstUserGPX(t *testing.T) {
	path := os.Getenv("RESTSTOP_GPX_FIXTURE")
	if path == "" {
		t.Skip("RESTSTOP_GPX_FIXTURE not set; skipping personal-data replay")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	track, err := gpx.Parse(f)
	if err != nil {
		t.Fatalf("load %q: %v", path, err)
	}
	if len(track.Points) < 2 {
		t.Fatalf("need at least 2 trkpts in user fixture, got %d", len(track.Points))
	}
	t.Logf("user GPX %s: %d trkpts", path, len(track.Points))
}
