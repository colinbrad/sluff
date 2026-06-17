/** A 2D coordinate tuple as required by maplibre and turf APIs. */
export type Coord2 = [number, number];

/** Convert a coordinate array (number[]) to a [lng, lat] tuple, validating length. */
export function toCoord(c: readonly number[] | undefined): Coord2 {
  if (!c || c.length < 2 || typeof c[0] !== 'number' || typeof c[1] !== 'number') {
    throw new Error('coordinate array must have at least two numbers');
  }
  return [c[0], c[1]];
}
