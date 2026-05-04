package geo

import "math"

// Way is a polyline geometry with the metadata needed for road matching.
// Coords are ordered start → end. For Oneway=true, traversal is only valid
// in coord order (the segment bearing equals the lane direction).
type Way struct {
	ID     string
	Coords []LatLng
	Oneway bool
	Ref    string
	Name   string
}

// MatchOpts tunes NearestForwardWay. Zero-valued fields fall back to defaults.
type MatchOpts struct {
	MaxDistanceMeters float64 // default 80
	MaxAngleDegrees   float64 // default 60
}

// Match describes a successful road match.
type Match struct {
	Way          *Way
	SegmentIndex int
	Distance     float64 // meters from user position to the matched segment
	Forward      bool    // true if user travels in coord order
}

const (
	defaultMaxDistanceMeters = 80.0
	defaultMaxAngleDegrees   = 60.0
)

// NearestForwardWay finds the nearest way segment that is consistent with the
// user's heading. For Oneway ways, only coord-order direction is accepted;
// for two-way ways either direction matches and Forward is set accordingly.
//
// Returns ok=false if no segment is within MaxDistanceMeters or no segment
// passes the angle filter.
func NearestForwardWay(pos LatLng, heading float64, ways []Way, opts MatchOpts) (Match, bool) {
	maxDist := opts.MaxDistanceMeters
	if maxDist <= 0 {
		maxDist = defaultMaxDistanceMeters
	}
	maxAngle := opts.MaxAngleDegrees
	if maxAngle <= 0 {
		maxAngle = defaultMaxAngleDegrees
	}

	var best Match
	bestDist := math.Inf(1)

	for i := range ways {
		w := &ways[i]
		if len(w.Coords) < 2 {
			continue
		}
		for s := 0; s < len(w.Coords)-1; s++ {
			a, b := w.Coords[s], w.Coords[s+1]
			d := DistancePointToSegment(pos, a, b)
			if d > maxDist || d >= bestDist {
				continue
			}

			forwardBearing := Bearing(a, b)
			forwardOK := AngleDiff(heading, forwardBearing) <= maxAngle
			backwardOK := false
			if !w.Oneway {
				backwardOK = AngleDiff(heading, Bearing(b, a)) <= maxAngle
			}
			if !forwardOK && !backwardOK {
				continue
			}

			bestDist = d
			best = Match{
				Way:          w,
				SegmentIndex: s,
				Distance:     d,
				Forward:      forwardOK,
			}
		}
	}

	if best.Way == nil {
		return Match{}, false
	}
	return best, true
}
