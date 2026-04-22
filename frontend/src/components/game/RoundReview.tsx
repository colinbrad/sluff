import { useEffect, useRef, useState } from 'react';
import maplibregl from 'maplibre-gl';
import type { Team, TeamRoute, ScoreDetails, Round } from '../../types/game';

interface RoundReviewProps {
  teamScores: Array<{ team_id: string; score: ScoreDetails }>;
  routeResults: TeamRoute[];
  teams: Team[];
  currentRound: Round | null;
  onNextRound: () => void;
}

const TEAM_COLORS_FALLBACK = ['#3B82F6', '#EF4444', '#10B981', '#F59E0B', '#8B5CF6', '#EC4899'];

export default function RoundReview({
  teamScores,
  routeResults,
  teams,
  currentRound,
  onNextRound,
}: RoundReviewProps) {
  const mapContainerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const [mapLoaded, setMapLoaded] = useState(false);

  const sorted = [...teamScores].sort(
    (a, b) => b.score.final_score - a.score.final_score
  );

  const getTeam = (teamId: string) => teams.find((t) => t.id === teamId);

  const getTeamColor = (teamId: string, index: number) => {
    const team = getTeam(teamId);
    return team?.color || TEAM_COLORS_FALLBACK[index % TEAM_COLORS_FALLBACK.length];
  };

  // Initialize map
  useEffect(() => {
    if (!mapContainerRef.current || mapRef.current) return;

    const map = new maplibregl.Map({
      container: mapContainerRef.current,
      style: {
        version: 8,
        sources: {
          opentopomap: {
            type: 'raster',
            tiles: [
              'https://a.tile.opentopomap.org/{z}/{x}/{y}.png',
              'https://b.tile.opentopomap.org/{z}/{x}/{y}.png',
              'https://c.tile.opentopomap.org/{z}/{x}/{y}.png',
            ],
            tileSize: 256,
            attribution:
              '&copy; <a href="https://opentopomap.org">OpenTopoMap</a> contributors',
          },
        },
        layers: [
          {
            id: 'opentopomap',
            type: 'raster',
            source: 'opentopomap',
            minzoom: 0,
            maxzoom: 17,
          },
        ],
      },
      center: [-111.5, 40.6],
      zoom: 12,
      maxZoom: 17,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');

    map.on('load', () => {
      setMapLoaded(true);
    });

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
      setMapLoaded(false);
    };
  }, []);

  // Add routes to map once loaded
  useEffect(() => {
    if (!mapLoaded || !mapRef.current) return;
    const map = mapRef.current;

    const bounds = new maplibregl.LngLatBounds();
    let hasBounds = false;

    // Add no-go zones
    for (let i = 0; i < (currentRound?.no_go_zones?.length ?? 0); i++) {
      const zone = currentRound!.no_go_zones![i];
      const srcId = `review-nogo-${i}`;
      map.addSource(srcId, {
        type: 'geojson',
        data: { type: 'Feature', geometry: zone, properties: {} },
      });
      map.addLayer({
        id: srcId + '-fill',
        type: 'fill',
        source: srcId,
        paint: { 'fill-color': '#EF4444', 'fill-opacity': 0.25 },
      });
      map.addLayer({
        id: srcId + '-outline',
        type: 'line',
        source: srcId,
        paint: { 'line-color': '#EF4444', 'line-width': 2, 'line-dasharray': [3, 2] },
      });
    }

    // Add start/end markers from the round
    if (currentRound?.start_point?.coordinates) {
      new maplibregl.Marker({ color: '#10B981' })
        .setLngLat(currentRound.start_point.coordinates as [number, number])
        .setPopup(new maplibregl.Popup().setText('Start'))
        .addTo(map);
      bounds.extend(currentRound.start_point.coordinates as [number, number]);
      hasBounds = true;
    }
    if (currentRound?.end_point?.coordinates) {
      new maplibregl.Marker({ color: '#EF4444' })
        .setLngLat(currentRound.end_point.coordinates as [number, number])
        .setPopup(new maplibregl.Popup().setText('End'))
        .addTo(map);
      bounds.extend(currentRound.end_point.coordinates as [number, number]);
      hasBounds = true;
    }

    // Add each team's route as a line layer
    routeResults.forEach((route, index) => {
      let geojson: GeoJSON.LineString;
      try {
        geojson =
          typeof route.path === 'string'
            ? JSON.parse(route.path)
            : route.path;
      } catch {
        return;
      }

      if (geojson.type !== 'LineString' || !geojson.coordinates?.length) return;

      const color = getTeamColor(route.team_id, index);
      const sourceId = `route-${route.id}`;
      const layerId = `route-layer-${route.id}`;

      map.addSource(sourceId, {
        type: 'geojson',
        data: {
          type: 'Feature',
          properties: {},
          geometry: geojson,
        },
      });

      // Outline for contrast
      map.addLayer({
        id: `${layerId}-outline`,
        type: 'line',
        source: sourceId,
        paint: {
          'line-color': '#000000',
          'line-width': 6,
          'line-opacity': 0.3,
        },
      });

      map.addLayer({
        id: layerId,
        type: 'line',
        source: sourceId,
        paint: {
          'line-color': color,
          'line-width': 4,
          'line-opacity': 0.9,
        },
      });

      // Extend bounds to include route coordinates
      for (const coord of geojson.coordinates) {
        bounds.extend(coord as [number, number]);
        hasBounds = true;
      }
    });

    // Fit map to show all routes
    if (hasBounds) {
      map.fitBounds(bounds, { padding: 60, maxZoom: 15 });
    }
  }, [mapLoaded, routeResults, currentRound]);

  return (
    <div className="h-screen flex flex-col bg-gray-900">
      {/* Header */}
      <div className="px-3 py-2 sm:px-4 sm:py-3 bg-gray-800 border-b border-gray-700 flex items-center justify-between">
        <h1 className="text-lg sm:text-xl font-bold text-white">Round Results</h1>
        <button
          onClick={onNextRound}
          className="px-4 py-2 sm:px-6 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold transition-colors text-sm sm:text-base"
        >
          Next Round
        </button>
      </div>

      {/* Main content: stacked on mobile, side-by-side on desktop */}
      <div className="flex-1 flex flex-col md:flex-row min-h-0">
        {/* Map */}
        <div className="flex-1 relative min-h-[40vh] md:min-h-0">
          <div ref={mapContainerRef} className="w-full h-full" />

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
                  <span className="text-white text-xs sm:text-sm">
                    {team?.name || 'Unknown'}
                  </span>
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
                <div
                  key={entry.team_id}
                  className="bg-gray-800 rounded-lg p-3 sm:p-4"
                >
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
                        No-go: -{entry.score.no_go_zone_penalty!.toFixed(0)}pts
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Mobile-only bottom button (easier to reach than header) */}
          <button
            onClick={onNextRound}
            className="md:hidden w-full mt-4 px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold transition-colors"
          >
            Next Round
          </button>
        </div>
      </div>
    </div>
  );
}
