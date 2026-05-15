import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import maplibregl from 'maplibre-gl';
import buffer from '@turf/buffer';
import type { Feature, FeatureCollection, LineString } from 'geojson';
import GameMapComponent from '../map/GameMap';
import * as api from '../../services/api';

export type FeatureLabel = 'start' | 'end' | 'corridor' | 'no-go' | null;

interface LabeledFeature {
  id: string;
  feature: Feature;
  label: FeatureLabel;
}

export interface ImportLocationState {
  features: Feature[];
  mapName: string;
}

const LABEL_COLORS: Record<string, string> = {
  start: '#10B981',
  end: '#EF4444',
  corridor: '#3B82F6',
  'no-go': '#DC2626',
  none: '#9CA3AF',
};

const LABEL_OPTIONS: { value: NonNullable<FeatureLabel>; label: string }[] = [
  { value: 'start', label: 'Start' },
  { value: 'end', label: 'End' },
  { value: 'corridor', label: 'Corridor' },
  { value: 'no-go', label: 'No-Go' },
];

function geomKind(f: Feature): 'point' | 'polygon' | 'line' | 'other' {
  const t = f.geometry?.type;
  if (t === 'Point' || t === 'MultiPoint') return 'point';
  if (t === 'Polygon' || t === 'MultiPolygon') return 'polygon';
  if (t === 'LineString' || t === 'MultiLineString') return 'line';
  return 'other';
}

function featureName(f: Feature, index: number): string {
  return (f.properties?.name as string)
    || (f.properties?.Name as string)
    || `Feature ${index + 1}`;
}

function buildFC(items: LabeledFeature[], selectedId: string | null): FeatureCollection {
  return {
    type: 'FeatureCollection',
    features: items.map((item) => ({
      ...item.feature,
      id: Number(item.id),
      properties: {
        ...(item.feature.properties ?? {}),
        _id: item.id,
        _label: item.label ?? 'none',
        _sel: item.id === selectedId ? 1 : 0,
      },
    })),
  };
}

function collectBounds(features: Feature[]): maplibregl.LngLatBounds | null {
  const coords: [number, number][] = [];
  const add = (c: number[]) => coords.push([c[0], c[1]]);
  for (const f of features) {
    const g = f.geometry;
    if (!g) continue;
    if (g.type === 'Point') add(g.coordinates);
    else if (g.type === 'MultiPoint') g.coordinates.forEach(add);
    else if (g.type === 'LineString') g.coordinates.forEach(add);
    else if (g.type === 'MultiLineString') g.coordinates.forEach((l) => l.forEach(add));
    else if (g.type === 'Polygon') g.coordinates[0].forEach(add);
    else if (g.type === 'MultiPolygon') g.coordinates.forEach((p) => p[0].forEach(add));
  }
  if (coords.length === 0) return null;
  return coords.reduce(
    (b, c) => b.extend(c),
    new maplibregl.LngLatBounds(coords[0], coords[0])
  );
}

