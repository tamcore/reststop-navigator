// Package overpass builds and runs Overpass QL queries against the public
// Overpass API. It exposes per-country query templates plus an HTTP client
// with backoff and endpoint failover.
package overpass

import (
	"fmt"
	"strings"
)

// CountryISO is an ISO 3166-1 alpha-2 country code. Always uppercase.
type CountryISO string

// MVP-supported countries.
const (
	DE CountryISO = "DE"
	AT CountryISO = "AT"
	SK CountryISO = "SK"
	CZ CountryISO = "CZ"
)

// SupportedCountries returns the ISO codes the MVP covers, in stable order.
func SupportedCountries() []CountryISO {
	return []CountryISO{DE, AT, SK, CZ}
}

// IsSupported reports whether the given country is in the MVP set.
// Comparison is case-sensitive — ISO 3166-1 alpha-2 codes are uppercase.
func IsSupported(c CountryISO) bool {
	switch c {
	case DE, AT, SK, CZ:
		return true
	}
	return false
}

// BBox is a geographic bounding box: south, west, north, east in WGS84 degrees.
type BBox struct {
	South, West, North, East float64
}

// CountryBBoxes returns the 4 quadrant sub-bboxes the hydrator should fetch
// for c. Splitting per-country into smaller bboxes keeps each Overpass query
// well under the 180s server-side timeout.
func CountryBBoxes(c CountryISO) ([]BBox, error) {
	if !IsSupported(c) {
		return nil, fmt.Errorf("overpass: unsupported country %q", string(c))
	}
	full := countryFullBBox(c)
	midLat := (full.South + full.North) / 2
	midLon := (full.West + full.East) / 2
	return []BBox{
		{South: full.South, West: full.West, North: midLat, East: midLon},
		{South: full.South, West: midLon, North: midLat, East: full.East},
		{South: midLat, West: full.West, North: full.North, East: midLon},
		{South: midLat, West: midLon, North: full.North, East: full.East},
	}, nil
}

func countryFullBBox(c CountryISO) BBox {
	switch c {
	case DE:
		return BBox{South: 47.27, West: 5.86, North: 55.10, East: 15.04}
	case AT:
		return BBox{South: 46.37, West: 9.53, North: 49.02, East: 17.16}
	case SK:
		return BBox{South: 47.73, West: 16.84, North: 49.61, East: 22.57}
	case CZ:
		return BBox{South: 48.55, West: 12.09, North: 51.06, East: 18.86}
	}
	return BBox{}
}

// BBoxQuery returns the Overpass QL string that retrieves motorway ways,
// rest-stop nodes/ways, and the fuel + EV-charging amenity nodes within the
// given bbox. Drops trunk and the heavier food/toilets amenities to keep
// payloads small enough for Overpass to deliver inside its 180s timeout.
func BBoxQuery(bb BBox) string {
	var sb strings.Builder
	fmt.Fprintln(&sb, "[out:json][timeout:120];")
	fmt.Fprintln(&sb, "(")
	fmt.Fprintf(&sb, "  way[\"highway\"=\"motorway\"](%g,%g,%g,%g);\n", bb.South, bb.West, bb.North, bb.East)
	fmt.Fprintf(&sb, "  node[\"highway\"~\"^(services|rest_area)$\"](%g,%g,%g,%g);\n", bb.South, bb.West, bb.North, bb.East)
	fmt.Fprintf(&sb, "  way[\"highway\"~\"^(services|rest_area)$\"](%g,%g,%g,%g);\n", bb.South, bb.West, bb.North, bb.East)
	fmt.Fprintf(&sb, "  node[\"amenity\"=\"fuel\"](%g,%g,%g,%g);\n", bb.South, bb.West, bb.North, bb.East)
	fmt.Fprintf(&sb, "  node[\"amenity\"=\"charging_station\"](%g,%g,%g,%g);\n", bb.South, bb.West, bb.North, bb.East)
	fmt.Fprintln(&sb, ");")
	fmt.Fprintln(&sb, "out geom;")
	return sb.String()
}

// CountryQuery returns the Overpass QL string for the entire country bbox in
// a single query. Production hydrator should prefer CountryBBoxes +
// BBoxQuery; kept here for tests + admin tools.
func CountryQuery(c CountryISO) (string, error) {
	if !IsSupported(c) {
		return "", fmt.Errorf("overpass: unsupported country %q", string(c))
	}
	return BBoxQuery(countryFullBBox(c)), nil
}
