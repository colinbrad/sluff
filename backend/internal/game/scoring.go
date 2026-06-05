// Package game implements scoring for a player-submitted route against a
// round's corridor, endpoints, and optional no-go zones.
package game

import (
	"errors"
	"fmt"
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/geojson"

	geoutil "github.com/colinbradley/sluff/internal/geo"
	"github.com/colinbradley/sluff/internal/model"
)

const (
	sampleCount       = 100
	endpointThreshold = 50.0 // meters
)

// ScoreRoute scores a player-drawn route against a round's corridor and no-go zones.
// pathJSON is a GeoJSON LineString geometry.
func ScoreRoute(pathJSON string, round *model.Round) (model.ScoreDetails, error) {
	geom, err := geojson.UnmarshalGeometry([]byte(pathJSON))
	if err != nil {
		return model.ScoreDetails{}, fmt.Errorf("decode path: %w", err)
	}

	route, ok := geom.Geometry().(orb.LineString)
	if !ok {
		return model.ScoreDetails{}, errors.New("path must be a LineString geometry")
	}

	return scoreLineAgainstCorridor(route, round.Corridor, round.StartPoint, round.EndPoint, round.NoGoZones), nil
}

func scoreLineAgainstCorridor(route orb.LineString, corridor orb.Polygon, start, end orb.Point, noGoZones []orb.Polygon) model.ScoreDetails {
	samples := geoutil.SampleLineString(route, sampleCount)

	insideCount := 0
	maxDeviation := 0.0
	noGoCount := 0

	for _, pt := range samples {
		if geoutil.PointInPolygon(pt, corridor) {
			insideCount++
		} else {
			dist := geoutil.MinDistanceToPolygonBoundary(pt, corridor)
			if dist > maxDeviation {
				maxDeviation = dist
			}
		}
		for _, zone := range noGoZones {
			if geoutil.PointInPolygon(pt, zone) {
				noGoCount++
				break
			}
		}
	}

	totalSamples := len(samples)
	percentInCorridor := 0.0
	if totalSamples > 0 {
		percentInCorridor = float64(insideCount) / float64(totalSamples) * 100
	}

	// Corridor adherence: 0-600 points
	adherenceScore := percentInCorridor / 100.0 * 600.0

	// Endpoint connection: 0-200 points (100 each)
	routeStart := route[0]
	routeEnd := route[len(route)-1]
	connectsStart := geo.DistanceHaversine(routeStart, start) <= endpointThreshold
	connectsEnd := geo.DistanceHaversine(routeEnd, end) <= endpointThreshold

	endpointScore := 0.0
	if connectsStart {
		endpointScore += 100
	}
	if connectsEnd {
		endpointScore += 100
	}

	// Route efficiency: 0-100 points
	routeLength := geoutil.LineStringLength(route)
	directDistance := geo.DistanceHaversine(start, end)
	efficiencyScore := 0.0
	if routeLength > 0 && directDistance > 0 {
		// Ratio of direct distance to route length. Perfect straight line = 1.0
		// We use a gentler curve: anything within 2x the direct distance gets decent points
		ratio := directDistance / routeLength
		efficiencyScore = math.Min(100, ratio*100)
	}

	// Low deviation penalty: 0-100 points
	deviationScore := 100.0
	if maxDeviation > 0 {
		// Lose 1 point per 10m of max deviation
		deviationScore = math.Max(0, 100-(maxDeviation/10))
	}

	// No-go zone penalty: up to 300 points deducted proportional to % of route in no-go zones
	noGoZonePenalty := 0.0
	if noGoCount > 0 && totalSamples > 0 {
		percentInNoGo := float64(noGoCount) / float64(totalSamples) * 100
		noGoZonePenalty = math.Round((percentInNoGo/100.0*300.0)*10) / 10
	}

	finalScore := math.Max(0, adherenceScore+endpointScore+efficiencyScore+deviationScore-noGoZonePenalty)

	return model.ScoreDetails{
		TotalPoints:       totalSamples,
		PointsInCorridor:  insideCount,
		PercentInCorridor: math.Round(percentInCorridor*10) / 10,
		RouteLengthKm:     math.Round(routeLength/100) / 10, // round to 0.1km
		MaxDeviationM:     math.Round(maxDeviation*10) / 10,
		ConnectsStart:     connectsStart,
		ConnectsEnd:       connectsEnd,
		PointsInNoGoZone:  noGoCount,
		NoGoZonePenalty:   noGoZonePenalty,
		FinalScore:        math.Round(finalScore*10) / 10,
	}
}
