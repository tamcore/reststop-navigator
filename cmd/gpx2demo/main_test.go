package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConvert_SyntheticLong(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Copy fixtures into srcDir with route-*.gpx names.
	for dst, src := range map[string]string{
		"route-1.gpx":     "../../testdata/gpx/synthetic-de-a8-long.gpx",
		"route-short.gpx": "../../testdata/gpx/synthetic-de-a8.gpx",
	} {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, dst), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := convertDir(srcDir, outDir, 15000, 15.0)
	if err != nil {
		t.Fatalf("convertDir: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	// Only the long fixture qualifies (11-point short one is rejected).
	if len(entries) != 1 {
		t.Errorf("got %d output files, want 1", len(entries))
	}

	data, err := os.ReadFile(filepath.Join(outDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}

	var tj trackJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if tj.ID == "" {
		t.Error("id is empty")
	}
	if tj.Label == "" {
		t.Error("label is empty")
	}
	if tj.DurationMS <= 0 {
		t.Errorf("duration_ms = %d, want > 0", tj.DurationMS)
	}
	if len(tj.Points) < 2 {
		t.Errorf("got %d points, want >= 2", len(tj.Points))
	}

	// delay_ms must be >= 0 for every point and 0 for the first.
	if tj.Points[0].DelayMS != 0 {
		t.Errorf("points[0].delay_ms = %d, want 0", tj.Points[0].DelayMS)
	}
	for i, pt := range tj.Points {
		if pt.DelayMS < 0 {
			t.Errorf("points[%d].delay_ms = %d < 0", i, pt.DelayMS)
		}
	}
}
