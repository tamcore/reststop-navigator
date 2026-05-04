// Package geo provides geographic primitives for road and rest-stop matching.
package geo

import "math"

// LatLng is a WGS84 coordinate pair in decimal degrees.
type LatLng struct {
	Lat float64
	Lon float64
}

// Bearing returns the initial great-circle bearing in degrees from a to b,
// normalised to [0, 360). 0 = north, 90 = east.
func Bearing(a, b LatLng) float64 {
	phi1 := degToRad(a.Lat)
	phi2 := degToRad(b.Lat)
	dLon := degToRad(b.Lon - a.Lon)

	y := math.Sin(dLon) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLon)

	return math.Mod(radToDeg(math.Atan2(y, x))+360, 360)
}

// AngleDiff returns the smallest unsigned angular difference between two
// bearings in degrees. Inputs may be any real number (negative, > 360);
// the result is always in [0, 180].
func AngleDiff(a, b float64) float64 {
	d := math.Mod(a-b, 360)
	if d < 0 {
		d += 360
	}
	if d > 180 {
		d = 360 - d
	}
	return d
}

func degToRad(d float64) float64 { return d * math.Pi / 180 }

func radToDeg(r float64) float64 { return r * 180 / math.Pi }
