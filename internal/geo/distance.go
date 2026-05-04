package geo

import "math"

// earthRadiusMeters is the WGS84 mean radius used by Distance.
const earthRadiusMeters = 6371008.8

// Distance returns the great-circle distance between two coordinates in meters.
func Distance(a, b LatLng) float64 {
	phi1 := degToRad(a.Lat)
	phi2 := degToRad(b.Lat)
	dPhi := degToRad(b.Lat - a.Lat)
	dLam := degToRad(b.Lon - a.Lon)

	h := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLam/2)*math.Sin(dLam/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// DistancePointToSegment returns the shortest distance in meters from p to the
// great-circle segment a-b. Uses an equirectangular projection centred on the
// segment; accurate to within a few cm for road-scale segments (< 10 km).
func DistancePointToSegment(p, a, b LatLng) float64 {
	if a == b {
		return Distance(p, a)
	}

	midLatRad := degToRad((a.Lat + b.Lat) / 2)
	mPerDegLat := math.Pi * earthRadiusMeters / 180
	mPerDegLon := mPerDegLat * math.Cos(midLatRad)

	px := (p.Lon - a.Lon) * mPerDegLon
	py := (p.Lat - a.Lat) * mPerDegLat
	bx := (b.Lon - a.Lon) * mPerDegLon
	by := (b.Lat - a.Lat) * mPerDegLat

	segLenSq := bx*bx + by*by
	t := (px*bx + py*by) / segLenSq
	switch {
	case t < 0:
		t = 0
	case t > 1:
		t = 1
	}

	dx := px - t*bx
	dy := py - t*by
	return math.Sqrt(dx*dx + dy*dy)
}
