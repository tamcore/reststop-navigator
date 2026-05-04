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

// FilterAhead drops stops that aren't ahead of the user's projection along the
// matched way and returns those that are sorted by ascending along-way
// distance. m.Forward controls direction: true = travel in coord order,
// false = reverse.
func FilterAhead(m Match, userPos LatLng, stops []StopOnWay) []AheadStop {
	if m.Way == nil {
		return nil
	}
	userProj, ok := ProjectOntoWay(userPos, m.Way)
	if !ok {
		return nil
	}

	out := make([]AheadStop, 0, len(stops))
	for _, s := range stops {
		sp, ok := ProjectOntoWay(s.Pos, m.Way)
		if !ok {
			continue
		}
		var dist float64
		if m.Forward {
			dist = sp.Cumulative - userProj.Cumulative
		} else {
			dist = userProj.Cumulative - sp.Cumulative
		}
		if dist <= 0 {
			continue
		}
		out = append(out, AheadStop{
			StopID:   s.StopID,
			Pos:      s.Pos,
			Distance: dist,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Distance < out[j].Distance })
	return out
}
