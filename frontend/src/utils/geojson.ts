import type { Geometry, LineString, Point, Polygon } from 'geojson';

/** A 2D coordinate tuple as required by maplibre and turf APIs. */
export type Coord2 = [number, number];

function expectGeometry<T extends Geometry>(g: Geometry, type: T['type']): T {
  if (g.type !== type) {
    throw new Error(`expected ${type} geometry, got ${g.type}`);
  }
  return g as T;
}

export function asPoint(g: Geometry): Point {
  return expectGeometry<Point>(g, 'Point');
}

export function asLineString(g: Geometry): LineString {
  return expectGeometry<LineString>(g, 'LineString');
}

export function asPolygon(g: Geometry): Polygon {
  return expectGeometry<Polygon>(g, 'Polygon');
}

/** Convert a coordinate array (number[]) to a [lng, lat] tuple, validating length. */
export function toCoord(c: readonly number[] | undefined): Coord2 {
  if (!c || c.length < 2 || typeof c[0] !== 'number' || typeof c[1] !== 'number') {
    throw new Error('coordinate array must have at least two numbers');
  }
  return [c[0], c[1]];
}
