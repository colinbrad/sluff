import { useEffect, useRef } from 'react';
import maplibregl from 'maplibre-gl';

interface GameMapProps {
  onMapReady?: (map: maplibregl.Map) => void;
  center?: [number, number];
  zoom?: number;
  className?: string;
}

export default function GameMap({
  onMapReady,
  center = [-111.5, 40.6], // Default: Wasatch Range, Utah
  zoom = 12,
  className = 'w-full h-full',
}: GameMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);

  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    const map = new maplibregl.Map({
      container: containerRef.current,
      style: {
        version: 8,
        sources: {
          'opentopomap': {
            type: 'raster',
            tiles: [
              'https://a.tile.opentopomap.org/{z}/{x}/{y}.png',
              'https://b.tile.opentopomap.org/{z}/{x}/{y}.png',
              'https://c.tile.opentopomap.org/{z}/{x}/{y}.png',
            ],
            tileSize: 256,
            attribution: '&copy; <a href="https://opentopomap.org">OpenTopoMap</a> contributors',
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
      center,
      zoom,
      maxZoom: 17,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');
    map.addControl(new maplibregl.ScaleControl(), 'bottom-left');

    map.on('load', () => {
      onMapReady?.(map);
    });

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return <div ref={containerRef} className={className} />;
}
