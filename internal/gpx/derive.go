package gpx

import "github.com/tamcore/reststop-navigator/internal/geo"

// DemoPoint is a single point in the demo-replay JSON format consumed by the
// frontend geo store.
type DemoPoint struct {
	DelayMS  int64   `json:"delay_ms"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Heading  float64 `json:"heading"`
	Speed    float64 `json:"speed"`
	Accuracy float64 `json:"accuracy"`
}

// ToDemoPoints converts a Track into the flat demo-point list expected by the
// frontend. delay_ms is the delta from the previous point (0 for the first).
// heading uses the segment to the next point for the first point and the
// segment from the previous point for all subsequent points. speed is in km/h.
// accuracy is a constant applied to every point.
func ToDemoPoints(t Track, accuracy float64) []DemoPoint {
	pts := t.Points
	if len(pts) == 0 {
		return nil
	}

	result := make([]DemoPoint, len(pts))
	for i, p := range pts {
		var delayMS int64
		if i > 0 {
			delayMS = p.Time.Sub(pts[i-1].Time).Milliseconds()
		}

		var heading float64
		switch {
		case i == 0 && len(pts) > 1:
			heading = geo.Bearing(
				geo.LatLng{Lat: pts[0].Lat, Lon: pts[0].Lon},
				geo.LatLng{Lat: pts[1].Lat, Lon: pts[1].Lon},
			)
		case i > 0:
			heading = geo.Bearing(
				geo.LatLng{Lat: pts[i-1].Lat, Lon: pts[i-1].Lon},
				geo.LatLng{Lat: p.Lat, Lon: p.Lon},
			)
		}

		var speed float64
		if i > 0 {
			dt := p.Time.Sub(pts[i-1].Time).Seconds()
			if dt > 0 {
				dist := geo.Distance(
					geo.LatLng{Lat: pts[i-1].Lat, Lon: pts[i-1].Lon},
					geo.LatLng{Lat: p.Lat, Lon: p.Lon},
				)
				speed = dist / dt * 3.6
			}
		}

		result[i] = DemoPoint{
			DelayMS:  delayMS,
			Lat:      p.Lat,
			Lon:      p.Lon,
			Heading:  heading,
			Speed:    speed,
			Accuracy: accuracy,
		}
	}
	return result
}
