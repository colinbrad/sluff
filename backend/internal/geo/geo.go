// Package geo provides geospatial helpers used by the scoring engine:
// sampling along a LineString, point-in-polygon, and distance utilities.
package geo

import (
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/planar"
)

// SampleLineString returns n evenly-spaced points along a LineString.
func SampleLineString(ls orb.LineString, n int) []orb.Point {
	if n <= 0 || len(ls) < 2 {
		return nil
	}

	totalLength := geo.LengthHaversine(ls)
	if totalLength == 0 {
		return []orb.Point{ls[0]}
	}

	step := totalLength / float64(n)
	points := make([]orb.Point, 0, n+1)

	for i := 0; i <= n; i++ {
		targetDist := float64(i) * step
		pt := interpolateAlongLine(ls, targetDist)
		points = append(points, pt)
	}
	return points
}

// interpolateAlongLine returns the point at distance d (meters) along the line.
func interpolateAlongLine(ls orb.LineString, d float64) orb.Point {
	if d <= 0 {
		return ls[0]
	}

	accumulated := 0.0
	for i := 0; i < len(ls)-1; i++ {
		segLen := geo.DistanceHaversine(ls[i], ls[i+1])
		if accumulated+segLen >= d {
			fraction := (d - accumulated) / segLen
			return orb.Point{
				ls[i][0] + fraction*(ls[i+1][0]-ls[i][0]),
				ls[i][1] + fraction*(ls[i+1][1]-ls[i][1]),
			}
		}
		accumulated += segLen
	}

	return ls[len(ls)-1]
}

// PointInPolygon tests whether a point is inside a polygon.
func PointInPolygon(p orb.Point, poly orb.Polygon) bool {
	return planar.PolygonContains(poly, p)
}

// MinDistanceToPolygonBoundary returns the minimum distance in meters from a point
// to the nearest edge of the polygon's exterior ring.
func MinDistanceToPolygonBoundary(pt orb.Point, poly orb.Polygon) float64 {
	if len(poly) == 0 || len(poly[0]) < 2 {
		return math.MaxFloat64
	}

	ring := poly[0] // exterior ring
	minDist := math.MaxFloat64

	for i := 0; i < len(ring)-1; i++ {
		d := distanceToSegment(pt, ring[i], ring[i+1])
		if d < minDist {
			minDist = d
		}
	}

	return minDist
}

// distanceToSegment returns the haversine distance from pt to the closest point on segment a-b.
func distanceToSegment(pt, a, b orb.Point) float64 {
	// Project pt onto line a-b in planar coords, then compute haversine distance
	dx := b[0] - a[0]
	dy := b[1] - a[1]
	if dx == 0 && dy == 0 {
		return geo.DistanceHaversine(pt, a)
	}

	t := ((pt[0]-a[0])*dx + (pt[1]-a[1])*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))

	closest := orb.Point{a[0] + t*dx, a[1] + t*dy}
	return geo.DistanceHaversine(pt, closest)
}

// LineStringLength returns the length of a LineString in meters.
func LineStringLength(ls orb.LineString) float64 {
	return geo.LengthHaversine(ls)
}
