package overpass

import (
	"strings"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

// hasFuelBrand reports whether the stop's tags name a fuel-station brand.
// Tank & Rast services often have brand=Aral / Shell / etc. on the way
// itself when the fuel-pump amenity isn't separately mapped.
func hasFuelBrand(tags map[string]string) bool {
	for _, k := range []string{"brand", "operator"} {
		v := strings.ToLower(tags[k])
		if v == "" {
			continue
		}
		for _, b := range []string{"aral", "shell", "esso", "total", "bp ", "agip", "jet", "omv", "tank & rast"} {
			if strings.Contains(v, b) {
				return true
			}
		}
	}
	return false
}

// amenityJoinRadiusMeters is the spatial-join radius used to attach amenity
// nodes to nearby stops at hydrate time. 350 m comfortably spans typical
// German Raststätten which can be 400+ m end-to-end with the fuel pump at
// one end and the restaurant at the other.
const amenityJoinRadiusMeters = 350.0

// EnrichDataset mutates ds.Stops in place to populate the AmenityFlags field
// based on (a) the stop's own tags and kind, and (b) amenity nodes within
// amenityJoinRadiusMeters of the stop.
//
// nil input is a safe no-op.
func EnrichDataset(ds *Dataset) {
	if ds == nil {
		return
	}
	for i := range ds.Stops {
		s := &ds.Stops[i]

		if s.Kind == "services" {
			s.Amenities.Food = true
			s.Amenities.Toilets = true
			// Tank & Rast services almost always include a fuel station;
			// many OSM editors don't tag it as a co-located node, only as a
			// brand on the parent way. Default services to fuel=true and let
			// the spatial join refine if needed.
			s.Amenities.Fuel = true
		}

		if s.Tags["opening_hours"] == "24/7" {
			s.Amenities.Open24h = true
		}
		if s.Tags["dog"] == "yes" {
			s.Amenities.DogFriendly = true
		}

		// Direct amenity tags on the stop way / node.
		if s.Tags["amenity"] == "fuel" || hasFuelBrand(s.Tags) {
			s.Amenities.Fuel = true
		}
		if s.Tags["amenity"] == "charging_station" ||
			s.Tags["charging_station"] == "yes" ||
			s.Tags["socket:type2"] != "" || s.Tags["socket:ccs"] != "" {
			s.Amenities.Charging = true
		}

		for _, a := range ds.Amenities {
			if geo.Distance(s.Pos, a.Pos) > amenityJoinRadiusMeters {
				continue
			}
			switch a.Kind {
			case "fuel":
				s.Amenities.Fuel = true
			case "charging_station":
				s.Amenities.Charging = true
			case "restaurant", "cafe", "fast_food":
				s.Amenities.Food = true
			case "toilets":
				s.Amenities.Toilets = true
			}
			if a.Tags["opening_hours"] == "24/7" {
				s.Amenities.Open24h = true
			}
			if a.Tags["dog"] == "yes" {
				s.Amenities.DogFriendly = true
			}
		}
	}
}
