package geo

import (
	"math"
	"sort"
)

// WayProjection describes the closest point on a Way to a query point.
type WayProjection struct {
	SegmentIndex int     // index of the matched segment within Way.Coords
	AlongMeters  float64 // along the matched segment, from its start
	Cumulative   float64 // along the entire way from its start
	Point        LatLng  // the projected point in lat/lon
	Distance     float64 // perpendicular distance from query point to Point
}

// StopOnWay is a rest-stop candidate to test against a matched way.
type StopOnWay struct {
	StopID string
	Pos    LatLng
}

// AheadStop is a stop that lies ahead of the user along a matched way.
type AheadStop struct {
	StopID   string
	Pos      LatLng
	Distance float64 // along-way meters from user position to stop projection
}

// ProjectOntoWay finds the closest point on w to pos and returns the
// cumulative distance from the way's start, plus segment-relative diagnostics.
// Returns ok=false for ways with fewer than two coordinates.
func ProjectOntoWay(pos LatLng, w *Way) (WayProjection, bool) {
	if w == nil || len(w.Coords) < 2 {
		return WayProjection{}, false
	}

	var best WayProjection
	bestDist := math.Inf(1)
	cum := 0.0

	for s := 0; s < len(w.Coords)-1; s++ {
		a, b := w.Coords[s], w.Coords[s+1]

		midLatRad := degToRad((a.Lat + b.Lat) / 2)
		mPerDegLat := math.Pi * earthRadiusMeters / 180
		mPerDegLon := mPerDegLat * math.Cos(midLatRad)

		bx := (b.Lon - a.Lon) * mPerDegLon
		by := (b.Lat - a.Lat) * mPerDegLat
		px := (pos.Lon - a.Lon) * mPerDegLon
		py := (pos.Lat - a.Lat) * mPerDegLat

		segLen := math.Sqrt(bx*bx + by*by)
		var t float64
		if segLen > 0 {
			t = (px*bx + py*by) / (bx*bx + by*by)
		}
		switch {
		case t < 0:
			t = 0
		case t > 1:
			t = 1
		}

		closeX, closeY := t*bx, t*by
		dx, dy := px-closeX, py-closeY
		d := math.Sqrt(dx*dx + dy*dy)

		if d < bestDist {
			bestDist = d
			best = WayProjection{
				SegmentIndex: s,
				AlongMeters:  t * segLen,
				Cumulative:   cum + t*segLen,
				Point: LatLng{
					Lat: a.Lat + closeY/mPerDegLat,
					Lon: a.Lon + closeX/mPerDegLon,
				},
				Distance: d,
			}
		}
		cum += segLen
	}
	return best, true
}

// FilterAhead returns stops that lie ahead of the user along the matched
// way's heading, ranked by straight-line distance.
//
// Computes the matched segment's bearing (reversed if the user is traversing
// the way backwards) and keeps only stops whose displacement from the user
// has a positive component in that direction. More robust than along-way
// projection when stops are tagged on different OSM ways from the user's
// matched motorway way (typical: rest areas on connector ways or as
// stand-alone nodes off the carriageway).
func FilterAhead(m Match, userPos LatLng, stops []StopOnWay) []AheadStop {
	if m.Way == nil || len(m.Way.Coords) < m.SegmentIndex+2 {
		return nil
	}
	a, b := m.Way.Coords[m.SegmentIndex], m.Way.Coords[m.SegmentIndex+1]
	if !m.Forward {
		a, b = b, a
	}
	headingRad := degToRad(Bearing(a, b))
	hx, hy := math.Sin(headingRad), math.Cos(headingRad)
	mPerDegLat := math.Pi * earthRadiusMeters / 180

	out := make([]AheadStop, 0, len(stops))
	for _, s := range stops {
		midLatRad := degToRad((userPos.Lat + s.Pos.Lat) / 2)
		dx := (s.Pos.Lon - userPos.Lon) * mPerDegLat * math.Cos(midLatRad)
		dy := (s.Pos.Lat - userPos.Lat) * mPerDegLat
		if dx*hx+dy*hy <= 0 {
			continue
		}
		out = append(out, AheadStop{
			StopID:   s.StopID,
			Pos:      s.Pos,
			Distance: Distance(userPos, s.Pos),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Distance < out[j].Distance })
	return out
}
