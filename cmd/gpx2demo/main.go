// Command gpx2demo converts GPX route files to pre-trimmed JSON demo tracks
// for the frontend round-robin replay. It globs route-*.gpx from -src,
// trims 15 km from each end, derives demo points, and writes JSON to -out.
//
// Usage:
//
//	go run ./cmd/gpx2demo -src ~/Downloads -out web/src/lib/data/tracks
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tamcore/reststop-navigator/internal/gpx"
)

type trackJSON struct {
	ID         string          `json:"id"`
	Label      string          `json:"label"`
	DurationMS int64           `json:"duration_ms"`
	Points     []gpx.DemoPoint `json:"points"`
}

var nonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeName(base string) string {
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return nonAlphanumRe.ReplaceAllString(stem, "_") + ".json"
}

func convertDir(srcDir, outDir string, skipMeters, accuracy float64) error {
	matches, err := filepath.Glob(filepath.Join(srcDir, "route-*.gpx"))
	if err != nil {
		return err
	}

	for _, path := range matches {
		f, err := os.Open(path)
		if err != nil {
			log.Printf("skip %s: open: %v", path, err)
			continue
		}

		track, err := gpx.Parse(f)
		_ = f.Close()
		if err != nil {
			log.Printf("skip %s: parse: %v", path, err)
			continue
		}

		trimmed, err := gpx.TrimByDistance(track, skipMeters)
		if errors.Is(err, gpx.ErrTrackTooShort) {
			log.Printf("skip %s: track too short to trim (< %.0f m × 2)", path, skipMeters)
			continue
		}
		if err != nil {
			log.Printf("skip %s: trim: %v", path, err)
			continue
		}

		pts := gpx.ToDemoPoints(trimmed, accuracy)

		var totalMS int64
		for _, p := range pts[1:] {
			totalMS += p.DelayMS
		}

		base := filepath.Base(path)
		id := strings.TrimSuffix(sanitizeName(base), ".json")
		label := strings.ReplaceAll(id, "_", " ")

		tj := trackJSON{
			ID:         id,
			Label:      label,
			DurationMS: totalMS,
			Points:     pts,
		}

		outPath := filepath.Join(outDir, sanitizeName(base))
		data, err := json.MarshalIndent(tj, "", "  ")
		if err != nil {
			log.Printf("skip %s: marshal: %v", path, err)
			continue
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		log.Printf("wrote %s (%d points)", outPath, len(pts))
	}
	return nil
}

func main() {
	src := flag.String("src", "", "directory containing route-*.gpx files")
	out := flag.String("out", "web/src/lib/data/tracks", "output directory for JSON track files")
	flag.Parse()

	if *src == "" {
		log.Fatal("missing -src")
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", *out, err)
	}

	if err := convertDir(*src, *out, 15000, 15.0); err != nil {
		log.Fatal(err)
	}
}
