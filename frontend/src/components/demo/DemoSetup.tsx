import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as api from '../../services/api';
import { usePlayerStore } from '../../stores/playerStore';

export default function DemoSetup() {
  const [playerName, setPlayerName] = useState('');
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const setPlayer = usePlayerStore((s) => s.setPlayer);

  const handleStart = async () => {
    if (!playerName.trim()) return;
    setStarting(true);
    setError('');
    try {
      const { session, player } = await api.createDemoSession(playerName.trim());
      setPlayer(player);
      navigate(`/session/${session.id}/play?demo=true`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start demo');
      setStarting(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-blue-50 to-white flex items-center justify-center p-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow p-6">
        <div className="flex items-center gap-3 mb-2">
          <button onClick={() => navigate('/')} className="text-gray-500 hover:text-gray-700">
            &larr;
          </button>
          <h2 className="text-xl font-bold">Try Sluff</h2>
        </div>
        <p className="text-gray-500 text-sm mb-6">
          No account needed. Draw a backcountry ski route and see how well it scores.
        </p>

        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-1">Your Name</label>
          <input
            type="text"
            value={playerName}
            onChange={(e) => setPlayerName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleStart()}
            placeholder="Enter your name"
            className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
            autoFocus
          />
        </div>

        {error && <p className="text-red-600 text-sm mb-4">{error}</p>}

        <button
          onClick={handleStart}
          disabled={!playerName.trim() || starting}
          className="w-full px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 font-semibold transition-colors"
        >
          {starting ? 'Starting...' : 'Play Demo'}
        </button>
      </div>
    </div>
  );
}
