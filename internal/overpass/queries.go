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

// CountryQuery returns the Overpass QL string that retrieves all
// motorway/trunk ways, rest-stop nodes/ways, and amenity nodes within the
// given country. Returns an error for unsupported country codes.
func CountryQuery(c CountryISO) (string, error) {
	if !IsSupported(c) {
		return "", fmt.Errorf("overpass: unsupported country %q", string(c))
	}
	var sb strings.Builder
	fmt.Fprintln(&sb, "[out:json][timeout:180];")
	fmt.Fprintf(&sb, "area[\"ISO3166-1\"=%q]->.country;\n", string(c))
	fmt.Fprintln(&sb, "(")
	fmt.Fprintln(&sb, `  way(area.country)["highway"~"^(motorway|trunk)$"];`)
	fmt.Fprintln(&sb, `  node(area.country)["highway"~"^(services|rest_area)$"];`)
	fmt.Fprintln(&sb, `  way(area.country)["highway"~"^(services|rest_area)$"];`)
	fmt.Fprintln(&sb, `  node(area.country)["amenity"~"^(fuel|charging_station|toilets|restaurant|cafe|fast_food)$"];`)
	fmt.Fprintln(&sb, ");")
	fmt.Fprintln(&sb, "out geom;")
	return sb.String(), nil
}
