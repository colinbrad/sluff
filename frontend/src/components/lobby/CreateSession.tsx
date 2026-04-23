import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { GameMap } from '../../types/game';
import * as api from '../../services/api';

export default function CreateSession() {
  const [maps, setMaps] = useState<GameMap[]>([]);
  const [selectedMap, setSelectedMap] = useState('');
  const [timeLimit, setTimeLimit] = useState(300);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    api.listMaps().then((m) => {
      setMaps(m);
      const preselect = searchParams.get('map');
      const initial = preselect && m.some((x) => x.id === preselect) ? preselect : m[0]?.id ?? '';
      setSelectedMap(initial);
      setLoading(false);
    });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleCreate = async () => {
    if (!selectedMap) return;
    const session = await api.createSession(selectedMap, timeLimit);
    navigate(`/session/${session.id}/lobby`);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading maps...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-start sm:items-center justify-center p-4">
    <div className="max-w-md w-full bg-white rounded-lg shadow p-4 sm:p-6">
      <div className="flex items-center gap-3 mb-4">
        <button onClick={() => navigate('/guide')} className="text-gray-400 hover:text-gray-600">&larr;</button>
        <h2 className="text-xl font-bold">Create Game Session</h2>
      </div>

      {maps.length === 0 ? (
        <p className="text-gray-500">
          No maps available. A guide needs to create one first.
        </p>
      ) : (
        <>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Select Map
            </label>
            <select
              value={selectedMap}
              onChange={(e) => setSelectedMap(e.target.value)}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {maps.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name} ({m.rounds?.length || '?'} rounds)
                </option>
              ))}
            </select>
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Time Limit per Round (seconds)
            </label>
            <input
              type="number"
              value={timeLimit}
              onChange={(e) => setTimeLimit(Number(e.target.value))}
              min={60}
              max={900}
              step={30}
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <button
            onClick={handleCreate}
            className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            Create Session
          </button>
        </>
      )}
    </div>
    </div>
  );
}
