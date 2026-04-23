import { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import maplibregl from 'maplibre-gl';
import {
  TerraDraw,
  TerraDrawLineStringMode,
  TerraDrawSelectMode,
} from 'terra-draw';
import { TerraDrawMapLibreGLAdapter } from 'terra-draw-maplibre-gl-adapter';
import type { Round } from '../../types/game';
import type { GameStatePayload, CursorUpdatePayload, DrawingUpdateFromServer, ScoresPayload } from '../../types/messages';
import * as api from '../../services/api';
import { addRoundMarkers, addNoGoZoneLayers, removeNoGoZoneLayers } from '../../utils/mapUtils';
import { usePlayerStore } from '../../stores/playerStore';
import { useGameStore } from '../../stores/gameStore';
import { useGuideStore } from '../../stores/guideStore';
import { GameWebSocket } from '../../services/ws';
import GameMapComponent from '../map/GameMap';
import MapOverlayControls from '../map/MapOverlayControls';
import GameHeader from './GameHeader';
import RoundReview from './RoundReview';
import GuideSessionControls from './GuideSessionControls';

export default function GameView() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const player = usePlayerStore((s) => s.player);
  const guide = useGuideStore((s) => s.guide);
  const {
    session,
    currentRound,
    phase,
    timeRemaining,
    teamScores,
    routeResults,
    setSession,
    setCurrentRound,
    setPhase,
    setTimeRemaining,
    updateCursor,
    setTeamDrawing,
    setTeamScores,
    setRouteResults,
    reset,
  } = useGameStore();

  const [showScores, setShowScores] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [mapReady, setMapReady] = useState(false);
  const [terrain3d, setTerrain3d] = useState(false);
  const [slopeShading, setSlopeShading] = useState(false);

  // Reset game store when entering a new session
  useEffect(() => {
    reset();
  }, [sessionId, reset]);

  const mapRef = useRef<maplibregl.Map | null>(null);
  const currentRoundRef = useRef<typeof currentRound>(currentRound);
  useEffect(() => { currentRoundRef.current = currentRound; }, [currentRound]);
  const drawRef = useRef<TerraDraw | null>(null);
  const wsRef = useRef<GameWebSocket | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const cursorMarkersRef = useRef<Map<string, maplibregl.Marker>>(new Map());
  const markersRef = useRef<maplibregl.Marker[]>([]);

  const isSolo = session?.is_solo ?? false;
  const isGuide = !!guide && session?.guide_id === guide.id;

  // Fetch session on mount
  useEffect(() => {
    if (!sessionId) return;
    api.getSession(sessionId).then((s) => {
      setSession(s);
      if (s.phase === 'playing' && s.current_round > 0) {
        api.getMap(s.map_id).then((m) => {
          const round = m.rounds?.[s.current_round - 1];
          if (round) {
            setCurrentRound(round);
            setPhase('playing');
            setTimeRemaining(s.time_limit_sec);
          }
        });
      }
    });
  }, [sessionId, setSession, setCurrentRound, setPhase, setTimeRemaining]);

  // WebSocket connection (skip for solo)
  useEffect(() => {
    if (!sessionId || !player || isSolo) return;

    const ws = new GameWebSocket(sessionId, player.id);
    ws.connect();
    wsRef.current = ws;

    ws.onMessage((msg) => {
      switch (msg.type) {
        case 'game_state': {
          const payload = msg.payload as GameStatePayload;
          setPhase(payload.phase);
          setTimeRemaining(payload.time_remaining);
          if (payload.phase === 'scoring') {
            setShowScores(true);
          }
          break;
        }
        case 'round_start': {
          const round = msg.payload as Round;
          setCurrentRound(round);
          setPhase('playing');
          setShowScores(false);
          setSubmitted(false);
          break;
        }
        case 'cursor_update': {
          const cursor = msg.payload as CursorUpdatePayload;
          updateCursor(cursor.player_id, cursor.lat, cursor.lng);
          showCursorOnMap(cursor.player_id, cursor.lat, cursor.lng);
          break;
        }
        case 'drawing_update': {
          const drawing = msg.payload as DrawingUpdateFromServer;
          if (drawing.player_id !== player.id) {
            setTeamDrawing(drawing.path);
          }
          break;
        }
        case 'scores': {
          const scores = msg.payload as ScoresPayload;
          setTeamScores(scores.team_scores);
          // Fetch full route data (with paths) for map display
          if (sessionId && currentRound?.id) {
            api.getScores(sessionId, currentRound.id).then((routes) => {
              setRouteResults(routes);
            });
          }
          setShowScores(true);
          break;
        }
        case 'round_end':
          handleAutoSubmit();
          break;
      }
    });

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [sessionId, player, isSolo]); // eslint-disable-line react-hooks/exhaustive-deps

  // Countdown timer
  useEffect(() => {
    if (phase !== 'playing' || timeRemaining <= 0) return;

    timerRef.current = setInterval(() => {
      setTimeRemaining(Math.max(0, timeRemaining - 1));
    }, 1000);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [phase, timeRemaining, setTimeRemaining]);

  // Reset mapReady when the score screen hides the map, so the flag flips
  // false → true when the map remounts for the next round, re-triggering fitBounds.
  useEffect(() => {
    if (showScores) setMapReady(false);
  }, [showScores]);

  const noGoLayerIdsRef = useRef<string[]>([]);

  // Show round on map whenever currentRound changes AND map is ready.
  // This is the single source of truth for rendering round markers/corridor,
  // eliminating race conditions between map load and data arrival.
  useEffect(() => {
    if (!currentRound || !mapReady) return;
    const map = mapRef.current;
    if (!map) return;

    // Remove previous markers
    for (const m of markersRef.current) {
      m.remove();
    }
    markersRef.current = [];

    // Remove previous no-go zone layers/sources
    removeNoGoZoneLayers(map, noGoLayerIdsRef.current);
    noGoLayerIdsRef.current = [];

    // Add no-go zone layers
    if (currentRound.no_go_zones?.length) {
      noGoLayerIdsRef.current = addNoGoZoneLayers(map, currentRound.no_go_zones, 'nogo-zone');
    }

    // Add start/end markers
    markersRef.current.push(...addRoundMarkers(map, currentRound));

    // Fit map to start and end points
    if (currentRound.start_point?.coordinates && currentRound.end_point?.coordinates) {
      const bounds = new maplibregl.LngLatBounds();
      bounds.extend(currentRound.start_point.coordinates as [number, number]);
      bounds.extend(currentRound.end_point.coordinates as [number, number]);
      map.fitBounds(bounds, { padding: 120, maxZoom: 16, animate: false });
    }
  }, [currentRound, mapReady]);

  const showCursorOnMap = (playerId: string, lat: number, lng: number) => {
    const map = mapRef.current;
    if (!map) return;

    let marker = cursorMarkersRef.current.get(playerId);
    if (!marker) {
      const el = document.createElement('div');
      el.className = 'w-3 h-3 rounded-full border-2 border-white shadow-md';
      el.style.backgroundColor = '#F59E0B';
      marker = new maplibregl.Marker({ element: el }).setLngLat([lng, lat]).addTo(map);
      cursorMarkersRef.current.set(playerId, marker);
    } else {
      marker.setLngLat([lng, lat]);
    }
  };

  const initDraw = useCallback(
    (map: maplibregl.Map) => {
      mapRef.current = map;
      setMapReady(true);

      // Fit to round bounds immediately on map load (handles first-round case
      // where the useEffect may fire before mapRef.current is set).
      const cr = currentRoundRef.current;
      if (cr?.start_point?.coordinates && cr?.end_point?.coordinates) {
        const bounds = new maplibregl.LngLatBounds();
        bounds.extend(cr.start_point.coordinates as [number, number]);
        bounds.extend(cr.end_point.coordinates as [number, number]);
        map.fitBounds(bounds, { padding: 120, maxZoom: 16, animate: false });
      }

      const draw = new TerraDraw({
        adapter: new TerraDrawMapLibreGLAdapter({ map }),
        modes: [
          new TerraDrawLineStringMode({
            keyEvents: { finish: 'Enter', cancel: 'Escape' },
          }),
          new TerraDrawSelectMode({
            flags: {
              linestring: {
                feature: { draggable: true, coordinates: { draggable: true, deletable: true } },
              },
            },
          }),
        ],
      });

      draw.start();
      draw.setMode('linestring');
      drawRef.current = draw;

      // After finishing a line (Enter, double-click, etc.) switch to select so
      // the user can review/adjust without accidentally starting a second line.
      draw.on('finish', () => {
        draw.setMode('select');
      });

      // Multiplayer: send cursor position and broadcast drawing updates
      if (!isSolo) {
        map.on('mousemove', (e) => {
          wsRef.current?.send('cursor_move', {
            lat: e.lngLat.lat,
            lng: e.lngLat.lng,
          });
        });

        draw.on('change', () => {
          const snapshot = draw.getSnapshot();
          const lineFeatures = snapshot.filter((f) => f.geometry.type === 'LineString');
          if (lineFeatures.length > 0) {
            const latest = lineFeatures[lineFeatures.length - 1];
            wsRef.current?.send('drawing_update', {
              team_id: player?.team_id || '',
              path: latest.geometry,
            });
          }
        });
      }
    },
    [player, isSolo]
  );

  const handleSubmit = async () => {
    if (!sessionId || !currentRound || !player || submitted) return;

    const draw = drawRef.current;
    if (!draw) return;

    const snapshot = draw.getSnapshot();
    const lineFeatures = snapshot.filter((f) => f.geometry.type === 'LineString');
    if (lineFeatures.length === 0) return;

    const latestLine = lineFeatures[lineFeatures.length - 1];

    setSubmitting(true);
    try {
      await api.submitRoute(
        sessionId,
        currentRound.id,
        player.team_id,
        latestLine.geometry as GeoJSON.Geometry
      );
      setSubmitted(true);

      // Solo: immediately fetch and show scores
      if (isSolo) {
        const scores = await api.getScores(sessionId, currentRound.id);
        setRouteResults(scores);
        const mappedScores = scores.map((r) => ({
          team_id: r.team_id,
          score: r.details!,
        }));
        setTeamScores(mappedScores);
        setShowScores(true);
      }
    } catch {
      // Submit errors are non-fatal; player can retry
    } finally {
      setSubmitting(false);
    }
  };

  const handleAutoSubmit = () => {
    if (!submitted) {
      handleSubmit();
    }
  };

  const handleScoreContinue = async () => {
    if (isSolo && sessionId) {
      try {
        const updated = await api.startGame(sessionId);
        setSession(updated);
        if (updated.phase === 'finished') {
          setPhase('finished');
          setShowScores(false);
          return;
        }
        const m = await api.getMap(updated.map_id);
        const round = m.rounds?.[updated.current_round - 1];
        if (round) {
          setCurrentRound(round);
          setPhase('playing');
          setTimeRemaining(updated.time_limit_sec);
          setShowScores(false);
          setSubmitted(false);
          setRouteResults([]);
          // Clear previous drawing
          const draw = drawRef.current;
          if (draw) {
            draw.clear();
            draw.setMode('linestring');
          }
          // Markers/corridor will be shown by the useEffect on currentRound
        }
      } catch {
        // Round advance errors are non-fatal
      }
    } else {
      setShowScores(false);
    }
  };

  // Game over screen
  if (phase === 'finished') {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center p-4">
        <div className="text-center">
          <h1 className="text-3xl sm:text-4xl font-bold text-white mb-4">Game Over</h1>
          <p className="text-gray-400 mb-8">Thanks for playing!</p>
          <button
            onClick={() => navigate('/')}
            className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold transition-colors"
          >
            Back to Home
          </button>
        </div>
      </div>
    );
  }

  if (showScores && teamScores.length > 0) {
    return (
      <RoundReview
        teamScores={teamScores}
        routeResults={routeResults}
        teams={session?.teams || []}
        currentRound={currentRound}
        onNextRound={handleScoreContinue}
      />
    );
  }

  return (
    <div className="h-screen flex flex-col">
      <GameHeader
        roundNumber={currentRound?.round_number || 0}
        roundName={currentRound?.name || ''}
        timeRemaining={timeRemaining}
        phase={phase || 'waiting'}
        submitted={submitted}
        onExit={() => {
          if (isSolo) {
            navigate('/');
          } else {
            navigate(`/session/${sessionId}/lobby`);
          }
        }}
      />

      <div className="flex-1 relative">
        <GameMapComponent onMapReady={initDraw} terrain3d={terrain3d} slopeShading={slopeShading} />
        {!currentRound && (
          <div className="absolute inset-0 flex items-center justify-center bg-gray-900 z-10">
            <div className="text-gray-400">Loading round...</div>
          </div>
        )}
        <MapOverlayControls
          terrain3d={terrain3d}
          slopeShading={slopeShading}
          onToggleTerrain={() => setTerrain3d((v) => !v)}
          onToggleSlope={() => setSlopeShading((v) => !v)}
        />

        {isGuide && sessionId && (
          <GuideSessionControls
            sessionId={sessionId}
            players={session?.players ?? []}
            teams={session?.teams ?? []}
            routeResults={routeResults}
            currentRound={currentRound}
            phase={phase ?? 'waiting'}
            onAdvanceRound={handleScoreContinue}
            onRefreshSession={() => {
              api.getSession(sessionId).then((s) => setSession(s));
            }}
          />
        )}

        {/* Submit button */}
        {phase === 'playing' && !submitted && (
          <div className="absolute bottom-4 sm:bottom-6 left-1/2 -translate-x-1/2 z-10 w-full px-4 sm:w-auto sm:px-0">
            <button
              onClick={handleSubmit}
              disabled={submitting}
              className="w-full sm:w-auto px-8 py-3 bg-green-600 text-white rounded-lg shadow-lg hover:bg-green-700 disabled:opacity-50 font-semibold text-base sm:text-lg transition-colors"
            >
              {submitting ? 'Submitting...' : 'Submit Route'}
            </button>
          </div>
        )}

        {submitted && phase === 'playing' && !isSolo && (
          <div className="absolute bottom-4 sm:bottom-6 left-1/2 -translate-x-1/2 z-10">
            <div className="px-4 sm:px-6 py-3 bg-white rounded-lg shadow-lg text-green-600 font-semibold text-sm sm:text-base text-center">
              Submitted! Waiting for other teams...
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
