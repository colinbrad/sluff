import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { GameMap } from '../../types/game';
import * as api from '../../services/api';
import { useGuideStore } from '../../stores/guideStore';
import { parseFile } from '../../utils/importGeo';

export default function GuideDashboard() {
  const [maps, setMaps] = useState<GameMap[]>([]);
  const [newName, setNewName] = useState('');
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();
  const { guide, clearAuth } = useGuideStore();

  useEffect(() => {
    api
      .listMaps()
      .then(setMaps)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    const m = await api.createMap(newName.trim(), '');
    setMaps([m, ...maps]);
    setNewName('');
    navigate(`/guide/maps/${m.id}`);
  };

  const handleDelete = async (id: string) => {
    await api.deleteMap(id);
    setMaps(maps.filter((m) => m.id !== id));
  };

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setImporting(true);
    setImportError('');

    try {
      const fc = await parseFile(file);
      const usable = fc.features.filter((f) => {
        const t = f.geometry?.type;
        return (
          t === 'Point' ||
          t === 'Polygon' ||
          t === 'LineString' ||
          t === 'MultiPoint' ||
          t === 'MultiPolygon' ||
          t === 'MultiLineString'
        );
      });

      if (usable.length === 0) {
        setImportError('No usable features found in file.');
        setImporting(false);
        return;
      }

      const mapName = file.name.replace(/\.(kml|gpx|geojson|json)$/i, '');
      navigate('/guide/import', { state: { features: usable, mapName } });
    } catch (err) {
      setImportError(err instanceof Error ? err.message : 'Failed to import file');
    } finally {
      setImporting(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg text-gray-600">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-4xl mx-auto px-4 py-3 sm:py-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-gray-900">Sluff Guide</h1>
            {guide && <p className="text-sm text-gray-500">{guide.username}</p>}
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => navigate('/')}
              className="text-sm text-gray-500 hover:text-gray-700 py-1"
            >
              Home
            </button>
            <button
              onClick={() => {
                clearAuth();
                navigate('/');
              }}
              className="text-sm text-red-500 hover:text-red-700 py-1"
            >
              Sign Out
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-4 sm:py-8">
        <div className="bg-white rounded-lg shadow p-4 sm:p-6 mb-6 sm:mb-8">
          <h2 className="text-lg font-semibold mb-3 sm:mb-4">Create New Map</h2>
          <div className="flex flex-col sm:flex-row gap-3">
            <input
              type="text"
              placeholder="Map name (e.g., Wasatch Backcountry)"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              className="flex-1 px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleCreate}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors shrink-0"
            >
              Create
            </button>
          </div>

          <div className="mt-4 pt-4 border-t">
            <p className="text-sm text-gray-500 mb-2">Or import from a file:</p>
            <input
              ref={fileInputRef}
              type="file"
              accept=".kml,.gpx,.geojson,.json"
              onChange={handleImport}
              disabled={importing}
              className="hidden"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={importing}
              className="px-6 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors text-sm disabled:opacity-50"
            >
              {importing ? 'Importing...' : 'Import KML / GPX / GeoJSON'}
            </button>
            {importError && <p className="mt-2 text-sm text-red-600">{importError}</p>}
          </div>
        </div>

        <h2 className="text-lg font-semibold mb-4">Your Maps</h2>
        {maps.length === 0 ? (
          <p className="text-gray-500">No maps yet. Create one above.</p>
        ) : (
          <div className="grid gap-3 sm:gap-4">
            {maps.map((m) => (
              <div
                key={m.id}
                className="bg-white rounded-lg shadow p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
              >
                <div className="min-w-0">
                  <h3 className="font-medium text-gray-900 truncate">{m.name}</h3>
                  <p className="text-sm text-gray-500">
                    {m.rounds?.length || 0} round(s) &middot; Created{' '}
                    {new Date(m.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex gap-2 shrink-0">
                  <button
                    onClick={() => navigate(`/create?map=${m.id}`)}
                    className="flex-1 sm:flex-none px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm font-medium"
                  >
                    Create Session
                  </button>
                  <button
                    onClick={() => navigate(`/guide/maps/${m.id}`)}
                    className="flex-1 sm:flex-none px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors text-sm"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(m.id)}
                    className="flex-1 sm:flex-none px-4 py-2 bg-red-50 text-red-600 rounded-lg hover:bg-red-100 transition-colors text-sm"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
