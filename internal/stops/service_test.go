package stops_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
	"github.com/tamcore/reststop-navigator/internal/stops"
)

type fakeTiles struct {
	ds  overpass.Dataset
	err error
}

func (f fakeTiles) GetMerged(_ context.Context, _ geo.LatLng) (overpass.Dataset, error) {
	if f.err != nil {
		return overpass.Dataset{}, f.err
	}
	return f.ds, nil
}

func straightEastA8DE() overpass.Dataset {
	return overpass.Dataset{
		Country: overpass.DE,
		Version: "1714824000",
		Ways: []geo.Way{
			{
				ID:     "way/1001",
				Ref:    "A8",
				Name:   "Stuttgart - Munchen",
				Oneway: true,
				Coords: []geo.LatLng{
					{Lat: 48.000, Lon: 11.000},
					{Lat: 48.000, Lon: 11.005},
					{Lat: 48.000, Lon: 11.010},
					{Lat: 48.000, Lon: 11.020},
				},
			},
		},
		Stops: []overpass.Stop{
			{
				OSMType: "node", OSMID: 100, Kind: "services",
				Pos: geo.LatLng{Lat: 48.000, Lon: 11.0075}, Name: "Aichen",
				Amenities: overpass.AmenityFlags{Fuel: true, Food: true, Toilets: true, Open24h: true},
				Tags:      map[string]string{"opening_hours": "24/7", "operator": "Tank & Rast"},
			},
			{
				OSMType: "node", OSMID: 101, Kind: "rest_area",
				Pos: geo.LatLng{Lat: 48.000, Lon: 11.012}, Name: "Holledau",
				Amenities: overpass.AmenityFlags{Toilets: true},
			},
			{
				OSMType: "node", OSMID: 102, Kind: "services",
				Pos: geo.LatLng{Lat: 48.000, Lon: 11.018}, Name: "Far",
				Amenities: overpass.AmenityFlags{Fuel: true, Charging: true, Food: true, Toilets: true},
			},
			{
				OSMType: "node", OSMID: 103, Kind: "services",
				Pos: geo.LatLng{Lat: 48.000, Lon: 11.001}, Name: "Behind",
				Amenities: overpass.AmenityFlags{Fuel: true},
			},
		},
	}
}

func TestUpcoming_HappyPath(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.000, Lon: 11.003},
		Heading: 90,
		Speed:   120,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("Upcoming: %v", err)
	}
	if resp.Country != "DE" {
		t.Errorf("country = %q", resp.Country)
	}
	if resp.Road == nil || resp.Road.Ref != "A8" {
		t.Errorf("road = %+v", resp.Road)
	}
	if resp.Road.Direction != "E" {
		t.Errorf("direction = %q, want E", resp.Road.Direction)
	}
	names := []string{}
	for _, s := range resp.Stops {
		names = append(names, s.Name)
	}
	want := []string{"Aichen", "Holledau", "Far"}
	if !equalStrings(names, want) {
		t.Errorf("stop names = %v, want %v", names, want)
	}
	if resp.Stops[0].DistanceM <= 0 {
		t.Errorf("first stop DistanceM = %d, want positive", resp.Stops[0].DistanceM)
	}
	if resp.Stops[0].ETASeconds <= 0 {
		t.Errorf("first stop ETASeconds = %d, want positive", resp.Stops[0].ETASeconds)
	}
	if resp.Stops[0].ID != "node/100" {
		t.Errorf("stop id = %q, want node/100", resp.Stops[0].ID)
	}
	if resp.Stops[0].OpeningHours != "24/7" {
		t.Errorf("opening_hours = %q", resp.Stops[0].OpeningHours)
	}
}

func TestUpcoming_FuelFilter(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.000, Lon: 11.003},
		Heading: 90,
		Speed:   120,
		Filters: stops.Filters{Fuel: true, Charging: true},
		Limit:   10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Stops) != 1 || resp.Stops[0].Name != "Far" {
		t.Errorf("filtered stops = %+v, want only Far", resp.Stops)
	}
}

func TestUpcoming_OutsideSupportedArea(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 0, Lon: 0},
		Heading: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reason != "outside-supported-area" {
		t.Errorf("reason = %q", resp.Reason)
	}
}

