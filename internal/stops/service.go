// Package stops orchestrates the upcoming-rest-stops query: country resolve,
// road match, ahead filter, amenity filter, ranking.
package stops

import (
	"context"
	"fmt"
	"math"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// DatasetReader reads the cached dataset for a country.
type DatasetReader interface {
	ReadDataset(ctx context.Context, c overpass.CountryISO) (overpass.Dataset, error)
}

// Service answers upcoming-stops queries.
type Service struct {
	reader DatasetReader
}

// NewService builds a Service.
func NewService(r DatasetReader) *Service {
	return &Service{reader: r}
}

// Filters is the user-supplied amenity filter set. A true field means "must
// have"; false means "don't care".
type Filters struct {
	Fuel     bool
	Charging bool
	Food     bool
	Toilets  bool
	Open24h  bool
	Dog      bool
}

// UpcomingRequest is what handlers translate query params into.
type UpcomingRequest struct {
	Pos     geo.LatLng
	Heading float64
	Speed   float64 // km/h
	Filters Filters
	Limit   int
}

// Road describes the matched motorway/trunk way.
type Road struct {
	Ref       string `json:"ref,omitempty"`
	Name      string `json:"name,omitempty"`
	Direction string `json:"direction,omitempty"`
}

// StopInfo is the public response shape per stop.
type StopInfo struct {
	ID           string                `json:"id"`
	Name         string                `json:"name,omitempty"`
	Kind         string                `json:"kind"`
	Lat          float64               `json:"lat"`
	Lon          float64               `json:"lon"`
	DistanceM    int                   `json:"distance_m"`
	ETASeconds   int                   `json:"eta_seconds"`
	Amenities    overpass.AmenityFlags `json:"amenities"`
	OpeningHours string                `json:"opening_hours,omitempty"`
	Operator     string                `json:"operator,omitempty"`
}

// UpcomingResponse is the public response shape.
type UpcomingResponse struct {
	Country string     `json:"country,omitempty"`
	Road    *Road      `json:"road,omitempty"`
	Stops   []StopInfo `json:"stops"`
	Version string     `json:"version,omitempty"`
	Reason  string     `json:"reason,omitempty"`
}

const (
	minSpeedKMH  = 60.0
	defaultLimit = 10
	maxLimit     = 25
)

// countryBBoxes are coarse bounding boxes for fast country resolution.
// Overlapping countries are disambiguated by SupportedCountries iteration
// order — adequate for MVP.
var countryBBoxes = map[overpass.CountryISO]struct{ MinLat, MaxLat, MinLon, MaxLon float64 }{
	overpass.DE: {47.27, 55.10, 5.86, 15.04},
	overpass.AT: {46.37, 49.02, 9.53, 17.16},
	overpass.SK: {47.73, 49.61, 16.84, 22.57},
	overpass.CZ: {48.55, 51.06, 12.09, 18.86},
}

// Upcoming returns the ranked list of rest stops the driver will encounter
// next, given their current position, heading, speed, and amenity filters.
func (s *Service) Upcoming(ctx context.Context, req UpcomingRequest) (UpcomingResponse, error) {
	country := resolveCountry(req.Pos)
	if country == "" {
		return UpcomingResponse{Reason: "outside-supported-area", Stops: []StopInfo{}}, nil
	}

	ds, err := s.reader.ReadDataset(ctx, country)
	if err != nil {
		return UpcomingResponse{}, err
	}

	match, ok := geo.NearestForwardWay(req.Pos, req.Heading, ds.Ways, geo.MatchOpts{})
	if !ok {
		return UpcomingResponse{
			Country: string(country),
			Version: ds.Version,
			Reason:  "off-highway-or-wrong-direction",
			Stops:   []StopInfo{},
		}, nil
	}

	soNw := make([]geo.StopOnWay, len(ds.Stops))
	for i, st := range ds.Stops {
		soNw[i] = geo.StopOnWay{
			StopID: stopID(st),
			Pos:    st.Pos,
		}
	}
	ahead := geo.FilterAhead(match, req.Pos, soNw)

	stopByID := make(map[string]overpass.Stop, len(ds.Stops))
	for _, st := range ds.Stops {
		stopByID[stopID(st)] = st
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}

	speed := req.Speed
	if speed < minSpeedKMH {
		speed = minSpeedKMH
	}
	speedMS := speed * 1000.0 / 3600.0

	out := UpcomingResponse{
		Country: string(country),
		Version: ds.Version,
		Road: &Road{
			Ref:       match.Way.Ref,
			Name:      match.Way.Name,
			Direction: directionFromMatch(match),
		},
		Stops: make([]StopInfo, 0, len(ahead)),
	}

	for _, a := range ahead {
		st, ok := stopByID[a.StopID]
		if !ok {
			continue
		}
		if !filtersMatch(req.Filters, st.Amenities) {
			continue
		}
		out.Stops = append(out.Stops, StopInfo{
			ID:           a.StopID,
			Name:         st.Name,
			Kind:         st.Kind,
			Lat:          st.Pos.Lat,
			Lon:          st.Pos.Lon,
			DistanceM:    int(math.Round(a.Distance)),
			ETASeconds:   int(math.Round(a.Distance / speedMS)),
			Amenities:    st.Amenities,
			OpeningHours: st.Tags["opening_hours"],
			Operator:     st.Tags["operator"],
		})
		if len(out.Stops) >= limit {
			break
		}
	}
	return out, nil
}

func resolveCountry(p geo.LatLng) overpass.CountryISO {
	for _, c := range overpass.SupportedCountries() {
		bb := countryBBoxes[c]
		if p.Lat >= bb.MinLat && p.Lat <= bb.MaxLat && p.Lon >= bb.MinLon && p.Lon <= bb.MaxLon {
			return c
		}
	}
	return ""
}

func stopID(s overpass.Stop) string {
	t := s.OSMType
	if t == "" {
		t = "stop"
	}
	return fmt.Sprintf("%s/%d", t, s.OSMID)
}

func filtersMatch(f Filters, a overpass.AmenityFlags) bool {
	if f.Fuel && !a.Fuel {
		return false
	}
	if f.Charging && !a.Charging {
		return false
	}
	if f.Food && !a.Food {
		return false
	}
	if f.Toilets && !a.Toilets {
		return false
	}
	if f.Open24h && !a.Open24h {
		return false
	}
	if f.Dog && !a.DogFriendly {
		return false
	}
	return true
}

func directionFromMatch(m geo.Match) string {
	if m.Way == nil || len(m.Way.Coords) < m.SegmentIndex+2 {
		return ""
	}
	a, b := m.Way.Coords[m.SegmentIndex], m.Way.Coords[m.SegmentIndex+1]
	if !m.Forward {
		a, b = b, a
	}
	return bearingToCardinal(geo.Bearing(a, b))
}

// bearingToCardinal converts a 0-360 bearing into one of 8 compass labels.
func bearingToCardinal(b float64) string {
	dirs := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	idx := int(math.Round(b/45)) % 8
	return dirs[idx]
}
