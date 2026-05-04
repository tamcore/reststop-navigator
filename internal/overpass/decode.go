package overpass

import (
	"encoding/json"
	"fmt"

	"github.com/tamcore/reststop-navigator/internal/geo"
)

// AmenityFlags is the per-stop boolean availability set, computed at hydrate
// time from co-located amenity nodes and the stop's own tags.
type AmenityFlags struct {
	Fuel        bool `json:"fuel"`
	Charging    bool `json:"charging"`
	Food        bool `json:"food"`
	Toilets     bool `json:"toilets"`
	Open24h     bool `json:"open24h"`
	DogFriendly bool `json:"dog"`
}

// Stop is a parsed OSM rest-stop element (highway=services|rest_area). The
// Amenities field is populated by EnrichDataset, not Decode.
type Stop struct {
	OSMType   string            `json:"osm_type"` // node|way
	OSMID     int64             `json:"osm_id"`
	Kind      string            `json:"kind"` // services|rest_area
	Pos       geo.LatLng        `json:"pos"`
	Name      string            `json:"name,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Amenities AmenityFlags      `json:"amenities"`
}

// AmenityNode is a parsed OSM amenity element (fuel, charging_station, …).
// Used by the hydrator to enrich Stop amenity flags via spatial join.
type AmenityNode struct {
	OSMID int64             `json:"osm_id"`
	Kind  string            `json:"kind"`
	Pos   geo.LatLng        `json:"pos"`
	Tags  map[string]string `json:"tags,omitempty"`
}

// Dataset holds the parsed Overpass response for one country.
type Dataset struct {
	Country   CountryISO    `json:"country"`
	Version   string        `json:"version"`
	Ways      []geo.Way     `json:"ways"`
	Stops     []Stop        `json:"stops"`
	Amenities []AmenityNode `json:"amenities,omitempty"`
}

// Decode parses an Overpass JSON response body into a Dataset. Country and
// Version are left zero-valued — the caller (hydrator) sets them.
//
// Returns an error when the body carries an Overpass-level "remark" (e.g.
// "runtime error: query timed out") rather than silently producing an empty
// dataset.
func Decode(raw []byte) (Dataset, error) {
	var resp overpassResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return Dataset{}, fmt.Errorf("overpass: decode json: %w", err)
	}
	if resp.Remark != "" {
		return Dataset{}, fmt.Errorf("overpass: server reported: %s", resp.Remark)
	}

	var ds Dataset
	for _, e := range resp.Elements {
		switch e.Type {
		case "way":
			highway := e.Tags["highway"]
			switch highway {
			case "motorway", "trunk":
				if len(e.Geometry) < 2 {
					continue
				}
				coords := make([]geo.LatLng, len(e.Geometry))
				for i, g := range e.Geometry {
					coords[i] = geo.LatLng{Lat: g.Lat, Lon: g.Lon}
				}
				ds.Ways = append(ds.Ways, geo.Way{
					ID:     fmt.Sprintf("way/%d", e.ID),
					Coords: coords,
					Oneway: e.Tags["oneway"] == "yes",
					Ref:    e.Tags["ref"],
					Name:   e.Tags["name"],
				})
			case "services", "rest_area":
				pos, ok := centroidOf(e)
				if !ok {
					continue
				}
				ds.Stops = append(ds.Stops, Stop{
					OSMType: "way",
					OSMID:   e.ID,
					Kind:    highway,
					Pos:     pos,
					Name:    e.Tags["name"],
					Tags:    e.Tags,
				})
			}
		case "node":
			if h := e.Tags["highway"]; h == "services" || h == "rest_area" {
				ds.Stops = append(ds.Stops, Stop{
					OSMType: "node",
					OSMID:   e.ID,
					Kind:    h,
					Pos:     geo.LatLng{Lat: e.Lat, Lon: e.Lon},
					Name:    e.Tags["name"],
					Tags:    e.Tags,
				})
				continue
			}
			if a := e.Tags["amenity"]; a != "" {
				ds.Amenities = append(ds.Amenities, AmenityNode{
					OSMID: e.ID,
					Kind:  a,
					Pos:   geo.LatLng{Lat: e.Lat, Lon: e.Lon},
					Tags:  e.Tags,
				})
			}
		}
	}
	return ds, nil
}

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
	Remark   string            `json:"remark,omitempty"`
}

type overpassElement struct {
	Type     string            `json:"type"`
	ID       int64             `json:"id"`
	Lat      float64           `json:"lat,omitempty"`
	Lon      float64           `json:"lon,omitempty"`
	Center   *latLonPoint      `json:"center,omitempty"`
	Geometry []latLonPoint     `json:"geometry,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

type latLonPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func centroidOf(e overpassElement) (geo.LatLng, bool) {
	if e.Center != nil {
		return geo.LatLng{Lat: e.Center.Lat, Lon: e.Center.Lon}, true
	}
	if len(e.Geometry) > 0 {
		var sumLat, sumLon float64
		for _, g := range e.Geometry {
			sumLat += g.Lat
			sumLon += g.Lon
		}
		n := float64(len(e.Geometry))
		return geo.LatLng{Lat: sumLat / n, Lon: sumLon / n}, true
	}
	return geo.LatLng{}, false
}
