// Command replay walks a GPX file's <trkpt> elements in order, computes a
// heading and speed for each point from the previous one, and calls the
// upcoming-stops API. Used as a substitute for real driving during tests.
//
// The GPX path comes from -gpx or the RESTSTOP_GPX_FIXTURE env var. The API
// target comes from -target (default http://localhost:8080).
//
// Personal-data note: the user's real ~/Downloads/route-1.gpx is gitignored;
// only synthetic fixtures may be committed to testdata/gpx/.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/gpx"
)

// upcomingResponse mirrors the public API just enough for the replay summary.
type upcomingResponse struct {
	Country string `json:"country"`
	Road    *struct {
		Ref       string `json:"ref"`
		Direction string `json:"direction"`
	} `json:"road"`
	Stops []struct {
		Name      string `json:"name"`
		DistanceM int    `json:"distance_m"`
	} `json:"stops"`
	Reason string `json:"reason"`
}

func main() {
	var (
		gpxPath = flag.String("gpx", os.Getenv("RESTSTOP_GPX_FIXTURE"), "GPX file path (or set RESTSTOP_GPX_FIXTURE)")
		target  = flag.String("target", "http://localhost:8080", "API base URL")
		stride  = flag.Int("stride", 1, "process every Nth track point")
	)
	flag.Parse()

	if *gpxPath == "" {
		log.Fatal("missing -gpx (or RESTSTOP_GPX_FIXTURE)")
	}
	f, err := os.Open(*gpxPath)
	if err != nil {
		log.Fatalf("open gpx: %v", err)
	}
	track, err := gpx.Parse(f)
	_ = f.Close()
	if err != nil {
		log.Fatalf("load gpx: %v", err)
	}
	pts := track.Points
	if len(pts) < 2 {
		log.Fatalf("need at least 2 trkpts, got %d", len(pts))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for i := 1; i < len(pts); i += *stride {
		prev, cur := pts[i-1], pts[i]
		heading := geo.Bearing(geo.LatLng{Lat: prev.Lat, Lon: prev.Lon}, geo.LatLng{Lat: cur.Lat, Lon: cur.Lon})
		dt := cur.Time.Sub(prev.Time).Seconds()
		var speedKMH float64
		if dt > 0 {
			distM := geo.Distance(geo.LatLng{Lat: prev.Lat, Lon: prev.Lon}, geo.LatLng{Lat: cur.Lat, Lon: cur.Lon})
			speedKMH = distM / dt * 3.6
		}
		summary := callAPI(client, *target, cur.Lat, cur.Lon, heading, speedKMH)
		fmt.Printf("t=%s lat=%.5f lon=%.5f hdg=%.0f spd=%.0f → %s\n",
			cur.Time.Format(time.RFC3339), cur.Lat, cur.Lon, heading, speedKMH, summary)
	}
}

func callAPI(c *http.Client, target string, lat, lon, heading, speed float64) string {
	q := url.Values{}
	q.Set("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	q.Set("lon", strconv.FormatFloat(lon, 'f', 6, 64))
	q.Set("heading", strconv.FormatFloat(math.Mod(heading+360, 360), 'f', 1, 64))
	q.Set("speed", strconv.FormatFloat(speed, 'f', 0, 64))
	q.Set("limit", "3")

	resp, err := c.Get(target + "/api/stops/upcoming?" + q.Encode())
	if err != nil {
		return "ERR " + err.Error()
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	var u upcomingResponse
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "DECODE " + err.Error()
	}
	if u.Reason != "" {
		return u.Reason
	}
	roadLabel := "?"
	if u.Road != nil {
		roadLabel = u.Road.Ref + " " + u.Road.Direction
	}
	var out strings.Builder
	out.WriteString("road=" + roadLabel + " next=[")
	for i, s := range u.Stops {
		if i > 0 {
			out.WriteString(", ")
		}
		fmt.Fprintf(&out, "%s %.1fkm", s.Name, float64(s.DistanceM)/1000.0)
	}
	out.WriteString("]")
	return out.String()
}
