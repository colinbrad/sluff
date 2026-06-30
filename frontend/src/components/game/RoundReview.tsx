import { useEffect, useRef, useState } from 'react';
import maplibregl from 'maplibre-gl';
import type { Team, TeamRoute, ScoreDetails, Round } from '../../types/game';
import { TEAM_COLORS } from '../../constants';
import { addRoundMarkers, addNoGoZoneLayers } from '../../utils/mapUtils';
import { toCoord } from '../../utils/geojson';
import GameMapComponent from '../map/GameMap';

interface RoundReviewProps {
  teamScores: Array<{ team_id: string; score: ScoreDetails }>;
  routeResults: TeamRoute[];
  teams: Team[];
  currentRound: Round | null;
  onNextRound: () => void;
  showAdvance: boolean;
}

export default function RoundReview({
  teamScores,
  routeResults,
  teams,
  currentRound,
  onNextRound,
  showAdvance,
}: RoundReviewProps) {
  const mapRef = useRef<maplibregl.Map | null>(null);
  const [mapLoaded, setMapLoaded] = useState(false);

  const sorted = [...teamScores].sort((a, b) => b.score.final_score - a.score.final_score);

  const getTeam = (teamId: string) => teams.find((t) => t.id === teamId);

  const getTeamColor = (teamId: string, index: number): string => {
    const team = getTeam(teamId);
    return team?.color || TEAM_COLORS[index % TEAM_COLORS.length] || '#3B82F6';
  };

  // Add routes, markers, and no-go zones once the map is ready
  useEffect(() => {
    if (!mapLoaded || !mapRef.current) return;
    const map = mapRef.current;

    const bounds = new maplibregl.LngLatBounds();
    let hasBounds = false;

    if (currentRound?.no_go_zones?.length) {
      addNoGoZoneLayers(map, currentRound.no_go_zones, 'review-nogo');
    }

    if (currentRound) {
      const markers = addRoundMarkers(map, currentRound);
      if (currentRound.start_point?.coordinates) {
        bounds.extend(toCoord(currentRound.start_point.coordinates));
        hasBounds = true;
      }
      if (currentRound.end_point?.coordinates) {
        bounds.extend(toCoord(currentRound.end_point.coordinates));
        hasBounds = true;
      }
      // Markers don't need cleanup — they're removed with the map on unmount
      void markers;
    }

    routeResults.forEach((route, index) => {
      let geojson: GeoJSON.LineString;
      try {
        geojson = typeof route.path === 'string' ? JSON.parse(route.path) : route.path;
      } catch {
        return;
      }
      if (geojson.type !== 'LineString' || !geojson.coordinates?.length) return;

      const color = getTeamColor(route.team_id, index);
      const sourceId = `route-${route.id}`;
      const layerId = `route-layer-${route.id}`;

      map.addSource(sourceId, {
        type: 'geojson',
        data: { type: 'Feature', properties: {}, geometry: geojson },
      });
      map.addLayer({
        id: `${layerId}-outline`,
        type: 'line',
        source: sourceId,
        paint: { 'line-color': '#000000', 'line-width': 6, 'line-opacity': 0.3 },
      });
      map.addLayer({
        id: layerId,
        type: 'line',
        source: sourceId,
        paint: { 'line-color': color, 'line-width': 4, 'line-opacity': 0.9 },
      });

      for (const coord of geojson.coordinates) {
        bounds.extend(toCoord(coord));
        hasBounds = true;
      }
    });

    if (hasBounds) {
      map.fitBounds(bounds, { padding: 60, maxZoom: 15 });
    }
  }, [mapLoaded, routeResults, currentRound]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="h-screen flex flex-col bg-gray-900">
      {/* Header */}
      <div className="px-3 py-2 sm:px-4 sm:py-3 bg-gray-800 border-b border-gray-700 flex items-center justify-between">
        <h1 className="text-lg sm:text-xl font-bold text-white">Round Results</h1>
        {showAdvance ? (
          <button
            onClick={onNextRound}
            className="px-4 py-2 sm:px-6 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold transition-colors text-sm sm:text-base"
          >
            Next Round
          </button>
        ) : (
          <span className="text-sm text-gray-400">Waiting for the guide…</span>
        )}
      </div>

      {/* Main content: stacked on mobile, side-by-side on desktop */}
      <div className="flex-1 flex flex-col md:flex-row min-h-0">
        {/* Map */}
        <div className="flex-1 relative min-h-[40vh] md:min-h-0">
          <GameMapComponent
            onMapReady={(map) => {
              mapRef.current = map;
              setMapLoaded(true);
            }}
          />

          {/* Route legend overlay */}
          <div className="absolute bottom-2 left-2 sm:bottom-4 sm:left-4 bg-gray-900/90 backdrop-blur-sm rounded-lg p-2 sm:p-3 space-y-1 sm:space-y-1.5">
            {sorted.map((entry, index) => {
              const team = getTeam(entry.team_id);
              const color = getTeamColor(entry.team_id, index);
              return (
                <div key={entry.team_id} className="flex items-center gap-2">
                  <div
                    className="w-5 sm:w-6 h-1 rounded-full shrink-0"
                    style={{ backgroundColor: color }}
                  />
                  <span className="text-white text-xs sm:text-sm">{team?.name || 'Unknown'}</span>
                  <span className="text-gray-400 text-xs sm:text-sm">
                    {Math.round(entry.score.final_score)}pts
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Score panel - scrollable below map on mobile, side panel on desktop */}
        <div className="md:w-96 bg-gray-900 border-t md:border-t-0 md:border-l border-gray-700 overflow-y-auto p-3 sm:p-4">
          <div className="space-y-3">
            {sorted.map((entry, index) => {
              const team = getTeam(entry.team_id);
              const color = getTeamColor(entry.team_id, index);
              return (
                <div key={entry.team_id} className="bg-gray-800 rounded-lg p-3 sm:p-4">
                  <div className="flex items-center justify-between mb-2 sm:mb-3">
                    <div className="flex items-center gap-2">
                      <span
                        className={`text-xl sm:text-2xl font-bold ${
                          index === 0
                            ? 'text-yellow-400'
                            : index === 1
                              ? 'text-gray-400'
                              : 'text-orange-700'
                        }`}
                      >
                        #{index + 1}
                      </span>
                      <div
                        className="w-3 h-3 rounded-full shrink-0"
                        style={{ backgroundColor: color }}
                      />
                      <span className="text-white font-semibold text-sm sm:text-base">
                        {team?.name || 'Unknown'}
                      </span>
                    </div>
                    <div className="text-right">
                      <span className="text-xl sm:text-2xl font-bold text-white">
                        {Math.round(entry.score.final_score)}
                      </span>
                      <span className="text-gray-400 text-xs sm:text-sm"> / 1000</span>
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-2 sm:gap-3 text-xs sm:text-sm">
                    <div>
                      <span className="text-gray-400">In corridor</span>
                      <div className="text-white font-medium">
                        {entry.score.percent_in_corridor.toFixed(1)}%
                      </div>
                    </div>
                    <div>
                      <span className="text-gray-400">Length</span>
                      <div className="text-white font-medium">
                        {entry.score.route_length_km.toFixed(2)} km
                      </div>
                    </div>
                    <div>
                      <span className="text-gray-400">Max dev.</span>
                      <div className="text-white font-medium">
                        {entry.score.max_deviation_m.toFixed(0)} m
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2 mt-2">
                    {entry.score.connects_start && (
                      <span className="text-xs px-2 py-0.5 bg-green-900 text-green-300 rounded">
                        Start
                      </span>
                    )}
                    {entry.score.connects_end && (
                      <span className="text-xs px-2 py-0.5 bg-green-900 text-green-300 rounded">
                        End
                      </span>
                    )}
                    {(entry.score.no_go_zone_penalty ?? 0) > 0 && (
                      <span className="text-xs px-2 py-0.5 bg-red-900 text-red-300 rounded">
                        No-go: -{(entry.score.no_go_zone_penalty ?? 0).toFixed(0)}pts
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Mobile-only bottom button (easier to reach than header) */}
          {showAdvance && (
            <button
              onClick={onNextRound}
              className="md:hidden w-full mt-4 px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold transition-colors"
            >
              Next Round
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
