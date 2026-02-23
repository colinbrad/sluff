import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { GameMap } from '../../types/game';
import * as api from '../../services/api';
import { usePlayerStore } from '../../stores/playerStore';

export default function SoloSetup() {
  const [maps, setMaps] = useState<GameMap[]>([]);
  const [selectedMap, setSelectedMap] = useState('');
  const [playerName, setPlayerName] = useState('');
  const [timeLimit, setTimeLimit] = useState(300);
  const [loading, setLoading] = useState(true);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const setPlayer = usePlayerStore((s) => s.setPlayer);

  useEffect(() => {
    api.listMaps().then((m) => {
      setMaps(m);
      if (m.length > 0) setSelectedMap(m[0].id);
      setLoading(false);
    });
  }, []);

  const handleStart = async () => {
    if (!selectedMap || !playerName.trim()) return;
    setStarting(true);
    setError('');
    try {
      const { session, player } = await api.createSoloSession(
        selectedMap,
        playerName.trim(),
        timeLimit,
      );
      setPlayer(player);
      await api.startGame(session.id);
      navigate(`/session/${session.id}/play`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start game');
      setStarting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading maps...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-blue-50 to-white flex items-center justify-center p-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow p-6">
        <div className="flex items-center gap-3 mb-6">
          <button
            onClick={() => navigate('/')}
            className="text-gray-500 hover:text-gray-700"
          >
            &larr;
          </button>
          <h2 className="text-xl font-bold">Solo Play</h2>
        </div>

        {maps.length === 0 ? (
          <p className="text-gray-500">
            No maps available. An admin needs to create one first.
          </p>
        ) : (
          <>
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Your Name
              </label>
              <input
                type="text"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                placeholder="Enter your name"
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
                autoFocus
              />
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Select Map
              </label>
              <select
                value={selectedMap}
                onChange={(e) => setSelectedMap(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
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
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>

            {error && (
              <p className="text-red-600 text-sm mb-4">{error}</p>
            )}

            <button
              onClick={handleStart}
              disabled={!selectedMap || !playerName.trim() || starting}
              className="w-full px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 font-semibold transition-colors"
            >
              {starting ? 'Starting...' : 'Play'}
            </button>
          </>
        )}
      </div>
    </div>
  );
}
