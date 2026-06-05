package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// GameMap is a guide-owned map composed of an ordered list of rounds.
type GameMap struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	GuideID     string    `json:"guide_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Rounds      []Round   `json:"rounds,omitempty"`
}

// Round is a single navigation challenge with a start point, end point,
// permitted corridor, and optional no-go zones.
type Round struct {
	ID          string        `json:"id"`
	MapID       string        `json:"map_id"`
	RoundNumber int           `json:"round_number"`
	Name        string        `json:"name"`
	StartPoint  orb.Point     `json:"start_point"`
	EndPoint    orb.Point     `json:"end_point"`
	Corridor    orb.Polygon   `json:"corridor"`
	NoGoZones   []orb.Polygon `json:"no_go_zones,omitempty"`
}

// ValidateRound checks that a round has valid, non-zero start/end points,
// that start and end are distinct, and that the corridor has at least 3 vertices.
func ValidateRound(r *Round) error {
	zeroPoint := orb.Point{0, 0}
	if r.StartPoint.Equal(zeroPoint) {
		return errors.New("start point is required")
	}
	if r.EndPoint.Equal(zeroPoint) {
		return errors.New("end point is required")
	}
	if r.StartPoint.Equal(r.EndPoint) {
		return errors.New("start and end points must be different locations")
	}
	if len(r.Corridor) == 0 || len(r.Corridor[0]) < 4 {
		// GeoJSON polygons require at least 4 coordinates (3 vertices + closing point).
		return errors.New("corridor polygon is required and must have at least 3 vertices")
	}
	return nil
}

// RoundFromJSON parses GeoJSON strings into a Round's geometry fields.
func RoundFromJSON(startPointJSON, endPointJSON, corridorJSON string) (orb.Point, orb.Point, orb.Polygon, error) {
	startGeom, err := geojson.UnmarshalGeometry([]byte(startPointJSON))
	if err != nil {
		return orb.Point{}, orb.Point{}, nil, fmt.Errorf("parse start point: %w", err)
	}
	endGeom, err := geojson.UnmarshalGeometry([]byte(endPointJSON))
	if err != nil {
		return orb.Point{}, orb.Point{}, nil, fmt.Errorf("parse end point: %w", err)
	}
	corrGeom, err := geojson.UnmarshalGeometry([]byte(corridorJSON))
	if err != nil {
		return orb.Point{}, orb.Point{}, nil, fmt.Errorf("parse corridor: %w", err)
	}

	start, ok := startGeom.Geometry().(orb.Point)
	if !ok {
		return orb.Point{}, orb.Point{}, nil, errors.New("start point: expected Point geometry")
	}
	end, ok := endGeom.Geometry().(orb.Point)
	if !ok {
		return orb.Point{}, orb.Point{}, nil, errors.New("end point: expected Point geometry")
	}
	corr, ok := corrGeom.Geometry().(orb.Polygon)
	if !ok {
		return orb.Point{}, orb.Point{}, nil, errors.New("corridor: expected Polygon geometry")
	}
	return start, end, corr, nil
}

// NoGoZonesToJSON encodes a slice of polygons as a JSON array of GeoJSON geometries.
func NoGoZonesToJSON(zones []orb.Polygon) string {
	if len(zones) == 0 {
		return "[]"
	}
	geoms := make([]json.RawMessage, len(zones))
	for i, z := range zones {
		b, _ := geojson.NewGeometry(z).MarshalJSON()
		geoms[i] = b
	}
	b, _ := json.Marshal(geoms)
	return string(b)
}

// NoGoZonesFromJSON decodes a JSON array of GeoJSON polygon geometries.
func NoGoZonesFromJSON(s string) ([]orb.Polygon, error) {
	if s == "" || s == "[]" {
		return nil, nil
	}
	var raws []json.RawMessage
	if err := json.Unmarshal([]byte(s), &raws); err != nil {
		return nil, fmt.Errorf("decode no-go zones: %w", err)
	}
	zones := make([]orb.Polygon, 0, len(raws))
	for _, raw := range raws {
		geom, err := geojson.UnmarshalGeometry(raw)
		if err != nil {
			return nil, fmt.Errorf("parse no-go zone geometry: %w", err)
		}
		poly, ok := geom.Geometry().(orb.Polygon)
		if !ok {
			return nil, errors.New("expected Polygon geometry in no-go zones")
		}
		zones = append(zones, poly)
	}
	return zones, nil
}
