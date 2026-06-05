import buffer from '@turf/buffer';
// @ts-expect-error no type declarations available
import * as toGeoJSON from '@mapbox/togeojson';
import type {
  Feature,
  FeatureCollection,
  LineString,
  MultiLineString,
  Point,
  Polygon,
} from 'geojson';
import { CORRIDOR_BUFFER_KM } from '../constants';
import { toCoord } from './geojson';

export interface ImportedRound {
  name: string;
  start_point: Point;
  end_point: Point;
  corridor: Polygon;
}

/** Parse an uploaded file into GeoJSON features */
export function parseFile(file: File): Promise<FeatureCollection> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const text = String(reader.result ?? '');
        const ext = file.name.split('.').pop()?.toLowerCase();

        if (ext === 'geojson' || ext === 'json') {
          const parsed = JSON.parse(text);
          // Handle bare geometry or feature
          if (parsed.type === 'FeatureCollection') {
            resolve(parsed);
          } else if (parsed.type === 'Feature') {
            resolve({ type: 'FeatureCollection', features: [parsed] });
          } else {
            // Bare geometry
            resolve({
              type: 'FeatureCollection',
              features: [{ type: 'Feature', properties: {}, geometry: parsed }],
            });
          }
        } else if (ext === 'kml') {
          const dom = new DOMParser().parseFromString(text, 'application/xml');
          resolve(toGeoJSON.kml(dom));
        } else if (ext === 'gpx') {
          const dom = new DOMParser().parseFromString(text, 'application/xml');
          resolve(toGeoJSON.gpx(dom));
        } else {
          reject(new Error(`Unsupported file format: .${ext}`));
        }
      } catch (err) {
        reject(err);
      }
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsText(file);
  });
}

/** Convert a LineString to start point, end point, and buffered corridor polygon */
function lineToRound(coords: number[][], name: string): ImportedRound {
  const start = toCoord(coords[0]);
  const end = toCoord(coords[coords.length - 1]);
  const line: Feature<LineString> = {
    type: 'Feature',
    properties: {},
    geometry: { type: 'LineString', coordinates: coords },
  };
  const buffered = buffer(line, CORRIDOR_BUFFER_KM, { units: 'kilometers' });
  if (!buffered) {
    throw new Error(`Failed to buffer line "${name}"`);
  }
  // @turf/buffer can return Polygon or MultiPolygon (e.g. self-crossing lines);
  // take the first polygon of a MultiPolygon to avoid aborting the import.
  let corridor: Polygon;
  if (buffered.geometry.type === 'Polygon') {
    corridor = buffered.geometry;
  } else if (buffered.geometry.type === 'MultiPolygon') {
    const firstRing = buffered.geometry.coordinates[0];
    if (!firstRing) throw new Error(`Buffer produced empty MultiPolygon for "${name}"`);
    corridor = { type: 'Polygon', coordinates: firstRing };
  } else {
    throw new Error(`Buffer returned unexpected geometry for "${name}"`);
  }
  return {
    name,
    start_point: { type: 'Point', coordinates: start },
    end_point: { type: 'Point', coordinates: end },
    corridor,
  };
}

/** Extract rounds from a GeoJSON FeatureCollection */
export function extractRounds(fc: FeatureCollection): ImportedRound[] {
  const rounds: ImportedRound[] = [];
  let idx = 1;

  for (const feature of fc.features) {
    const name = (feature.properties?.name as string) || `Route ${idx}`;
    const geom = feature.geometry;

    if (geom.type === 'LineString') {
      if (geom.coordinates.length >= 2) {
        rounds.push(lineToRound(geom.coordinates, name));
        idx++;
      }
    } else if (geom.type === 'MultiLineString') {
      const multi: MultiLineString = geom;
      for (const line of multi.coordinates) {
        if (line.length >= 2) {
          rounds.push(lineToRound(line, `${name} (${idx})`));
          idx++;
        }
      }
    } else if (geom.type === 'Polygon') {
      // Use polygon as corridor, derive start/end from first and midpoint vertices.
      const polygon: Polygon = geom;
      const ring = polygon.coordinates[0];
      if (ring && ring.length >= 4) {
        const mid = Math.floor(ring.length / 2);
        rounds.push({
          name,
          start_point: { type: 'Point', coordinates: toCoord(ring[0]) },
          end_point: { type: 'Point', coordinates: toCoord(ring[mid]) },
          corridor: polygon,
        });
        idx++;
      }
    }
  }

  return rounds;
}
