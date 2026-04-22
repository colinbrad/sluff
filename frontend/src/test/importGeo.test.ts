import { describe, it, expect, vi } from 'vitest';
import type { FeatureCollection } from 'geojson';
import { extractRounds, parseFile } from '../utils/importGeo';

// ---------------------------------------------------------------------------
// extractRounds — pure function, no mocking needed
// ---------------------------------------------------------------------------

describe('extractRounds', () => {
  it('extracts a single LineString feature', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: { name: 'My Route' },
          geometry: {
            type: 'LineString',
            coordinates: [[-111.58, 40.59], [-111.57, 40.60], [-111.56, 40.61]],
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(1);
    expect(rounds[0].name).toBe('My Route');
    expect(rounds[0].start_point.type).toBe('Point');
    expect(rounds[0].start_point.coordinates).toEqual([-111.58, 40.59]);
    expect(rounds[0].end_point.coordinates).toEqual([-111.56, 40.61]);
    expect(rounds[0].corridor.type).toBe('Polygon');
    // Corridor should have ring coordinates
    expect(rounds[0].corridor.coordinates.length).toBeGreaterThan(0);
    expect(rounds[0].corridor.coordinates[0].length).toBeGreaterThanOrEqual(4);
  });

  it('assigns fallback name when feature has no name', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: {},
          geometry: {
            type: 'LineString',
            coordinates: [[-111.58, 40.59], [-111.56, 40.61]],
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds[0].name).toBe('Route 1');
  });

  it('increments index for each extracted round', () => {
    const line = {
      type: 'Feature' as const,
      properties: {},
      geometry: {
        type: 'LineString' as const,
        coordinates: [[-111.58, 40.59], [-111.56, 40.61]],
      },
    };

    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [line, line],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(2);
    expect(rounds[0].name).toBe('Route 1');
    expect(rounds[1].name).toBe('Route 2');
  });

  it('skips LineString features with fewer than 2 coordinates', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: {},
          geometry: {
            type: 'LineString',
            coordinates: [[-111.58, 40.59]], // only one point
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(0);
  });

  it('extracts rounds from MultiLineString', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: { name: 'Multi' },
          geometry: {
            type: 'MultiLineString',
            coordinates: [
              [[-111.58, 40.59], [-111.56, 40.61]],
              [[-111.57, 40.60], [-111.55, 40.62]],
            ],
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(2);
    expect(rounds[0].name).toBe('Multi (1)');
    expect(rounds[1].name).toBe('Multi (2)');
    expect(rounds[0].start_point.coordinates).toEqual([-111.58, 40.59]);
    expect(rounds[1].start_point.coordinates).toEqual([-111.57, 40.60]);
  });

  it('skips MultiLineString segments with fewer than 2 coordinates', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: {},
          geometry: {
            type: 'MultiLineString',
            coordinates: [
              [[-111.58, 40.59]], // too short
              [[-111.57, 40.60], [-111.55, 40.62]], // valid
            ],
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(1);
  });

  it('extracts round from Polygon, using first and mid vertex as start/end', () => {
    const ring = [
      [-111.60, 40.58],
      [-111.54, 40.58],
      [-111.54, 40.62],
      [-111.60, 40.62],
      [-111.60, 40.58], // closing
    ];
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: { name: 'Poly Route' },
          geometry: {
            type: 'Polygon',
            coordinates: [ring],
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(1);
    expect(rounds[0].name).toBe('Poly Route');
    // start = first vertex
    expect(rounds[0].start_point.coordinates).toEqual(ring[0]);
    // corridor is the polygon itself
    expect(rounds[0].corridor.type).toBe('Polygon');
    expect(rounds[0].corridor.coordinates[0]).toEqual(ring);
  });

  it('skips Polygon with fewer than 4 ring points', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: {},
          geometry: {
            type: 'Polygon',
            coordinates: [[[-111.60, 40.58], [-111.54, 40.58], [-111.60, 40.58]]],
          },
        },
      ],
    };

    const rounds = extractRounds(fc);
    expect(rounds).toHaveLength(0);
  });

  it('returns empty array for empty FeatureCollection', () => {
    const fc: FeatureCollection = { type: 'FeatureCollection', features: [] };
    expect(extractRounds(fc)).toHaveLength(0);
  });

  it('ignores non-geometry feature types (Point)', () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: {},
          geometry: { type: 'Point', coordinates: [-111.58, 40.59] },
        },
      ],
    };
    // Points are not handled → zero rounds
    expect(extractRounds(fc)).toHaveLength(0);
  });
});

// ---------------------------------------------------------------------------
// parseFile — requires FileReader, mocked via a fake File-like object
// ---------------------------------------------------------------------------

describe('parseFile', () => {
  function makeFile(content: string, name: string): File {
    return new File([content], name, { type: 'application/octet-stream' });
  }

  it('parses a .geojson FeatureCollection', async () => {
    const fc: FeatureCollection = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          properties: {},
          geometry: { type: 'Point', coordinates: [0, 0] },
        },
      ],
    };
    const file = makeFile(JSON.stringify(fc), 'test.geojson');
    const result = await parseFile(file);
    expect(result.type).toBe('FeatureCollection');
    expect(result.features).toHaveLength(1);
  });

  it('wraps a bare Feature in a FeatureCollection', async () => {
    const feature = {
      type: 'Feature',
      properties: {},
      geometry: { type: 'Point', coordinates: [0, 0] },
    };
    const file = makeFile(JSON.stringify(feature), 'route.geojson');
    const result = await parseFile(file);
    expect(result.type).toBe('FeatureCollection');
    expect(result.features).toHaveLength(1);
  });

  it('wraps a bare geometry in a FeatureCollection', async () => {
    const geometry = { type: 'LineString', coordinates: [[0, 0], [1, 1]] };
    const file = makeFile(JSON.stringify(geometry), 'line.json');
    const result = await parseFile(file);
    expect(result.type).toBe('FeatureCollection');
    expect(result.features).toHaveLength(1);
    expect(result.features[0].geometry).toEqual(geometry);
  });

  it('rejects unsupported file extensions', async () => {
    const file = makeFile('some content', 'route.csv');
    await expect(parseFile(file)).rejects.toThrow('Unsupported file format: .csv');
  });

  it('rejects invalid JSON in .geojson file', async () => {
    const file = makeFile('not valid json', 'bad.geojson');
    await expect(parseFile(file)).rejects.toThrow();
  });

  it('parses a .kml file', async () => {
    // Minimal KML with a Placemark LineString
    const kml = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <Placemark>
      <name>Test Route</name>
      <LineString>
        <coordinates>-111.58,40.59,0 -111.56,40.61,0</coordinates>
      </LineString>
    </Placemark>
  </Document>
</kml>`;
    const file = makeFile(kml, 'route.kml');
    const result = await parseFile(file);
    expect(result.type).toBe('FeatureCollection');
    // toGeoJSON should produce at least one feature
    expect(result.features.length).toBeGreaterThan(0);
  });

  it('parses a .gpx file', async () => {
    const gpx = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="40.59" lon="-111.58"></trkpt>
      <trkpt lat="40.61" lon="-111.56"></trkpt>
    </trkseg>
  </trk>
</gpx>`;
    const file = makeFile(gpx, 'track.gpx');
    const result = await parseFile(file);
    expect(result.type).toBe('FeatureCollection');
    expect(result.features.length).toBeGreaterThan(0);
  });
});