export default function ImportLabeler() {
  const location = useLocation();
  const navigate = useNavigate();
  const state = location.state as ImportLocationState | null;

  const [items, setItems] = useState<LabeledFeature[]>(() =>
    (state?.features ?? []).map((f, i) => ({ id: String(i), feature: f, label: null }))
  );
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [mapName, setMapName] = useState(state?.mapName ?? '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const mapRef = useRef<maplibregl.Map | null>(null);
  const itemsRef = useRef(items);
  const selectedIdRef = useRef(selectedId);
  itemsRef.current = items;
  selectedIdRef.current = selectedId;

  // Keep map source in sync
  useEffect(() => {
    const src = mapRef.current?.getSource('import-features') as maplibregl.GeoJSONSource | undefined;
    src?.setData(buildFC(items, selectedId));
  }, [items, selectedId]);

  const initMap = useCallback((map: maplibregl.Map) => {
    mapRef.current = map;

    const fc = buildFC(itemsRef.current, selectedIdRef.current);

    map.addSource('import-features', { type: 'geojson', data: fc });

    map.addLayer({
      id: 'import-poly-fill',
      type: 'fill',
      source: 'import-features',
      filter: ['match', ['geometry-type'], ['Polygon', 'MultiPolygon'], true, false],
      paint: {
        'fill-color': ['match', ['get', '_label'], 'start', '#10B981', 'end', '#EF4444', 'corridor', '#3B82F6', 'no-go', '#DC2626', '#9CA3AF'],
        'fill-opacity': ['case', ['==', ['get', '_sel'], 1], 0.45, 0.2],
      },
    });

    map.addLayer({
      id: 'import-poly-outline',
      type: 'line',
      source: 'import-features',
      filter: ['match', ['geometry-type'], ['Polygon', 'MultiPolygon'], true, false],
      paint: {
        'line-color': ['match', ['get', '_label'], 'start', '#10B981', 'end', '#EF4444', 'corridor', '#3B82F6', 'no-go', '#DC2626', '#9CA3AF'],
        'line-width': ['case', ['==', ['get', '_sel'], 1], 3, 2],
        'line-dasharray': [3, 2],
      },
    });

    map.addLayer({
      id: 'import-line',
      type: 'line',
      source: 'import-features',
      filter: ['match', ['geometry-type'], ['LineString', 'MultiLineString'], true, false],
      paint: {
        'line-color': ['match', ['get', '_label'], 'start', '#10B981', 'end', '#EF4444', 'corridor', '#3B82F6', 'no-go', '#DC2626', '#9CA3AF'],
        'line-width': ['case', ['==', ['get', '_sel'], 1], 4, 2.5],
      },
    });

    map.addLayer({
      id: 'import-point-halo',
      type: 'circle',
      source: 'import-features',
      filter: ['match', ['geometry-type'], ['Point', 'MultiPoint'], true, false],
      paint: {
        'circle-radius': 14,
        'circle-color': '#FBBF24',
        'circle-opacity': ['case', ['==', ['get', '_sel'], 1], 0.5, 0],
      },
    });

    map.addLayer({
      id: 'import-point',
      type: 'circle',
      source: 'import-features',
      filter: ['match', ['geometry-type'], ['Point', 'MultiPoint'], true, false],
      paint: {
        'circle-radius': 8,
        'circle-color': ['match', ['get', '_label'], 'start', '#10B981', 'end', '#EF4444', 'corridor', '#3B82F6', 'no-go', '#DC2626', '#9CA3AF'],
        'circle-stroke-width': 2,
        'circle-stroke-color': '#ffffff',
      },
    });

    const clickable = ['import-poly-fill', 'import-line', 'import-point'];
    let suppressNext = false;

    for (const layerId of clickable) {
      map.on('click', layerId, (e) => {
        const id = e.features?.[0]?.properties?._id as string | undefined;
        if (id != null) {
          suppressNext = true;
          setSelectedId((prev) => (prev === id ? null : id));
        }
      });
      map.on('mouseenter', layerId, () => { map.getCanvas().style.cursor = 'pointer'; });
      map.on('mouseleave', layerId, () => { map.getCanvas().style.cursor = ''; });
    }

    map.on('click', () => {
      if (suppressNext) { suppressNext = false; return; }
      setSelectedId(null);
    });

    const bounds = collectBounds(itemsRef.current.map((i) => i.feature));
    if (bounds) map.fitBounds(bounds, { padding: 80 });
  }, []);

  const setLabel = (id: string, label: FeatureLabel) => {
    setItems((prev) => prev.map((item) => item.id === id ? { ...item, label } : item));
  };

  const startCount = items.filter((i) => i.label === 'start').length;
  const endCount = items.filter((i) => i.label === 'end').length;
  const corridorCount = items.filter((i) => i.label === 'corridor').length;
  const canSave = startCount === 1 && endCount === 1 && corridorCount === 1 && mapName.trim() !== '';

  const handleSave = async () => {
    if (!canSave) return;
    setSaving(true);
    setError('');
    try {
      const startItem = items.find((i) => i.label === 'start')!;
      const endItem = items.find((i) => i.label === 'end')!;
      const corridorItem = items.find((i) => i.label === 'corridor')!;
      const noGoItems = items.filter((i) => i.label === 'no-go');

      // Resolve start point geometry
      let startGeom: GeoJSON.Geometry = startItem.feature.geometry;
      if (startGeom.type !== 'Point') {
        const c = startGeom.type === 'Polygon' ? startGeom.coordinates[0][0]
          : startGeom.type === 'LineString' ? startGeom.coordinates[0]
          : [0, 0];
        startGeom = { type: 'Point', coordinates: c };
      }

      // Resolve end point geometry
      let endGeom: GeoJSON.Geometry = endItem.feature.geometry;
      if (endGeom.type !== 'Point') {
        const c = endGeom.type === 'Polygon'
          ? endGeom.coordinates[0][Math.floor(endGeom.coordinates[0].length / 2)]
          : endGeom.type === 'LineString'
          ? endGeom.coordinates[endGeom.coordinates.length - 1]
          : [0, 0];
        endGeom = { type: 'Point', coordinates: c };
      }

      // Resolve corridor — buffer lines automatically
      let corridorGeom: GeoJSON.Geometry = corridorItem.feature.geometry;
      if (corridorGeom.type === 'LineString' || corridorGeom.type === 'MultiLineString') {
        const buffered = buffer(corridorItem.feature as Feature<LineString>, 0.05, { units: 'kilometers' });
        if (!buffered) throw new Error('Failed to buffer corridor line');
        corridorGeom = buffered.geometry;
      }
      if (corridorGeom.type !== 'Polygon') {
        throw new Error('Corridor must be a polygon (or line, which gets auto-buffered)');
      }

      const noGoZones = noGoItems
        .filter((i) => i.feature.geometry.type === 'Polygon')
        .map((i) => i.feature.geometry as GeoJSON.Polygon);

      const m = await api.createMap(mapName.trim(), 'Imported');
      await api.createRound(m.id, {
        round_number: 1,
        name: 'Round 1',
        start_point: startGeom,
        end_point: endGeom,
        corridor: corridorGeom,
        no_go_zones: noGoZones,
      });

      navigate(`/guide/maps/${m.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  if (!state?.features?.length) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <p className="text-gray-600 mb-4">No features to label.</p>
          <button onClick={() => navigate('/guide')} className="text-blue-600 hover:underline text-sm">
            Back to Dashboard
          </button>
        </div>
      </div>
    );
  }

  const selected = selectedId != null ? items.find((i) => i.id === selectedId) : null;

  return (
    <div className="h-screen flex flex-col">
      <header className="bg-white shadow-sm border-b px-4 py-3 flex items-center gap-4 z-10">
        <button onClick={() => navigate('/guide')} className="text-gray-500 hover:text-gray-700 shrink-0">
          &larr; Back
        </button>
        <h1 className="font-bold text-lg truncate">Label Imported Features</h1>
      </header>

      <div className="flex-1 flex min-h-0">
        {/* Sidebar */}
        <div className="w-72 bg-white border-r flex flex-col overflow-hidden shrink-0">
          {/* Map name */}
          <div className="p-4 border-b">
            <label className="block text-xs font-medium text-gray-600 mb-1 uppercase tracking-wide">Map Name</label>
            <input
              type="text"
              value={mapName}
              onChange={(e) => setMapName(e.target.value)}
              placeholder="e.g., Wasatch Backcountry"
              className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {/* Instructions */}
          <div className="px-4 py-2.5 bg-blue-50 border-b">
            <p className="text-xs text-blue-700 leading-relaxed">
              Click a feature on the map or in the list below, then assign it a label. You need exactly one Start, End, and Corridor to save.
            </p>
          </div>

          {/* Label picker */}
          {selected && (
            <div className="p-3 border-b bg-amber-50">
              <p className="text-xs text-gray-600 mb-2 font-medium truncate">
                {featureName(selected.feature, Number(selected.id))}
              </p>
              <div className="flex flex-wrap gap-1.5">
                {LABEL_OPTIONS.map((opt) => {
                  const active = selected.label === opt.value;
                  return (
                    <button
                      key={opt.value}
                      onClick={() => setLabel(selected.id, active ? null : opt.value)}
                      className="px-2.5 py-1 rounded text-xs font-medium transition-colors"
                      style={active
                        ? { backgroundColor: LABEL_COLORS[opt.value], color: '#fff' }
                        : { backgroundColor: '#F3F4F6', color: '#374151' }
                      }
                    >
                      {opt.label}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Feature list */}
          <div className="flex-1 overflow-y-auto p-2 flex flex-col gap-1">
            {items.map((item) => {
              const name = featureName(item.feature, Number(item.id));
              const kind = geomKind(item.feature);
              const isSelected = item.id === selectedId;
              const color = item.label ? LABEL_COLORS[item.label] : LABEL_COLORS.none;
              return (
                <button
                  key={item.id}
                  onClick={() => setSelectedId(isSelected ? null : item.id)}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-left w-full transition-colors ${
                    isSelected ? 'bg-amber-50 ring-1 ring-amber-300' : 'bg-gray-50 hover:bg-gray-100'
                  }`}
                >
                  <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: color }} />
                  <span className="flex-1 truncate text-gray-800 text-xs">{name}</span>
                  <span className="text-gray-400 text-xs shrink-0">{kind}</span>
                  {item.label && (
                    <span
                      className="text-xs font-semibold px-1.5 py-0.5 rounded text-white shrink-0"
                      style={{ backgroundColor: color }}
                    >
                      {item.label}
                    </span>
                  )}
                </button>
              );
            })}
          </div>

          {/* Status + save */}
          <div className="p-3 border-t bg-gray-50 flex flex-col gap-2">
            <div className="flex gap-3 text-xs">
              <span className={startCount === 1 ? 'text-green-600 font-medium' : 'text-gray-400'}>
                Start {startCount}/1
              </span>
              <span className={endCount === 1 ? 'text-green-600 font-medium' : 'text-gray-400'}>
                End {endCount}/1
              </span>
              <span className={corridorCount === 1 ? 'text-green-600 font-medium' : 'text-gray-400'}>
                Corridor {corridorCount}/1
              </span>
            </div>
            {error && <p className="text-xs text-red-600">{error}</p>}
            <button
              onClick={handleSave}
              disabled={!canSave || saving}
              className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm font-semibold transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {saving ? 'Saving...' : 'Save Round'}
            </button>
          </div>
        </div>

        {/* Map */}
        <div className="flex-1 relative">
          <GameMapComponent onMapReady={initMap} />
        </div>
      </div>
    </div>
  );
}
