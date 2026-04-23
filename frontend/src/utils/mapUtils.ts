import maplibregl from 'maplibre-gl';
import type { Round } from '../types/game';

/**
 * Adds start (green) and end (red) markers for a round to the map.
 * Returns the created markers so the caller can remove them later.
 */
export function addRoundMarkers(map: maplibregl.Map, round: Round): maplibregl.Marker[] {
  const markers: maplibregl.Marker[] = [];

  if (round.start_point?.coordinates) {
    markers.push(
      new maplibregl.Marker({ color: '#10B981' })
        .setLngLat(round.start_point.coordinates as [number, number])
        .setPopup(new maplibregl.Popup().setText('Start'))
        .addTo(map),
    );
  }

  if (round.end_point?.coordinates) {
    markers.push(
      new maplibregl.Marker({ color: '#EF4444' })
        .setLngLat(round.end_point.coordinates as [number, number])
        .setPopup(new maplibregl.Popup().setText('End'))
        .addTo(map),
    );
  }

  return markers;
}

/**
 * Adds fill + dashed-outline layers for each no-go zone polygon.
 * Returns the source IDs so the caller can remove them later.
 * Source IDs follow the pattern `${prefix}-${index}`.
 */
export function addNoGoZoneLayers(
  map: maplibregl.Map,
  zones: GeoJSON.Polygon[],
  prefix: string,
): string[] {
  return zones.map((zone, i) => {
    const srcId = `${prefix}-${i}`;
    map.addSource(srcId, {
      type: 'geojson',
      data: { type: 'Feature', geometry: zone, properties: {} },
    });
    map.addLayer({
      id: `${srcId}-fill`,
      type: 'fill',
      source: srcId,
      paint: { 'fill-color': '#EF4444', 'fill-opacity': 0.25 },
    });
    map.addLayer({
      id: `${srcId}-outline`,
      type: 'line',
      source: srcId,
      paint: { 'line-color': '#EF4444', 'line-width': 2, 'line-dasharray': [3, 2] },
    });
    return srcId;
  });
}

/**
 * Removes no-go zone layers and sources previously added by addNoGoZoneLayers.
 */
export function removeNoGoZoneLayers(map: maplibregl.Map, sourceIds: string[]): void {
  for (const id of sourceIds) {
    if (map.getLayer(`${id}-fill`)) map.removeLayer(`${id}-fill`);
    if (map.getLayer(`${id}-outline`)) map.removeLayer(`${id}-outline`);
    if (map.getSource(id)) map.removeSource(id);
  }
}
