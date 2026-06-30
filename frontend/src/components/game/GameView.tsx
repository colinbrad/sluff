import { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import maplibregl from 'maplibre-gl';
import { TerraDraw, TerraDrawLineStringMode, TerraDrawSelectMode } from 'terra-draw';
import { TerraDrawMapLibreGLAdapter } from 'terra-draw-maplibre-gl-adapter';
import type { Round } from '../../types/game';
import type {
  GameStatePayload,
  CursorUpdatePayload,
  DrawingUpdateFromServer,
  ScoresPayload,
  TeamSubmittedPayload,
} from '../../types/messages';
import * as api from '../../services/api';
import { addRoundMarkers, addNoGoZoneLayers, removeNoGoZoneLayers } from '../../utils/mapUtils';
import { toCoord } from '../../utils/geojson';
import { usePlayerStore } from '../../stores/playerStore';
import { useGameStore } from '../../stores/gameStore';
import { useGuideStore } from '../../stores/guideStore';
import { GameWebSocket } from '../../services/ws';
import GameMapComponent from '../map/GameMap';
import MapOverlayControls from '../map/MapOverlayControls';
import GameHeader from './GameHeader';
import RoundReview from './RoundReview';
import GuideSessionControls from './GuideSessionControls';
import DemoTutorial, { type DemoStep } from '../demo/DemoTutorial';

export default function GameView() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const isDemo = searchParams.get('demo') === 'true';
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
  const [demoStep, setDemoStep] = useState<DemoStep>('welcome');
  const [hasRoute, setHasRoute] = useState(false);
  const [submittedTeams, setSubmittedTeams] = useState<Set<string>>(new Set());

  // Reset game store when entering a new session
  useEffect(() => {
    reset();
  }, [sessionId, reset]);

  const mapRef = useRef<maplibregl.Map | null>(null);
  const currentRoundRef = useRef<typeof currentRound>(currentRound);
  useEffect(() => {
    currentRoundRef.current = currentRound;
  }, [currentRound]);
  const drawRef = useRef<TerraDraw | null>(null);
  const wsRef = useRef<GameWebSocket | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const cursorMarkersRef = useRef<Map<string, maplibregl.Marker>>(new Map());
  const markersRef = useRef<maplibregl.Marker[]>([]);

  const isSolo = session?.is_solo ?? false;
  const isGuide = !!guide && session?.guide_id === guide.id;
  const teamsWithPlayers = new Set(
    (session?.players ?? []).filter((p) => p.team_id).map((p) => p.team_id),
  ).size;

  useEffect(() => {
    if (isDemo && hasRoute && demoStep === 'drawing') {
      setDemoStep('ready');
    }
  }, [isDemo, hasRoute, demoStep]);

  useEffect(() => {
    if (!sessionId) return;
    api.getSession(sessionId).then((s) => {
      setSession(s);
      if (s.phase === 'playing' && s.current_round > 0) {
        if (isDemo) {
          api.getCurrentRound(sessionId).then((round) => {
            setCurrentRound(round);
            setPhase('playing');
            setTimeRemaining(s.time_limit_sec);
          });
        } else {
          api.getMap(s.map_id).then((m) => {
            const round = m.rounds?.[s.current_round - 1];
            if (round) {
              setCurrentRound(round);
              setPhase('playing');
              setTimeRemaining(s.time_limit_sec);
            }
          });
        }
      } else if (s.phase === 'scoring' && s.current_round > 0) {
        // Reconnecting mid-scoring: restore the round and its results.
        api.getMap(s.map_id).then((m) => {
          const round = m.rounds?.[s.current_round - 1];
          if (!round) return;
          setCurrentRound(round);
          setPhase('scoring');
          api.getScores(sessionId, round.id).then((routes) => {
            setRouteResults(routes);
            setTeamScores(
              routes.flatMap((r) => (r.details ? [{ team_id: r.team_id, score: r.details }] : [])),
            );
            setShowScores(true);
          });
        });
      }
    });
  }, [
    sessionId,
    isDemo,
    setSession,
    setCurrentRound,
    setPhase,
    setTimeRemaining,
    setRouteResults,
    setTeamScores,
  ]);

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
          setHasRoute(false);
          setTeamScores([]);
          setRouteResults([]);
          setSubmittedTeams(new Set());
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
          // The review screen is opened by game_state(scoring); this just
          // fills in the (progressively revealed) results.
          const scores = msg.payload as ScoresPayload;
          setTeamScores(scores.team_scores);
          // Fetch full route data (with paths) for map display. Read the round
          // via ref since this closure was created at connect time.
          const round = currentRoundRef.current;
          if (sessionId && round?.id) {
            api.getScores(sessionId, round.id).then((routes) => {
              setRouteResults(routes);
            });
          }
          break;
        }
        case 'team_submitted': {
          const { team_id } = msg.payload as TeamSubmittedPayload;
          setSubmittedTeams((prev) => new Set(prev).add(team_id));
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

    for (const m of markersRef.current) {
      m.remove();
    }
    markersRef.current = [];

    removeNoGoZoneLayers(map, noGoLayerIdsRef.current);
    noGoLayerIdsRef.current = [];

    if (currentRound.no_go_zones?.length) {
      noGoLayerIdsRef.current = addNoGoZoneLayers(map, currentRound.no_go_zones, 'nogo-zone');
    }

    markersRef.current.push(...addRoundMarkers(map, currentRound));

    if (currentRound.start_point?.coordinates && currentRound.end_point?.coordinates) {
      const bounds = new maplibregl.LngLatBounds();
      bounds.extend(toCoord(currentRound.start_point.coordinates));
      bounds.extend(toCoord(currentRound.end_point.coordinates));
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
        bounds.extend(toCoord(cr.start_point.coordinates));
        bounds.extend(toCoord(cr.end_point.coordinates));
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
        if (isDemo) setHasRoute(true);
      });

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
          const latest = lineFeatures[lineFeatures.length - 1];
          if (latest) {
            wsRef.current?.send('drawing_update', {
              team_id: player?.team_id || '',
              path: latest.geometry,
            });
          }
        });
      }
    },
    [player, isSolo, isDemo],
  );

  const handleSubmit = async () => {
    // Read the round through the ref so the WS auto-submit closure (which
    // captured an earlier render) still sees the active round.
    const round = currentRoundRef.current;
    if (!sessionId || !round || !player || submitted) return;

    const draw = drawRef.current;
    if (!draw) return;

    const snapshot = draw.getSnapshot();
    const lineFeatures = snapshot.filter((f) => f.geometry.type === 'LineString');
    const latestLine = lineFeatures[lineFeatures.length - 1];
    if (!latestLine) return;

    setSubmitting(true);
    try {
      await api.submitRoute(
        sessionId,
        round.id,
        player.team_id,
        latestLine.geometry as GeoJSON.Geometry,
      );
      setSubmitted(true);

      if (isSolo) {
        const scores = await api.getScores(sessionId, round.id);
        setRouteResults(scores);
        const mappedScores = scores.flatMap((r) =>
          r.details ? [{ team_id: r.team_id, score: r.details }] : [],
        );
        setTeamScores(mappedScores);
        setShowScores(true);
      }
    } catch {
      // Submission failed; UI shows the unsubmitted state, user can retry.
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
        let updatedSession;
        let nextRound;

        if (isDemo) {
          const { session: s, round: r } = await api.demoNextRound(sessionId);
          updatedSession = s;
          nextRound = r;
        } else {
          updatedSession = await api.startGame(sessionId);
          if (updatedSession.phase !== 'finished') {
            const m = await api.getMap(updatedSession.map_id);
            nextRound = m.rounds?.[updatedSession.current_round - 1] ?? null;
          }
        }

        setSession(updatedSession);
        if (updatedSession.phase === 'finished') {
          setPhase('finished');
          setShowScores(false);
          return;
        }
        if (nextRound) {
          setCurrentRound(nextRound);
          setPhase('playing');
          setTimeRemaining(updatedSession.time_limit_sec);
          setShowScores(false);
          setSubmitted(false);
          setRouteResults([]);
          setHasRoute(false);
          setDemoStep('drawing');
          const draw = drawRef.current;
          if (draw) {
            draw.clear();
            draw.setMode('linestring');
          }
        }
      } catch {
        // Round advance errors are non-fatal
      }
    } else if (isGuide && sessionId) {
      // Multiplayer: only the guide advances. The round_start (or finished)
      // broadcast moves every client, including the guide, to the next round.
      try {
        await api.startGame(sessionId);
      } catch {
        // non-fatal
      }
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
        showAdvance={isSolo || isGuide}
      />
    );
  }

  return (
    <div className="h-screen flex flex-col">
      {isDemo && demoStep === 'welcome' && (
        <DemoTutorial step="welcome" onDismissWelcome={() => setDemoStep('drawing')} />
      )}
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
        {isDemo && demoStep !== 'welcome' && (
          <DemoTutorial step={demoStep} onDismissWelcome={() => setDemoStep('drawing')} />
        )}
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

        {isGuide && !isSolo && sessionId && (
          <GuideSessionControls
            sessionId={sessionId}
            players={session?.players ?? []}
            teams={session?.teams ?? []}
            routeResults={routeResults}
            currentRound={currentRound}
            phase={phase ?? 'waiting'}
            onEndRound={() => api.endRound(sessionId)}
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
              Submitted! {submittedTeams.size}/{teamsWithPlayers} teams in.
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
