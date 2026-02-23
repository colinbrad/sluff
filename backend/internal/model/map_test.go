package model

import (
	"testing"

	"github.com/paulmach/orb"
)

// Helper to build a valid round for testing.
func validRound() *Round {
	return &Round{
		StartPoint: orb.Point{-111.5, 40.6},
		EndPoint:   orb.Point{-111.6, 40.7},
		Corridor: orb.Polygon{
			orb.Ring{
				{-111.7, 40.5},
				{-111.4, 40.5},
				{-111.4, 40.8},
				{-111.7, 40.5}, // closing point
			},
		},
	}
}

func TestValidateRound(t *testing.T) {
	t.Run("valid round passes", func(t *testing.T) {
		r := validRound()
		if err := ValidateRound(r); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("fails for zero start point", func(t *testing.T) {
		r := validRound()
		r.StartPoint = orb.Point{0, 0}
		err := ValidateRound(r)
		if err == nil {
			t.Fatal("expected error for zero start point")
		}
		if err.Error() != "start point is required" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("fails for zero end point", func(t *testing.T) {
		r := validRound()
		r.EndPoint = orb.Point{0, 0}
		err := ValidateRound(r)
		if err == nil {
			t.Fatal("expected error for zero end point")
		}
		if err.Error() != "end point is required" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("fails when start equals end", func(t *testing.T) {
		r := validRound()
		r.EndPoint = r.StartPoint
		err := ValidateRound(r)
		if err == nil {
			t.Fatal("expected error when start == end")
		}
		if err.Error() != "start and end points must be different locations" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("fails for empty corridor", func(t *testing.T) {
		r := validRound()
		r.Corridor = orb.Polygon{}
		err := ValidateRound(r)
		if err == nil {
			t.Fatal("expected error for empty corridor")
		}
	})

	t.Run("fails for corridor with fewer than 4 points", func(t *testing.T) {
		r := validRound()
		r.Corridor = orb.Polygon{
			orb.Ring{
				{-111.7, 40.5},
				{-111.4, 40.5},
				{-111.4, 40.8},
				// missing closing point — only 3 coordinates
			},
		}
		err := ValidateRound(r)
		if err == nil {
			t.Fatal("expected error for corridor with < 4 points")
		}
	})
}

func TestRoundFromJSON(t *testing.T) {
	t.Run("parses valid GeoJSON", func(t *testing.T) {
		startJSON := `{"type":"Point","coordinates":[-111.5,40.6]}`
		endJSON := `{"type":"Point","coordinates":[-111.6,40.7]}`
		corridorJSON := `{"type":"Polygon","coordinates":[[[-111.7,40.5],[-111.4,40.5],[-111.4,40.8],[-111.7,40.5]]]}`

		start, end, corridor, err := RoundFromJSON(startJSON, endJSON, corridorJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if start[0] != -111.5 || start[1] != 40.6 {
			t.Errorf("start point mismatch: got %v", start)
		}
		if end[0] != -111.6 || end[1] != 40.7 {
			t.Errorf("end point mismatch: got %v", end)
		}
		if len(corridor) == 0 {
			t.Fatal("corridor should not be empty")
		}
		if len(corridor[0]) != 4 {
			t.Errorf("expected 4 coordinates in corridor ring, got %d", len(corridor[0]))
		}
	})

	t.Run("fails on invalid JSON", func(t *testing.T) {
		_, _, _, err := RoundFromJSON("not json", `{"type":"Point","coordinates":[0,0]}`, `{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]]]}`)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("fails on invalid end point JSON", func(t *testing.T) {
		_, _, _, err := RoundFromJSON(`{"type":"Point","coordinates":[0,0]}`, "bad", `{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]]]}`)
		if err == nil {
			t.Fatal("expected error for invalid end point JSON")
		}
	})

	t.Run("fails on invalid corridor JSON", func(t *testing.T) {
		_, _, _, err := RoundFromJSON(`{"type":"Point","coordinates":[0,0]}`, `{"type":"Point","coordinates":[1,1]}`, "bad")
		if err == nil {
			t.Fatal("expected error for invalid corridor JSON")
		}
	})
}
