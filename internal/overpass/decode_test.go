package overpass_test

import (
	"testing"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

const fixtureJSON = `{
  "version": 0.6,
  "elements": [
    {
      "type": "way", "id": 1001,
      "tags": {"highway":"motorway","oneway":"yes","ref":"A8","name":"Stuttgart - Munchen"},
      "geometry": [
        {"lat": 48.0, "lon": 11.0},
        {"lat": 48.0, "lon": 11.005},
        {"lat": 48.0, "lon": 11.010}
      ]
    },
    {
      "type": "way", "id": 1002,
      "tags": {"highway":"trunk","oneway":"yes","ref":"B12"},
      "geometry": [
        {"lat": 48.5, "lon": 11.0},
        {"lat": 48.501, "lon": 11.0}
      ]
    },
    {
      "type": "way", "id": 1003,
      "tags": {"highway":"residential"},
      "geometry": [
        {"lat": 48.0, "lon": 11.5},
        {"lat": 48.001, "lon": 11.5}
      ]
    },
    {
      "type": "node", "id": 2001,
      "lat": 48.0, "lon": 11.007,
      "tags": {"highway":"services","name":"Rasthof Aichen Sud","operator":"Tank & Rast"}
    },
    {
      "type": "way", "id": 2002,
      "tags": {"highway":"rest_area","name":"Holledau"},
      "center": {"lat": 48.5, "lon": 11.5},
      "geometry": [
        {"lat": 48.499, "lon": 11.499},
        {"lat": 48.501, "lon": 11.501}
      ]
    },
    {
      "type": "node", "id": 3001,
      "lat": 48.0, "lon": 11.0075,
      "tags": {"amenity":"fuel","brand":"Aral"}
    },
    {
      "type": "node", "id": 3002,
      "lat": 48.0, "lon": 11.0072,
      "tags": {"amenity":"charging_station","capacity":"4"}
    },
    {
      "type": "node", "id": 3003,
      "lat": 48.0, "lon": 11.0073,
      "tags": {"amenity":"restaurant","name":"Burger Bar"}
    }
  ]
}`

func TestDecode_Ways(t *testing.T) {
	t.Parallel()

	ds, err := overpass.Decode([]byte(fixtureJSON))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if len(ds.Ways) != 2 {
		t.Fatalf("ways = %d, want 2 (motorway + trunk; residential excluded)", len(ds.Ways))
	}

	byID := map[string]int{}
	for i, w := range ds.Ways {
		byID[w.ID] = i
	}
	if _, ok := byID["way/1001"]; !ok {
		t.Error("way/1001 (motorway) missing")
	}
	if _, ok := byID["way/1002"]; !ok {
		t.Error("way/1002 (trunk) missing")
	}

	a8 := ds.Ways[byID["way/1001"]]
	if !a8.Oneway || a8.Ref != "A8" || a8.Name != "Stuttgart - Munchen" {
		t.Errorf("A8 metadata: oneway=%v ref=%q name=%q", a8.Oneway, a8.Ref, a8.Name)
	}
	if len(a8.Coords) != 3 {
		t.Errorf("A8 coords len = %d, want 3", len(a8.Coords))
	}
}

func TestDecode_Stops(t *testing.T) {
	t.Parallel()

	ds, err := overpass.Decode([]byte(fixtureJSON))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if len(ds.Stops) != 2 {
		t.Fatalf("stops = %d, want 2 (services node + rest_area way)", len(ds.Stops))
	}

	byKind := map[string]overpass.Stop{}
	for _, s := range ds.Stops {
		byKind[s.Kind] = s
	}

	svc, ok := byKind["services"]
	if !ok {
		t.Fatal("services stop missing")
	}
	if svc.Name != "Rasthof Aichen Sud" {
		t.Errorf("services name = %q", svc.Name)
	}
	if svc.OSMID != 2001 {
		t.Errorf("services OSMID = %d, want 2001", svc.OSMID)
	}

	rest, ok := byKind["rest_area"]
	if !ok {
		t.Fatal("rest_area stop missing")
	}
	if rest.Pos.Lat != 48.5 || rest.Pos.Lon != 11.5 {
		t.Errorf("rest_area centroid = %+v, want {48.5, 11.5}", rest.Pos)
	}
}

func TestDecode_Amenities(t *testing.T) {
	t.Parallel()

	ds, err := overpass.Decode([]byte(fixtureJSON))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	wantKinds := map[string]bool{"fuel": false, "charging_station": false, "restaurant": false}
	for _, a := range ds.Amenities {
		wantKinds[a.Kind] = true
	}
	for k, found := range wantKinds {
		if !found {
			t.Errorf("amenity kind %q missing from decoded set", k)
		}
	}
}

func TestDecode_RejectsBadJSON(t *testing.T) {
	t.Parallel()
	if _, err := overpass.Decode([]byte("{not json")); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestDecode_DegenerateWaySkipped(t *testing.T) {
	t.Parallel()
	const j = `{"elements":[{"type":"way","id":99,"tags":{"highway":"motorway"},"geometry":[{"lat":48,"lon":11}]}]}`
	ds, err := overpass.Decode([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Ways) != 0 {
		t.Errorf("single-coord way should be skipped, got %d ways", len(ds.Ways))
	}
}

func TestDecode_WayWithoutGeometryAndCenterSkipped(t *testing.T) {
	t.Parallel()
	const j = `{"elements":[{"type":"way","id":99,"tags":{"highway":"services"}}]}`
	ds, err := overpass.Decode([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Stops) != 0 {
		t.Errorf("services way without geometry should be skipped, got %d stops", len(ds.Stops))
	}
}
