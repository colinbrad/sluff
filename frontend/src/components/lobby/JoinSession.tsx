import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as api from '../../services/api';
import { usePlayerStore } from '../../stores/playerStore';

export default function JoinSession() {
  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [step, setStep] = useState<'code' | 'name'>('code');
  const [sessionId, setSessionId] = useState('');
  const navigate = useNavigate();
  const setPlayer = usePlayerStore((s) => s.setPlayer);

  const handleLookup = async () => {
    setError('');
    try {
      const session = await api.getSessionByCode(code.toUpperCase().trim());
      setSessionId(session.id);
      setStep('name');
    } catch {
      setError('Session not found. Check the code and try again.');
    }
  };

  const handleJoin = async () => {
    if (!name.trim()) return;
    setError('');
    try {
      const player = await api.joinSession(sessionId, name.trim());
      setPlayer(player);
      navigate(`/session/${sessionId}/lobby`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to join');
    }
  };

  return (
    <div className="max-w-md mx-auto bg-white rounded-lg shadow p-6">
      <h2 className="text-xl font-bold mb-4">Join Game</h2>

      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-600 rounded-lg text-sm">
          {error}
        </div>
      )}

      {step === 'code' ? (
        <>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Game Code
            </label>
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              placeholder="ABCD12"
              maxLength={6}
              className="w-full px-4 py-3 border rounded-lg text-center text-2xl tracking-widest font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button
            onClick={handleLookup}
            disabled={code.length < 6}
            className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            Find Game
          </button>
        </>
      ) : (
        <>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Your Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleJoin()}
              placeholder="Enter your name"
              maxLength={20}
              className="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button
            onClick={handleJoin}
            disabled={!name.trim()}
            className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            Join Game
          </button>
        </>
      )}
    </div>
  );
}
