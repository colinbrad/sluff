import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { GameMap } from '../../types/game';
import * as api from '../../services/api';

export default function CreateSession() {
  const [maps, setMaps] = useState<GameMap[]>([]);
  const [selectedMap, setSelectedMap] = useState('');
  const [timeLimit, setTimeLimit] = useState(300);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    api.listMaps().then((m) => {
      setMaps(m);
      if (m.length > 0) setSelectedMap(m[0].id);
      setLoading(false);
    });
  }, []);

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
    <div className="max-w-md mx-auto bg-white rounded-lg shadow p-6">
      <h2 className="text-xl font-bold mb-4">Create Game Session</h2>

      {maps.length === 0 ? (
        <p className="text-gray-500">
          No maps available. An admin needs to create one first.
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
  );
}
