import { useEffect, useRef } from 'react';
import maplibregl from 'maplibre-gl';

interface GameMapProps {
  onMapReady?: (map: maplibregl.Map) => void;
  center?: [number, number];
  zoom?: number;
  className?: string;
  terrain3d?: boolean;
  slopeShading?: boolean;
}

export default function GameMap({
  onMapReady,
  center = [-111.5, 40.6], // Default: Wasatch Range, Utah
  zoom = 12,
  className = 'w-full h-full',
  terrain3d = false,
  slopeShading = false,
}: GameMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const mapLoadedRef = useRef(false);

  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    const map = new maplibregl.Map({
      container: containerRef.current,
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
            attribution: '&copy; <a href="https://opentopomap.org">OpenTopoMap</a> contributors',
          },
          'terrain-dem': {
            type: 'raster-dem',
            tiles: ['https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png'],
            tileSize: 256,
            encoding: 'terrarium',
            maxzoom: 15,
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
          {
            id: 'slope-hillshade',
            type: 'hillshade',
            source: 'terrain-dem',
            layout: { visibility: 'none' },
            paint: {
              'hillshade-exaggeration': 0.5,
              'hillshade-shadow-color': '#e63946',
              'hillshade-highlight-color': 'rgba(255,255,255,0)',
              'hillshade-accent-color': '#c1121f',
            },
          },
        ],
      },
      center,
      zoom,
      maxZoom: 17,
      maxPitch: 85,
    });

    map.addControl(new maplibregl.NavigationControl({ visualizePitch: true }), 'top-right');
    map.addControl(new maplibregl.ScaleControl(), 'bottom-left');

    map.on('load', () => {
      mapLoadedRef.current = true;
      onMapReady?.(map);
    });

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
      mapLoadedRef.current = false;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Toggle 3D terrain
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !mapLoadedRef.current) return;

    if (terrain3d) {
      map.setTerrain({ source: 'terrain-dem', exaggeration: 1 });
      map.easeTo({ pitch: 60, duration: 500 });
    } else {
      map.setTerrain(null);
      map.easeTo({ pitch: 0, duration: 500 });
    }
  }, [terrain3d]);

  // Toggle slope hillshade
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !mapLoadedRef.current) return;

    map.setLayoutProperty('slope-hillshade', 'visibility', slopeShading ? 'visible' : 'none');
  }, [slopeShading]);

  return <div ref={containerRef} className={className} />;
}
