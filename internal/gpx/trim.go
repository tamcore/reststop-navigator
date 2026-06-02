package gpx

import (
	"errors"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

// ErrTrackTooShort is returned when the track is too short to trim.
var ErrTrackTooShort = errors.New("gpx: track too short to trim")

// TrimByDistance returns the sub-track with skipFromEnds meters removed from
// both the start and the end. Returns ErrTrackTooShort if the total length is
// less than 2×skipFromEnds.
func TrimByDistance(t Track, skipFromEnds float64) (Track, error) {
	pts := t.Points
	if len(pts) < 2 {
		return Track{}, ErrTrackTooShort
	}

	cumDist := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		cumDist[i] = cumDist[i-1] + geo.Distance(
			geo.LatLng{Lat: pts[i-1].Lat, Lon: pts[i-1].Lon},
			geo.LatLng{Lat: pts[i].Lat, Lon: pts[i].Lon},
		)
	}
	total := cumDist[len(pts)-1]
	if total < 2*skipFromEnds {
		return Track{}, ErrTrackTooShort
	}

	iStart := 0
	for i, d := range cumDist {
		if d >= skipFromEnds {
			iStart = i
			break
		}
	}

	iEnd := len(pts) - 1
	for i := len(pts) - 1; i >= 0; i-- {
		if total-cumDist[i] >= skipFromEnds {
			iEnd = i
			break
		}
	}

	if iStart >= iEnd {
		return Track{}, ErrTrackTooShort
	}

	result := make([]Point, iEnd-iStart+1)
	copy(result, pts[iStart:iEnd+1])
	return Track{Points: result}, nil
}
