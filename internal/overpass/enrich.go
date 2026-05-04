package overpass

import "github.com/tamcore/reststop-navigator/internal/geo"

// amenityJoinRadiusMeters is the spatial-join radius used to attach amenity
// nodes to nearby stops at hydrate time.
const amenityJoinRadiusMeters = 200.0

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
		}

		if s.Tags["opening_hours"] == "24/7" {
			s.Amenities.Open24h = true
		}
		if s.Tags["dog"] == "yes" {
			s.Amenities.DogFriendly = true
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
