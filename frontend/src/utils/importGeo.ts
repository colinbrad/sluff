import buffer from '@turf/buffer';
// @ts-expect-error no type declarations available
import * as toGeoJSON from '@mapbox/togeojson';
import type { Feature, FeatureCollection, LineString, MultiLineString, Point, Polygon } from 'geojson';

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
        const text = reader.result as string;
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
            resolve({ type: 'FeatureCollection', features: [{ type: 'Feature', properties: {}, geometry: parsed }] });
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
  const start = coords[0];
  const end = coords[coords.length - 1];
  const line: Feature<LineString> = {
    type: 'Feature',
    properties: {},
    geometry: { type: 'LineString', coordinates: coords },
  };
  const buffered = buffer(line, 0.05, { units: 'kilometers' });
  return {
    name,
    start_point: { type: 'Point', coordinates: [start[0], start[1]] },
    end_point: { type: 'Point', coordinates: [end[0], end[1]] },
    corridor: buffered.geometry as Polygon,
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
      for (const line of (geom as MultiLineString).coordinates) {
        if (line.length >= 2) {
          rounds.push(lineToRound(line, `${name} (${idx})`));
          idx++;
        }
      }
    } else if (geom.type === 'Polygon') {
      // Use polygon as corridor, derive start/end from first and midpoint vertices
      const ring = (geom as Polygon).coordinates[0];
      if (ring.length >= 4) {
        const mid = Math.floor(ring.length / 2);
        rounds.push({
          name,
          start_point: { type: 'Point', coordinates: [ring[0][0], ring[0][1]] },
          end_point: { type: 'Point', coordinates: [ring[mid][0], ring[mid][1]] },
          corridor: geom as Polygon,
        });
        idx++;
      }
    }
  }

  return rounds;
}
