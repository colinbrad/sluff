package model

import (
	"fmt"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

type GameMap struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Rounds      []Round   `json:"rounds,omitempty"`
}

type Round struct {
	ID          string      `json:"id"`
	MapID       string      `json:"map_id"`
	RoundNumber int         `json:"round_number"`
	Name        string      `json:"name"`
	StartPoint  orb.Point   `json:"start_point"`
	EndPoint    orb.Point   `json:"end_point"`
	Corridor    orb.Polygon `json:"corridor"`
}

// ValidateRound checks that a round has valid, non-zero start/end points,
// that start and end are distinct, and that the corridor has at least 3 vertices.
func ValidateRound(r *Round) error {
	zeroPoint := orb.Point{0, 0}
	if r.StartPoint.Equal(zeroPoint) {
		return fmt.Errorf("start point is required")
	}
	if r.EndPoint.Equal(zeroPoint) {
		return fmt.Errorf("end point is required")
	}
	if r.StartPoint.Equal(r.EndPoint) {
		return fmt.Errorf("start and end points must be different locations")
	}
	if len(r.Corridor) == 0 || len(r.Corridor[0]) < 4 {
		// GeoJSON polygons require at least 4 coordinates (3 vertices + closing point)
		return fmt.Errorf("corridor polygon is required and must have at least 3 vertices")
	}
	return nil
}

// RoundFromJSON parses GeoJSON strings into a Round's geometry fields.
func RoundFromJSON(startPointJSON, endPointJSON, corridorJSON string) (orb.Point, orb.Point, orb.Polygon, error) {
	startGeom, err := geojson.UnmarshalGeometry([]byte(startPointJSON))
	if err != nil {
		return orb.Point{}, orb.Point{}, nil, err
	}
	endGeom, err := geojson.UnmarshalGeometry([]byte(endPointJSON))
	if err != nil {
		return orb.Point{}, orb.Point{}, nil, err
	}
	corrGeom, err := geojson.UnmarshalGeometry([]byte(corridorJSON))
	if err != nil {
		return orb.Point{}, orb.Point{}, nil, err
	}

	return startGeom.Geometry().(orb.Point), endGeom.Geometry().(orb.Point), corrGeom.Geometry().(orb.Polygon), nil
}
