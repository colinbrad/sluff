import { useEffect, useState, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import maplibregl from 'maplibre-gl';
import {
  TerraDraw,
  TerraDrawPolygonMode,
  TerraDrawPointMode,
  TerraDrawSelectMode,
} from 'terra-draw';
import { TerraDrawMapLibreGLAdapter } from 'terra-draw-maplibre-gl-adapter';
import type { GameMap, Round } from '../../types/game';

type FeatureId = string | number;
import * as api from '../../services/api';
import GameMapComponent from '../map/GameMap';
import MapOverlayControls from '../map/MapOverlayControls';

function useMobileBreakpoint(breakpoint = 768) {
  const [isMobile, setIsMobile] = useState(
    typeof window !== 'undefined' ? window.innerWidth < breakpoint : false
  );
  useEffect(() => {
    const mq = window.matchMedia(`(max-width: ${breakpoint - 1}px)`);
    const handler = (e: MediaQueryListEvent) => setIsMobile(e.matches);
    setIsMobile(mq.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [breakpoint]);
  return isMobile;
}

type Step = 'idle' | 'start' | 'end' | 'corridor' | 'no_go_zone' | 'review';

const STEP_INFO: Record<Exclude<Step, 'idle'>, { label: string; number: number; instruction: string }> = {
  start: { label: 'Start Point', number: 1, instruction: 'Click on the map to place the start point.' },
  end: { label: 'End Point', number: 2, instruction: 'Click on the map to place the end point.' },
  corridor: { label: 'Corridor', number: 3, instruction: 'Click to draw the corridor polygon. Double-click or click the first point to close it.' },
  no_go_zone: { label: 'No-Go Zones', number: 4, instruction: 'Draw areas players should avoid. Click "Draw Zone" to add one, or skip if none needed.' },
  review: { label: 'Review', number: 5, instruction: 'Review your round and click Save when ready.' },
};

export default function MapEditor() {
  const { mapId } = useParams<{ mapId: string }>();
  const navigate = useNavigate();
  const [gameMap, setGameMap] = useState<GameMap | null>(null);
  const [editingRound, setEditingRound] = useState<number | null>(null);
  const [roundName, setRoundName] = useState('');
  const [step, setStep] = useState<Step>('idle');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [placed, setPlaced] = useState({ start: false, end: false, corridor: false });
  const [noGoZoneIds, setNoGoZoneIds] = useState<FeatureId[]>([]); // optional, can be empty
  const [terrain3d, setTerrain3d] = useState(false);
  const [slopeShading, setSlopeShading] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const isMobile = useMobileBreakpoint();

  const mapRef = useRef<maplibregl.Map | null>(null);
  const drawRef = useRef<TerraDraw | null>(null);
  const roundMarkersRef = useRef<maplibregl.Marker[]>([]);

  // Track drawn feature IDs
  const startPointId = useRef<FeatureId | null>(null);
  const endPointId = useRef<FeatureId | null>(null);
  const corridorId = useRef<FeatureId | null>(null);

  useEffect(() => {
    if (mapId) {
      api.getMap(mapId).then(setGameMap);
    }
  }, [mapId]);

  // Display saved rounds as static map layers
  const showSavedRounds = useCallback(() => {
    const map = mapRef.current;
    if (!map) return;

    for (const m of roundMarkersRef.current) {
      m.remove();
    }
    roundMarkersRef.current = [];

    const existingLayers = map.getStyle().layers || [];
    for (const layer of existingLayers) {
      if (layer.id.startsWith('saved-round-')) {
        map.removeLayer(layer.id);
      }
    }
    const sources = map.getStyle().sources || {};
    for (const srcId of Object.keys(sources)) {
      if (srcId.startsWith('saved-round-')) {
        map.removeSource(srcId);
      }
    }

    const rounds = gameMap?.rounds || [];
    for (const round of rounds) {
      const srcId = `saved-round-${round.id}`;

      if (round.corridor?.coordinates) {
        map.addSource(srcId, {
          type: 'geojson',
          data: {
            type: 'Feature',
            geometry: round.corridor,
            properties: {},
          },
        });
        map.addLayer({
          id: `${srcId}-fill`,
          type: 'fill',
          source: srcId,
          paint: { 'fill-color': '#3B82F6', 'fill-opacity': 0.12 },
        });
        map.addLayer({
          id: `${srcId}-outline`,
          type: 'line',
          source: srcId,
          paint: { 'line-color': '#3B82F6', 'line-width': 2, 'line-dasharray': [3, 2] },
        });
      }

      // No-go zones: red fill
      for (let zi = 0; zi < (round.no_go_zones?.length ?? 0); zi++) {
        const zone = round.no_go_zones![zi];
        const zoneId = `${srcId}-nogo-${zi}`;
        map.addSource(zoneId, {
          type: 'geojson',
          data: { type: 'Feature', geometry: zone, properties: {} },
        });
        map.addLayer({
          id: `${zoneId}-fill`,
          type: 'fill',
          source: zoneId,
          paint: { 'fill-color': '#EF4444', 'fill-opacity': 0.25 },
        });
        map.addLayer({
          id: `${zoneId}-outline`,
          type: 'line',
          source: zoneId,
          paint: { 'line-color': '#EF4444', 'line-width': 2, 'line-dasharray': [3, 2] },
        });
      }

      if (round.start_point?.coordinates) {
        const startMarker = new maplibregl.Marker({ color: '#10B981', scale: 0.8 })
          .setLngLat(round.start_point.coordinates as [number, number])
          .setPopup(new maplibregl.Popup({ offset: 25 }).setText(`#${round.round_number} Start`))
          .addTo(map);
        roundMarkersRef.current.push(startMarker);
      }

      if (round.end_point?.coordinates) {
        const endMarker = new maplibregl.Marker({ color: '#EF4444', scale: 0.8 })
          .setLngLat(round.end_point.coordinates as [number, number])
          .setPopup(new maplibregl.Popup({ offset: 25 }).setText(`#${round.round_number} End`))
          .addTo(map);
        roundMarkersRef.current.push(endMarker);
      }
    }
  }, [gameMap]);

  useEffect(() => {
    showSavedRounds();
  }, [showSavedRounds]);

  const initDraw = useCallback((map: maplibregl.Map) => {
    mapRef.current = map;

    const draw = new TerraDraw({
      adapter: new TerraDrawMapLibreGLAdapter({ map }),
      modes: [
        new TerraDrawPointMode(),
        new TerraDrawPolygonMode(),
        new TerraDrawSelectMode({
          flags: {
            point: { feature: { draggable: true } },
            polygon: {
              feature: { draggable: true, coordinates: { draggable: true, deletable: true } },
            },
          },
        }),
      ],
    });

    draw.start();
    draw.setMode('select');
    drawRef.current = draw;
    showSavedRounds();
  }, [showSavedRounds]);

  // Set TerraDraw mode when step changes
  useEffect(() => {
    const draw = drawRef.current;
    if (!draw) return;

    switch (step) {
      case 'start':
      case 'end':
        draw.setMode('point');
        break;
      case 'corridor':
        draw.setMode('polygon');
        break;
      case 'no_go_zone':
      case 'review':
      case 'idle':
        draw.setMode('select');
        break;
    }
  }, [step]);

  const clearDrawing = () => {
    const draw = drawRef.current;
    if (!draw) return;
    draw.clear();
    startPointId.current = null;
    endPointId.current = null;
    corridorId.current = null;
    setPlaced({ start: false, end: false, corridor: false });
    setNoGoZoneIds([]);
  };

  // Listen for draw finish events and auto-advance steps
  useEffect(() => {
    const draw = drawRef.current;
    if (!draw) return;

    const handleFinish = (id: FeatureId) => {
      const snapshot = draw.getSnapshot();
      const feature = snapshot.find((f) => f.id === id);
      if (!feature) return;

      if (step === 'start' && feature.geometry.type === 'Point') {
        if (startPointId.current != null) {
          try { draw.removeFeatures([startPointId.current]); } catch { /* noop */ }
        }
        startPointId.current = id;
        setPlaced((p) => ({ ...p, start: true }));
        // Auto-advance: skip end if already placed, skip corridor if already placed
        if (endPointId.current == null) {
          setStep('end');
        } else if (corridorId.current == null) {
          setStep('corridor');
        } else {
          setStep('review');
        }
      } else if (step === 'end' && feature.geometry.type === 'Point') {
        if (endPointId.current != null) {
          try { draw.removeFeatures([endPointId.current]); } catch { /* noop */ }
        }
        endPointId.current = id;
        setPlaced((p) => ({ ...p, end: true }));
        if (corridorId.current == null) {
          setStep('corridor');
        } else {
          setStep('review');
        }
      } else if (step === 'corridor' && feature.geometry.type === 'Polygon') {
        if (corridorId.current != null) {
          try { draw.removeFeatures([corridorId.current]); } catch { /* noop */ }
        }
        corridorId.current = id;
        setPlaced((p) => ({ ...p, corridor: true }));
        setStep('no_go_zone');
      } else if (step === 'no_go_zone' && feature.geometry.type === 'Polygon') {
        setNoGoZoneIds((prev) => [...prev, id]);
        draw.setMode('select');
      }
    };

    draw.on('finish', handleFinish);
    return () => {
      draw.off('finish', handleFinish);
    };
  }, [step]);

  const loadRound = (round: Round) => {
    const draw = drawRef.current;
    const map = mapRef.current;
    if (!draw || !map) return;

    clearDrawing();

    // terra-draw addFeatures expects GeoJSONStoreFeatures (Point | LineString | Polygon only)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const featuresToAdd: any[] = [
      { type: 'Feature', geometry: round.start_point, properties: { mode: 'point' } },
      { type: 'Feature', geometry: round.end_point, properties: { mode: 'point' } },
      { type: 'Feature', geometry: round.corridor, properties: { mode: 'polygon' } },
    ];
    const noGoCount = round.no_go_zones?.length ?? 0;
    for (const zone of round.no_go_zones ?? []) {
      featuresToAdd.push({ type: 'Feature', geometry: zone, properties: { mode: 'polygon' } });
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const results = draw.addFeatures(featuresToAdd as any);

    const newPlaced = { start: false, end: false, corridor: false };
    if (results[0]?.id != null) {
      startPointId.current = results[0].id;
      newPlaced.start = true;
    }
    if (results[1]?.id != null) {
      endPointId.current = results[1].id;
      newPlaced.end = true;
    }
    if (results[2]?.id != null) {
      corridorId.current = results[2].id;
      newPlaced.corridor = true;
    }
    const loadedZoneIds: FeatureId[] = [];
    for (let i = 0; i < noGoCount; i++) {
      const r = results[3 + i];
      if (r?.id != null) loadedZoneIds.push(r.id);
    }
    setNoGoZoneIds(loadedZoneIds);
    setPlaced(newPlaced);

    const coords = round.corridor.coordinates[0];
    const bounds = coords.reduce(
      (b, c) => b.extend(c as [number, number]),
      new maplibregl.LngLatBounds(
        coords[0] as [number, number],
        coords[0] as [number, number]
      )
    );
    map.fitBounds(bounds, { padding: 80 });
    setRoundName(round.name);
  };

  const saveRound = async () => {
    const draw = drawRef.current;
    if (!draw || !mapId || !gameMap) return;

    const snapshot = draw.getSnapshot();
    const startFeature = startPointId.current != null
      ? snapshot.find((f) => f.id === startPointId.current)
      : null;
    const endFeature = endPointId.current != null
      ? snapshot.find((f) => f.id === endPointId.current)
      : null;
    const corridorFeature = corridorId.current != null
      ? snapshot.find((f) => f.id === corridorId.current)
      : null;

    if (!startFeature || !endFeature || !corridorFeature) {
      setSaveError('Missing features: ensure start point, end point, and corridor are all placed.');
      return;
    }

    // Client-side validation
    const startCoords = (startFeature.geometry as GeoJSON.Point).coordinates;
    const endCoords = (endFeature.geometry as GeoJSON.Point).coordinates;
    if (startCoords[0] === endCoords[0] && startCoords[1] === endCoords[1]) {
      setSaveError('Start and end points must be different locations.');
      return;
    }
    const corridorCoords = (corridorFeature.geometry as GeoJSON.Polygon).coordinates;
    if (!corridorCoords[0] || corridorCoords[0].length < 4) {
      setSaveError('Corridor polygon must have at least 3 vertices.');
      return;
    }

    setSaveError('');
    setSaving(true);
    try {
      const noGoZoneGeometries = noGoZoneIds
        .map((id) => snapshot.find((f) => f.id === id))
        .filter((f): f is NonNullable<typeof f> => f != null)
        .map((f) => f.geometry as GeoJSON.Polygon);

      const roundNumber = editingRound ?? (gameMap.rounds?.length || 0) + 1;
      const data = {
        round_number: roundNumber,
        name: roundName || `Round ${roundNumber}`,
        start_point: startFeature.geometry as GeoJSON.Geometry,
        end_point: endFeature.geometry as GeoJSON.Geometry,
        corridor: corridorFeature.geometry as GeoJSON.Geometry,
        no_go_zones: noGoZoneGeometries,
      };

      if (editingRound !== null && gameMap.rounds) {
        const existingRound = gameMap.rounds.find((r) => r.round_number === editingRound);
        if (existingRound) {
          await api.updateRound(mapId, existingRound.id, data);
        }
      } else {
        await api.createRound(mapId, data);
      }

      const updated = await api.getMap(mapId);
      setGameMap(updated);
      cancelEditing();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save round');
    } finally {
      setSaving(false);
    }
  };

  const startNewRound = () => {
    clearDrawing();
    setEditingRound(null);
    setRoundName('');
    setStep('start');
    if (isMobile) setSidebarOpen(false);
  };

  const editRound = (round: Round) => {
    setEditingRound(round.round_number);
    loadRound(round);
    setStep('review');
  };

  const cancelEditing = () => {
    clearDrawing();
    setEditingRound(null);
    setRoundName('');
    setStep('idle');
  };

  const deleteRound = async (round: Round) => {
    if (!mapId) return;
    await api.deleteRound(mapId, round.id);
    const updated = await api.getMap(mapId);
    setGameMap(updated);
  };

  // Redo a step: remove existing feature so user can re-place it
  const redoStep = (target: 'start' | 'end' | 'corridor') => {
    const draw = drawRef.current;
    if (!draw) return;

    if (target === 'start' && startPointId.current != null) {
      try { draw.removeFeatures([startPointId.current]); } catch { /* noop */ }
      startPointId.current = null;
      setPlaced((p) => ({ ...p, start: false }));
    } else if (target === 'end' && endPointId.current != null) {
      try { draw.removeFeatures([endPointId.current]); } catch { /* noop */ }
      endPointId.current = null;
      setPlaced((p) => ({ ...p, end: false }));
    } else if (target === 'corridor' && corridorId.current != null) {
      try { draw.removeFeatures([corridorId.current]); } catch { /* noop */ }
      corridorId.current = null;
      setPlaced((p) => ({ ...p, corridor: false }));
    }
    setStep(target);
  };

  if (!gameMap) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg text-gray-600">Loading map...</div>
      </div>
    );
  }

  const isEditing = step !== 'idle';
  const allPlaced = placed.start && placed.end && placed.corridor;

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <header className="bg-white shadow-sm border-b px-3 py-2 sm:px-4 sm:py-3 flex items-center justify-between z-10">
        <div className="flex items-center gap-2 sm:gap-4 min-w-0">
          <button
            onClick={() => {
              if (isEditing) {
                cancelEditing();
              } else {
                navigate('/guide');
              }
            }}
            className="text-gray-500 hover:text-gray-700 shrink-0"
          >
            &larr; {isEditing ? 'Cancel' : 'Back'}
          </button>
          <h1 className="text-base sm:text-lg font-bold truncate">{gameMap.name}</h1>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {isEditing && (
            <span className="text-xs sm:text-sm text-gray-600 hidden sm:inline">
              {editingRound !== null ? `Editing Round ${editingRound}` : 'New Round'}
            </span>
          )}
          {isMobile && (
            <button
              onClick={() => setSidebarOpen((v) => !v)}
              className="px-2.5 py-1.5 bg-gray-100 text-gray-700 rounded-lg text-xs font-medium hover:bg-gray-200"
            >
              {sidebarOpen ? 'Map' : 'Panel'}
            </button>
          )}
        </div>
      </header>

      <div className="flex-1 flex flex-col md:flex-row min-h-0">
        {/* Sidebar - full width sheet on mobile, fixed sidebar on desktop */}
        <div className={`${
          isMobile
            ? sidebarOpen ? 'flex' : 'hidden'
            : 'flex'
        } md:w-72 bg-white border-b md:border-b-0 md:border-r overflow-y-auto p-4 flex-col gap-4 z-10 ${
          isMobile && sidebarOpen ? 'flex-1' : ''
        }`}>
          {isEditing ? (
            <>
              {/* Step progress */}
              <div>
                <h3 className="font-semibold text-sm text-gray-700 mb-3">
                  {editingRound !== null ? `Edit Round ${editingRound}` : 'New Round'}
                </h3>
                <div className="flex flex-col gap-1">
                  {(['start', 'end', 'corridor', 'no_go_zone'] as const).map((s) => {
                    const info = STEP_INFO[s];
                    const isCurrent = step === s;
                    const isComplete = s === 'no_go_zone' ? noGoZoneIds.length > 0 : placed[s];

                    return (
                      <div key={s} className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                        isCurrent
                          ? 'bg-blue-50 border border-blue-200 text-blue-800'
                          : isComplete
                            ? 'bg-green-50 text-green-700'
                            : 'text-gray-400'
                      }`}>
                        <span
                          className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ${
                            isCurrent
                              ? 'bg-blue-600 text-white'
                              : isComplete
                                ? 'bg-green-600 text-white'
                                : 'bg-gray-200 text-gray-500'
                          }`}
                        >
                          {isComplete && !isCurrent ? '\u2713' : info.number}
                        </span>
                        <span className="flex-1">{info.label}</span>
                        {isComplete && !isCurrent && s !== 'no_go_zone' && (
                          <button
                            onClick={() => redoStep(s as 'start' | 'end' | 'corridor')}
                            className="text-xs text-green-600 hover:text-green-800"
                          >
                            redo
                          </button>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Current step instruction */}
              {step !== 'review' && step !== 'no_go_zone' && (
                <div className="bg-blue-50 border border-blue-100 rounded-lg p-3">
                  <p className="text-sm text-blue-800">
                    {STEP_INFO[step].instruction}
                  </p>
                </div>
              )}

              {/* No-go zone controls */}
              {step === 'no_go_zone' && (
                <div className="flex flex-col gap-2">
                  <div className="bg-red-50 border border-red-100 rounded-lg p-3">
                    <p className="text-sm text-red-800">
                      {STEP_INFO.no_go_zone.instruction}
                    </p>
                  </div>
                  <button
                    onClick={() => drawRef.current?.setMode('polygon')}
                    className="px-3 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 text-sm font-medium transition-colors"
                  >
                    + Draw Zone
                  </button>
                  {noGoZoneIds.length > 0 && (
                    <div className="flex flex-col gap-1">
                      {noGoZoneIds.map((id, i) => (
                        <div key={String(id)} className="flex items-center justify-between bg-red-50 rounded px-3 py-1.5 text-sm">
                          <span className="text-red-800">Zone {i + 1}</span>
                          <button
                            onClick={() => {
                              try { drawRef.current?.removeFeatures([id]); } catch { /* noop */ }
                              setNoGoZoneIds((prev) => prev.filter((z) => z !== id));
                            }}
                            className="text-red-600 hover:text-red-800 text-xs"
                          >
                            Delete
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                  <button
                    onClick={() => setStep('review')}
                    className="px-3 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm font-medium transition-colors"
                  >
                    {noGoZoneIds.length === 0 ? 'Skip (no zones)' : 'Done'}
                  </button>
                </div>
              )}

              {/* Round name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Round Name
                </label>
                <input
                  type="text"
                  value={roundName}
                  onChange={(e) => setRoundName(e.target.value)}
                  placeholder="e.g., Big Cottonwood"
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* Save error */}
              {saveError && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                  <p className="text-sm text-red-700">{saveError}</p>
                </div>
              )}

              {/* Save button — shown when all features are placed */}
              {allPlaced && (
                <div className="bg-green-50 border border-green-100 rounded-lg p-3">
                  {step === 'review' && (
                    <p className="text-sm text-green-800 mb-3">
                      All set! Click redo on any step to change it, or save when ready.
                    </p>
                  )}
                  <button
                    onClick={saveRound}
                    disabled={saving}
                    className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm disabled:opacity-50 font-semibold transition-colors"
                  >
                    {saving ? 'Saving...' : 'Save Round'}
                  </button>
                </div>
              )}

              {/* Cancel */}
              <button
                onClick={cancelEditing}
                className="px-4 py-2 bg-gray-100 text-gray-600 rounded-lg hover:bg-gray-200 text-sm transition-colors"
              >
                Cancel
              </button>
            </>
          ) : (
            <>
              {/* Idle: show round list and add button */}
              <button
                onClick={startNewRound}
                className="px-4 py-2.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-semibold transition-colors"
              >
                + Add Round
              </button>

              {gameMap.rounds && gameMap.rounds.length > 0 ? (
                <div className="flex flex-col gap-2">
                  <h3 className="font-semibold text-sm text-gray-700">
                    Rounds ({gameMap.rounds.length})
                  </h3>
                  {gameMap.rounds.map((round) => (
                    <div
                      key={round.id}
                      className="bg-gray-50 rounded-lg p-3 text-sm"
                    >
                      <div className="font-medium text-gray-900">
                        #{round.round_number}: {round.name || 'Untitled'}
                      </div>
                      <div className="flex gap-3 mt-2">
                        <button
                          onClick={() => {
                            if (round.corridor?.coordinates?.[0]) {
                              const coords = round.corridor.coordinates[0];
                              const bounds = coords.reduce(
                                (b, c) => b.extend(c as [number, number]),
                                new maplibregl.LngLatBounds(
                                  coords[0] as [number, number],
                                  coords[0] as [number, number]
                                )
                              );
                              mapRef.current?.fitBounds(bounds, { padding: 80 });
                            }
                            if (isMobile) setSidebarOpen(false);
                          }}
                          className="text-gray-600 hover:text-gray-800 text-xs py-1"
                        >
                          View
                        </button>
                        <button
                          onClick={() => {
                            editRound(round);
                            if (isMobile) setSidebarOpen(false);
                          }}
                          className="text-blue-600 hover:text-blue-800 text-xs py-1"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => deleteRound(round)}
                          className="text-red-600 hover:text-red-800 text-xs py-1"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">
                  No rounds yet. Add one to get started.
                </p>
              )}
            </>
          )}
        </div>

        {/* Map */}
        <div className={`flex-1 relative ${isMobile && sidebarOpen ? 'hidden' : ''}`}>
          <GameMapComponent onMapReady={initDraw} terrain3d={terrain3d} slopeShading={slopeShading} />
          <MapOverlayControls
            terrain3d={terrain3d}
            slopeShading={slopeShading}
            onToggleTerrain={() => setTerrain3d((v) => !v)}
            onToggleSlope={() => setSlopeShading((v) => !v)}
          />
        </div>
      </div>
    </div>
  );
}
