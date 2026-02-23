interface MapOverlayControlsProps {
  terrain3d: boolean;
  slopeShading: boolean;
  onToggleTerrain: () => void;
  onToggleSlope: () => void;
}

export default function MapOverlayControls({
  terrain3d,
  slopeShading,
  onToggleTerrain,
  onToggleSlope,
}: MapOverlayControlsProps) {
  return (
    <div className="absolute top-2 left-2 z-10 flex flex-col gap-1">
      <button
        onClick={onToggleTerrain}
        className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium shadow transition-colors ${
          terrain3d
            ? 'bg-blue-600 text-white'
            : 'bg-white text-gray-700 hover:bg-gray-100 border border-gray-300'
        }`}
        title="Toggle 3D terrain"
      >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-4 h-4">
          <path d="M2 16l5-7 3 4 4-6 4 9H2z" />
        </svg>
        3D
      </button>
      <button
        onClick={onToggleSlope}
        className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium shadow transition-colors ${
          slopeShading
            ? 'bg-blue-600 text-white'
            : 'bg-white text-gray-700 hover:bg-gray-100 border border-gray-300'
        }`}
        title="Toggle slope angle shading"
      >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-4 h-4">
          <path d="M3 17l14-14v6l-6 8H3z" />
        </svg>
        Slope
      </button>
    </div>
  );
}
