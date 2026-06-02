// Package stops orchestrates the upcoming-rest-stops query: country resolve,
// road match, ahead filter, amenity filter, ranking.
package stops

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// TileSource fetches the merged dataset for the tiles around a position.
// Implemented by *cache.TileCache in production; faked in tests.
type TileSource interface {
	GetMerged(ctx context.Context, pos geo.LatLng) (overpass.Dataset, error)
}

// Service answers upcoming-stops queries.
type Service struct {
	tiles TileSource
}

// NewService builds a Service.
func NewService(tiles TileSource) *Service {
	return &Service{tiles: tiles}
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
	Pos      geo.LatLng
	Heading  float64
	Speed    float64 // km/h
	Accuracy float64 // GPS accuracy in meters (0 = not provided)
	Filters  Filters
	Limit    int
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

// DeepLinks are precomputed nav-app handoff URLs for one stop.
type DeepLinks struct {
	Google string `json:"google"`
	Apple  string `json:"apple"`
	Waze   string `json:"waze"`
}

// DetailResponse is the per-stop detail shape used by GET /api/stops/detail.
type DetailResponse struct {
	Country   string            `json:"country"`
	Stop      StopInfo          `json:"stop"`
	DeepLinks DeepLinks         `json:"deep_links"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// ErrStopNotFound is returned by Get when no supported country contains a stop
// with the given id.
var ErrStopNotFound = fmt.Errorf("stops: not found")

const (
	minSpeedKMH         = 60.0
	defaultLimit        = 10
	maxLimit            = 25
	maxMatchDistanceCap = 250.0 // upper bound for accuracy-widened match radius
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

	ds, err := s.tiles.GetMerged(ctx, req.Pos)
	if err != nil {
		return UpcomingResponse{}, err
	}

	match, ok := geo.NearestForwardWay(req.Pos, req.Heading, ds.Ways, geo.MatchOpts{
		MaxDistanceMeters: matchRadiusFromAccuracy(req.Accuracy),
	})
	if !ok {
		return UpcomingResponse{
			Country: string(country),
			Version: ds.Version,
			Reason:  "off-highway-or-wrong-direction",
			Stops:   []StopInfo{},
		}, nil
	}

	// Filter to the matched carriageway: same highway ref, same direction (≤90°).
	matchedBearing := matchedSegmentBearing(match)
	var droppedHighway, droppedDirection, droppedUnsnapped int
	carriageStops := make([]overpass.Stop, 0, len(ds.Stops))
	for _, st := range ds.Stops {
		if st.HighwayRef == "" {
			droppedUnsnapped++
			continue
		}
		if st.HighwayRef != match.Way.Ref {
			droppedHighway++
			continue
		}
		if geo.AngleDiff(st.HighwayBearing, matchedBearing) > 90 {
			droppedDirection++
			continue
		}
		carriageStops = append(carriageStops, st)
	}

	soNw := make([]geo.StopOnWay, len(carriageStops))
	for i, st := range carriageStops {
		soNw[i] = geo.StopOnWay{
			StopID: stopID(st),
			Pos:    st.Pos,
		}
	}
	ahead := geo.FilterAhead(match, req.Pos, soNw)

	stopByID := make(map[string]overpass.Stop, len(carriageStops))
	for _, st := range carriageStops {
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
	slog.InfoContext(ctx, "stops.upcoming",
		"road_ref", out.Road.Ref,
		"road_dir", out.Road.Direction,
		"returned", len(out.Stops),
		"dropped_wrong_highway", droppedHighway,
		"dropped_wrong_direction", droppedDirection,
		"dropped_unsnapped", droppedUnsnapped,
		"stop_ids", stopIDsOf(out.Stops),
	)
	return out, nil
}

// Get returns the detail view for one stop. id is the public id form
// "{osm_type}/{osm_id}" (e.g. "node/123"). pos is the approximate location
// to look in (typically the lat/lon the client already has from the
// upcoming-stops list); the lookup scans the tiles around that position.
func (s *Service) Get(ctx context.Context, id string, pos geo.LatLng) (DetailResponse, error) {
	country := resolveCountry(pos)
	if country == "" {
		return DetailResponse{}, ErrStopNotFound
	}
	ds, err := s.tiles.GetMerged(ctx, pos)
	if err != nil {
		return DetailResponse{}, err
	}
	for _, st := range ds.Stops {
		if stopID(st) != id {
			continue
		}
		return DetailResponse{
			Country: string(country),
			Stop: StopInfo{
				ID:           id,
				Name:         st.Name,
				Kind:         st.Kind,
				Lat:          st.Pos.Lat,
				Lon:          st.Pos.Lon,
				Amenities:    st.Amenities,
				OpeningHours: st.Tags["opening_hours"],
				Operator:     st.Tags["operator"],
			},
			DeepLinks: deepLinks(st.Pos),
			Tags:      st.Tags,
		}, nil
	}
	return DetailResponse{}, ErrStopNotFound
}

func deepLinks(p geo.LatLng) DeepLinks {
	return DeepLinks{
		Google: fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%g,%g&travelmode=driving", p.Lat, p.Lon),
		Apple:  fmt.Sprintf("https://maps.apple.com/?daddr=%g,%g&dirflg=d", p.Lat, p.Lon),
		Waze:   fmt.Sprintf("https://waze.com/ul?ll=%g,%g&navigate=yes", p.Lat, p.Lon),
	}
}

// matchRadiusFromAccuracy computes a dynamic MaxDistanceMeters based on GPS
// accuracy. Returns 0 (use default 80m) when accuracy is low/absent, otherwise
// max(80, accuracy) capped at 250m.
func matchRadiusFromAccuracy(accuracy float64) float64 {
	if accuracy <= 0 {
		return 0 // let NearestForwardWay use its default (80m)
	}
	r := accuracy
	if r < 80 {
		r = 80
	}
	if r > maxMatchDistanceCap {
		r = maxMatchDistanceCap
	}
	return r
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

// matchedSegmentBearing returns the bearing of the matched way segment in the
// driver's direction of travel (0-360°, 0=N). Returns 0 when the match is
// incomplete; callers use it with geo.AngleDiff so the zero fallback is safe.
func matchedSegmentBearing(m geo.Match) float64 {
	if m.Way == nil || len(m.Way.Coords) < m.SegmentIndex+2 {
		return 0
	}
	a, b := m.Way.Coords[m.SegmentIndex], m.Way.Coords[m.SegmentIndex+1]
	if !m.Forward {
		a, b = b, a
	}
	return geo.Bearing(a, b)
}

func directionFromMatch(m geo.Match) string {
	if m.Way == nil || len(m.Way.Coords) < m.SegmentIndex+2 {
		return ""
	}
	return bearingToCardinal(matchedSegmentBearing(m))
}

func stopIDsOf(stops []StopInfo) []string {
	ids := make([]string, len(stops))
	for i, s := range stops {
		ids[i] = s.ID
	}
	return ids
}

// bearingToCardinal converts a 0-360 bearing into one of 8 compass labels.
func bearingToCardinal(b float64) string {
	dirs := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	idx := int(math.Round(b/45)) % 8
	return dirs[idx]
}
