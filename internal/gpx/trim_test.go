package gpx_test

import (
	"errors"
	"os"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/gpx"
)

func TestTrimByDistance_LongTrack(t *testing.T) {
	f, err := os.Open("../../testdata/gpx/synthetic-de-a8-long.gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tr, err := gpx.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	trimmed, err := gpx.TrimByDistance(tr, 15000)
	if err != nil {
		t.Fatalf("TrimByDistance: %v", err)
	}

	// First trimmed point must be clearly inside: beyond 15 km from original start.
	// At lat=48°, 0.01° lon ≈ 745 m → 15 km ≈ 20 steps → lon > 11.15 expected.
	if trimmed.Points[0].Lon <= 11.10 {
		t.Errorf("trimmed start lon = %.3f, expected > 11.10 (≥15 km in)", trimmed.Points[0].Lon)
	}

	// Last trimmed point must be clearly inside: more than 15 km before original end.
	// Original last lon = 11.000 + 199*0.01 = 12.990. Trimmed last should be < 12.85.
	last := trimmed.Points[len(trimmed.Points)-1]
	if last.Lon >= 12.85 {
		t.Errorf("trimmed end lon = %.3f, expected < 12.85 (≥15 km from end)", last.Lon)
	}
}

func TestTrimByDistance_TooShort(t *testing.T) {
	f, err := os.Open("../../testdata/gpx/synthetic-de-a8.gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tr, err := gpx.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gpx.TrimByDistance(tr, 15000)
	if !errors.Is(err, gpx.ErrTrackTooShort) {
		t.Errorf("want ErrTrackTooShort, got %v", err)
	}
}
