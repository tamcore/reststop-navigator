// Command stubbedoverpass is a tiny HTTP server that mimics the public
// Overpass API by serving canned per-country JSON fixtures. Used by
// docker-compose.dev.yaml and the e2e CI job so the stack never depends on
// the real overpass-api.de.
package main

import (
	"embed"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed fixtures/*.json
var fixtures embed.FS

const defaultAddr = ":7000"

func main() {
	addr := os.Getenv("STUBBED_OVERPASS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           http.HandlerFunc(handle),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("stubbed-overpass listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// bbox is a south,west,north,east tuple in WGS84 degrees.
type bbox struct {
	south, west, north, east float64
}

// countryBBoxes are the full-country bboxes from internal/overpass/queries.go.
// Kept in sync manually so this stub stays a single self-contained binary.
var countryBBoxes = []struct {
	iso  string
	bbox bbox
}{
	{"DE", bbox{south: 47.27, west: 5.86, north: 55.10, east: 15.04}},
	{"AT", bbox{south: 46.37, west: 9.53, north: 49.02, east: 17.16}},
	{"SK", bbox{south: 47.73, west: 16.84, north: 49.61, east: 22.57}},
	{"CZ", bbox{south: 48.55, west: 12.09, north: 51.06, east: 18.86}},
}

// bboxRe captures the first (south,west,north,east) tuple in an Overpass QL
// body. The backend's BBoxQuery emits floats with Go's %g formatting, so the
// pattern accepts integers and decimals (with optional sign).
var bboxRe = regexp.MustCompile(`\(\s*(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)\s*\)`)

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	q := decodeQuery(body)

	// Legacy area-filter format: keep working for any older client.
	for _, iso := range []string{"DE", "AT", "SK", "CZ"} {
		needle := `"ISO3166-1"="` + iso + `"`
		if strings.Contains(q, needle) {
			serveFixture(w, iso)
			return
		}
	}

	// New bbox-quadrant format: locate the tuple and pick whichever country's
	// full bbox contains its centre.
	if iso, ok := matchCountryByBBox(q); ok {
		serveFixture(w, iso)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"elements":[]}`))
}

// decodeQuery extracts the Overpass QL string from a form-encoded body.
// Falls back to the raw body if it isn't form-encoded — the legacy area
// matcher handled URL-encoded substrings directly, so we accept either shape.
func decodeQuery(body []byte) string {
	values, err := url.ParseQuery(string(body))
	if err == nil {
		if data := values.Get("data"); data != "" {
			return data
		}
	}
	return string(body)
}

// matchCountryByBBox parses the first (south,west,north,east) tuple in q and
// returns the ISO code whose full-country bbox contains the tuple's centre.
func matchCountryByBBox(q string) (string, bool) {
	m := bboxRe.FindStringSubmatch(q)
	if m == nil {
		return "", false
	}
	south, err1 := strconv.ParseFloat(m[1], 64)
	west, err2 := strconv.ParseFloat(m[2], 64)
	north, err3 := strconv.ParseFloat(m[3], 64)
	east, err4 := strconv.ParseFloat(m[4], 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return "", false
	}
	centerLat := (south + north) / 2
	centerLon := (west + east) / 2
	for _, c := range countryBBoxes {
		if centerLat >= c.bbox.south && centerLat <= c.bbox.north &&
			centerLon >= c.bbox.west && centerLon <= c.bbox.east {
			return c.iso, true
		}
	}
	return "", false
}

func serveFixture(w http.ResponseWriter, iso string) {
	data, err := fixtures.ReadFile("fixtures/" + iso + ".json")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"elements":[]}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
