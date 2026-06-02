// Package gpx parses GPX track files and derives demo-replay data from them.
package gpx

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

// Track holds the flattened list of track points from a GPX file.
type Track struct {
	Points []Point
}

// Point is a single GPX track point.
type Point struct {
	Lat  float64
	Lon  float64
	Time time.Time
}

type gpxFile struct {
	XMLName xml.Name   `xml:"gpx"`
	Tracks  []xmlTrack `xml:"trk"`
}

type xmlTrack struct {
	Segments []xmlSegment `xml:"trkseg"`
}

type xmlSegment struct {
	Points []xmlPoint `xml:"trkpt"`
}

type xmlPoint struct {
	Lat  float64   `xml:"lat,attr"`
	Lon  float64   `xml:"lon,attr"`
	Time time.Time `xml:"time"`
}

// Parse reads a GPX document from r and returns all track points flattened
// across all tracks and segments in document order.
func Parse(r io.Reader) (Track, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Track{}, err
	}
	var doc gpxFile
	if err := xml.Unmarshal(data, &doc); err != nil {
		return Track{}, fmt.Errorf("parse gpx: %w", err)
	}
	var pts []Point
	for _, t := range doc.Tracks {
		for _, s := range t.Segments {
			for _, p := range s.Points {
				pts = append(pts, Point(p))
			}
		}
	}
	return Track{Points: pts}, nil
}
