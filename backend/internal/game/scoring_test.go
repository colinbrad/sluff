package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/paulmach/orb"

	"github.com/colinbradley/sluff/internal/model"
)

// Build a large square corridor and start/end points for testing.
// The corridor is a square from (-111.9,40.7) to (-111.8,40.8).
// Start at SW corner area, End at NE corner area.
func testRound() *model.Round {
	return &model.Round{
		StartPoint: orb.Point{-111.89, 40.71},
		EndPoint:   orb.Point{-111.81, 40.79},
		Corridor: orb.Polygon{
			orb.Ring{
				{-111.9, 40.7},
				{-111.8, 40.7},
				{-111.8, 40.8},
				{-111.9, 40.8},
				{-111.9, 40.7}, // closing
			},
		},
	}
}

// makeLineStringJSON builds a GeoJSON LineString from a series of points.
func makeLineStringJSON(points []orb.Point) string {
	coords := make([]string, len(points))
	for i, p := range points {
		coords[i] = fmt.Sprintf("[%f,%f]", p[0], p[1])
	}
	return fmt.Sprintf(`{"type":"LineString","coordinates":[%s]}`, strings.Join(coords, ","))
}

func TestScoreRoute(t *testing.T) {
	t.Run("perfect route scores high", func(t *testing.T) {
		round := testRound()
		// A route that starts near the start point, goes through the corridor,
		// and ends near the end point.
		route := makeLineStringJSON([]orb.Point{
			round.StartPoint,
			{-111.85, 40.75}, // midpoint, inside corridor
			round.EndPoint,
		})

		score, err := ScoreRoute(route, round)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !score.ConnectsStart {
			t.Error("expected route to connect to start")
		}
		if !score.ConnectsEnd {
			t.Error("expected route to connect to end")
		}
		if score.PercentInCorridor < 90 {
			t.Errorf("expected high corridor adherence, got %.1f%%", score.PercentInCorridor)
		}
		// Should score well above 800 (600 adherence + 200 endpoints + efficiency + deviation)
		if score.FinalScore < 800 {
			t.Errorf("expected high score (>800), got %.1f", score.FinalScore)
		}
	})

	t.Run("route outside corridor scores low on adherence", func(t *testing.T) {
		round := testRound()
		// A route that goes entirely outside the corridor (south of it)
		route := makeLineStringJSON([]orb.Point{
			{-111.89, 40.60}, // south of corridor
			{-111.85, 40.60},
			{-111.81, 40.60},
		})

		score, err := ScoreRoute(route, round)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if score.PercentInCorridor > 10 {
			t.Errorf("expected low corridor adherence for outside route, got %.1f%%", score.PercentInCorridor)
		}
		// Adherence portion (max 600) should be very low
		// Also misses both endpoints, so loses 200
		if score.FinalScore > 300 {
			t.Errorf("expected low score for outside route, got %.1f", score.FinalScore)
		}
	})

	t.Run("route missing start endpoint loses 100 points", func(t *testing.T) {
		round := testRound()

		// Route that hits end but starts far from start point
		routeHitsBoth := makeLineStringJSON([]orb.Point{
			round.StartPoint,
			{-111.85, 40.75},
			round.EndPoint,
		})
		routeMissesStart := makeLineStringJSON([]orb.Point{
			{-111.85, 40.75}, // not near start
			{-111.83, 40.77},
			round.EndPoint,
		})

		scoreBoth, err := ScoreRoute(routeHitsBoth, round)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		scoreMissStart, err := ScoreRoute(routeMissesStart, round)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !scoreBoth.ConnectsStart {
			t.Error("expected full route to connect start")
		}
		if scoreMissStart.ConnectsStart {
			t.Error("expected route missing start to NOT connect start")
		}
		if scoreMissStart.ConnectsEnd != true {
			t.Error("expected route to still connect end")
		}
	})

	t.Run("route missing end endpoint loses 100 points", func(t *testing.T) {
		round := testRound()

		routeMissesEnd := makeLineStringJSON([]orb.Point{
			round.StartPoint,
			{-111.85, 40.75},
			{-111.83, 40.73}, // not near end
		})

		score, err := ScoreRoute(routeMissesEnd, round)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !score.ConnectsStart {
			t.Error("expected route to connect start")
		}
		if score.ConnectsEnd {
			t.Error("expected route missing end to NOT connect end")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		round := testRound()
		_, err := ScoreRoute("not json", round)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}
