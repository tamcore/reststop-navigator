package gpx_test

import (
	"os"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/gpx"
)

func TestParse_SyntheticA8(t *testing.T) {
	f, err := os.Open("../../testdata/gpx/synthetic-de-a8.gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tr, err := gpx.Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(tr.Points) != 11 {
		t.Errorf("got %d points, want 11", len(tr.Points))
	}
	if tr.Points[0].Lat != 48.0 {
		t.Errorf("first lat = %f, want 48.0", tr.Points[0].Lat)
	}
	if tr.Points[0].Lon != 11.0 {
		t.Errorf("first lon = %f, want 11.0", tr.Points[0].Lon)
	}
	if tr.Points[10].Lon != 11.02 {
		t.Errorf("last lon = %f, want 11.02", tr.Points[10].Lon)
	}
}