func TestUpcoming_OffHighway(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 50.0, Lon: 11.0},
		Heading: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reason != "off-highway-or-wrong-direction" {
		t.Errorf("reason = %q", resp.Reason)
	}
	if resp.Country != "DE" {
		t.Errorf("country = %q, want DE", resp.Country)
	}
}

func TestUpcoming_RespectsLimit(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.000, Lon: 11.003},
		Heading: 90,
		Speed:   120,
		Limit:   2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Stops) != 2 {
		t.Errorf("stops = %d, want 2 (Limit=2)", len(resp.Stops))
	}
}

func TestUpcoming_DefaultLimitWhenZero(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.000, Lon: 11.003},
		Heading: 90,
		Speed:   120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Stops) == 0 {
		t.Error("expected non-empty stops with default limit")
	}
}

func TestUpcoming_PropagatesReaderError(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{err: errors.New("redis is down")})
	_, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.000, Lon: 11.003},
		Heading: 90,
	})
	if err == nil {
		t.Fatal("expected reader error to propagate")
	}
}

func TestUpcoming_LowSpeedClampsETA(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.000, Lon: 11.003},
		Heading: 90,
		Speed:   5,
		Limit:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Stops[0].ETASeconds > 60 {
		t.Errorf("ETA = %ds at 5 km/h, expected speed floor to clamp", resp.Stops[0].ETASeconds)
	}
}

func TestGet_HappyPath(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	got, err := svc.Get(context.Background(), "node/100", geo.LatLng{Lat: 48.000, Lon: 11.0075})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Country != "DE" {
		t.Errorf("country = %q", got.Country)
	}
	if got.Stop.Name != "Aichen" {
		t.Errorf("name = %q", got.Stop.Name)
	}
	if !strings.Contains(got.DeepLinks.Google, "destination=48,11.0075") {
		t.Errorf("google deep link = %q", got.DeepLinks.Google)
	}
	if !strings.Contains(got.DeepLinks.Apple, "daddr=48,11.0075") {
		t.Errorf("apple deep link = %q", got.DeepLinks.Apple)
	}
	if !strings.Contains(got.DeepLinks.Waze, "ll=48,11.0075") {
		t.Errorf("waze deep link = %q", got.DeepLinks.Waze)
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	svc := stops.NewService(fakeTiles{ds: straightEastA8DE()})
	_, err := svc.Get(context.Background(), "node/9999", geo.LatLng{Lat: 48.000, Lon: 11.005})
	if !errors.Is(err, stops.ErrStopNotFound) {
		t.Fatalf("err = %v, want ErrStopNotFound", err)
	}
}

func TestUpcoming_AccuracyWidensMatchRadius(t *testing.T) {
	t.Parallel()
	// Place user 150m from the highway — beyond default 80m, within accuracy-widened radius
	ds := straightEastA8DE()
	svc := stops.NewService(fakeTiles{ds: ds})

	// First: with zero accuracy (default 80m), user at 150m should NOT match
	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:     geo.LatLng{Lat: 48.00135, Lon: 11.003}, // ~150m north of the highway at 48.000
		Heading: 90,
		Speed:   120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reason != "off-highway-or-wrong-direction" {
		t.Fatalf("without accuracy, expected off-highway, got reason=%q road=%+v", resp.Reason, resp.Road)
	}

	// Now: with accuracy=200 (GPS is imprecise), match radius should widen to 200m
	resp, err = svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:      geo.LatLng{Lat: 48.00135, Lon: 11.003},
		Heading:  90,
		Speed:    120,
		Accuracy: 200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Road == nil || resp.Road.Ref != "A8" {
		t.Errorf("with accuracy=200, expected A8 match, got reason=%q road=%+v", resp.Reason, resp.Road)
	}
}

func TestUpcoming_AccuracyCappedAt250(t *testing.T) {
	t.Parallel()
	// Place user 300m from the highway — beyond the 250m cap
	ds := straightEastA8DE()
	svc := stops.NewService(fakeTiles{ds: ds})

	resp, err := svc.Upcoming(context.Background(), stops.UpcomingRequest{
		Pos:      geo.LatLng{Lat: 48.0027, Lon: 11.003}, // ~300m north
		Heading:  90,
		Speed:    120,
		Accuracy: 500, // way beyond cap
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reason != "off-highway-or-wrong-direction" {
		t.Errorf("with 300m distance and capped accuracy, expected off-highway, got reason=%q", resp.Reason)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
