package geo

import (
	"math"
	"testing"

	"github.com/paulmach/orb"
)

// A simple square polygon around Salt Lake City area for testing.
// Corners at roughly (-111.9,40.7), (-111.8,40.7), (-111.8,40.8), (-111.9,40.8).
func testSquare() orb.Polygon {
	return orb.Polygon{
		orb.Ring{
			{-111.9, 40.7},
			{-111.8, 40.7},
			{-111.8, 40.8},
			{-111.9, 40.8},
			{-111.9, 40.7}, // closing
		},
	}
}

func TestSampleLineString(t *testing.T) {
	t.Run("returns correct number of samples", func(t *testing.T) {
		ls := orb.LineString{
			{-111.9, 40.7},
			{-111.85, 40.75},
			{-111.8, 40.8},
		}
		// SampleLineString with n=10 returns n+1 points (0..n inclusive)
		samples := SampleLineString(ls, 10)
		expected := 11
		if len(samples) != expected {
			t.Errorf("expected %d samples, got %d", expected, len(samples))
		}
	})

	t.Run("first sample is start of line", func(t *testing.T) {
		ls := orb.LineString{
			{-111.9, 40.7},
			{-111.8, 40.8},
		}
		samples := SampleLineString(ls, 5)
		if samples[0] != ls[0] {
			t.Errorf("first sample should equal line start, got %v", samples[0])
		}
	})

	t.Run("last sample is end of line", func(t *testing.T) {
		ls := orb.LineString{
			{-111.9, 40.7},
			{-111.8, 40.8},
		}
		samples := SampleLineString(ls, 5)
		last := samples[len(samples)-1]
		if last != ls[len(ls)-1] {
			t.Errorf("last sample should equal line end, got %v", last)
		}
	})

	t.Run("returns nil for empty line", func(t *testing.T) {
		samples := SampleLineString(orb.LineString{}, 10)
		if samples != nil {
			t.Errorf("expected nil for empty line, got %v", samples)
		}
	})

	t.Run("returns nil for single point line", func(t *testing.T) {
		samples := SampleLineString(orb.LineString{{-111.9, 40.7}}, 10)
		if samples != nil {
			t.Errorf("expected nil for single-point line, got %v", samples)
		}
	})

	t.Run("returns nil for n <= 0", func(t *testing.T) {
		ls := orb.LineString{{-111.9, 40.7}, {-111.8, 40.8}}
		samples := SampleLineString(ls, 0)
		if samples != nil {
			t.Errorf("expected nil for n=0, got %v", samples)
		}
		samples = SampleLineString(ls, -1)
		if samples != nil {
			t.Errorf("expected nil for n=-1, got %v", samples)
		}
	})
}

func TestPointInPolygon(t *testing.T) {
	poly := testSquare()

	t.Run("point inside returns true", func(t *testing.T) {
		inside := orb.Point{-111.85, 40.75} // center of square
		if !PointInPolygon(inside, poly) {
			t.Error("expected point inside polygon to return true")
		}
	})

	t.Run("point outside returns false", func(t *testing.T) {
		outside := orb.Point{-112.0, 40.75} // west of square
		if PointInPolygon(outside, poly) {
			t.Error("expected point outside polygon to return false")
		}
	})

	t.Run("point far away returns false", func(t *testing.T) {
		far := orb.Point{-100.0, 30.0}
		if PointInPolygon(far, poly) {
			t.Error("expected distant point to be outside polygon")
		}
	})
}

func TestLineStringLength(t *testing.T) {
	t.Run("returns reasonable distance", func(t *testing.T) {
		// A line spanning about 0.1 degrees of latitude (~11 km)
		ls := orb.LineString{
			{-111.85, 40.7},
			{-111.85, 40.8},
		}
		length := LineStringLength(ls)
		// 0.1 degrees latitude is approximately 11,100 meters
		if length < 10000 || length > 12000 {
			t.Errorf("expected length around 11100m, got %.1f", length)
		}
	})

	t.Run("empty line has zero length", func(t *testing.T) {
		length := LineStringLength(orb.LineString{})
		if length != 0 {
			t.Errorf("expected 0 for empty line, got %f", length)
		}
	})

	t.Run("single point has zero length", func(t *testing.T) {
		length := LineStringLength(orb.LineString{{-111.85, 40.7}})
		if length != 0 {
			t.Errorf("expected 0 for single point, got %f", length)
		}
	})
}

func TestMinDistanceToPolygonBoundary(t *testing.T) {
	poly := testSquare()

	t.Run("boundary point returns approximately zero", func(t *testing.T) {
		// A point on the southern edge of the square
		onBoundary := orb.Point{-111.85, 40.7}
		dist := MinDistanceToPolygonBoundary(onBoundary, poly)
		// Should be very close to 0 (within floating point tolerance)
		if dist > 1.0 { // within 1 meter
			t.Errorf("expected ~0 distance for boundary point, got %f", dist)
		}
	})

	t.Run("outside point returns positive distance", func(t *testing.T) {
		outside := orb.Point{-111.85, 40.65} // south of the square
		dist := MinDistanceToPolygonBoundary(outside, poly)
		if dist <= 0 {
			t.Errorf("expected positive distance for outside point, got %f", dist)
		}
		// ~0.05 degrees lat south of boundary ≈ ~5500m
		if dist < 4000 || dist > 7000 {
			t.Errorf("expected distance around 5500m, got %.1f", dist)
		}
	})

	t.Run("inside point returns positive distance to nearest edge", func(t *testing.T) {
		// Slightly inside the southern edge
		inside := orb.Point{-111.85, 40.71}
		dist := MinDistanceToPolygonBoundary(inside, poly)
		// Should be distance to nearest boundary edge (south edge at 40.7)
		// ~0.01 degrees ≈ ~1100m
		if dist < 500 || dist > 2000 {
			t.Errorf("expected distance around 1100m, got %.1f", dist)
		}
	})

	t.Run("empty polygon returns MaxFloat64", func(t *testing.T) {
		dist := MinDistanceToPolygonBoundary(orb.Point{0, 0}, orb.Polygon{})
		if dist != math.MaxFloat64 {
			t.Errorf("expected MaxFloat64 for empty polygon, got %f", dist)
		}
	})
}
