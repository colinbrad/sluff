import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import type { Session } from '../../types/game';
import * as api from '../../services/api';
import { usePlayerStore } from '../../stores/playerStore';
import { useGameStore } from '../../stores/gameStore';
import { GameWebSocket } from '../../services/ws';

const TEAM_COLORS = ['#3B82F6', '#EF4444', '#10B981', '#F59E0B'];
const TEAM_NAMES = ['Blue Team', 'Red Team', 'Green Team', 'Gold Team'];

export default function WaitingRoom() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const player = usePlayerStore((s) => s.player);
  const setPlayer = usePlayerStore((s) => s.setPlayer);
  const setSession = useGameStore((s) => s.setSession);
  const [session, setLocalSession] = useState<Session | null>(null);
  const [playerName, setPlayerName] = useState('');
  const [, setWs] = useState<GameWebSocket | null>(null);

  const refreshSession = useCallback(async () => {
    if (!sessionId) return;
    const s = await api.getSession(sessionId);
    setLocalSession(s);
    setSession(s);
  }, [sessionId, setSession]);

  useEffect(() => {
    refreshSession();
  }, [refreshSession]);

  // Connect WebSocket once player is set
  useEffect(() => {
    if (!sessionId || !player) return;
    const socket = new GameWebSocket(sessionId, player.id);
    socket.connect();
    socket.onMessage((msg) => {
      if (msg.type === 'player_joined' || msg.type === 'player_left') {
        refreshSession();
      }
      if (msg.type === 'game_state') {
        navigate(`/session/${sessionId}/play`);
      }
    });
    setWs(socket);
    return () => socket.close();
  }, [sessionId, player, navigate, refreshSession]);

  const handleJoin = async () => {
    if (!sessionId || !playerName.trim()) return;
    const p = await api.joinSession(sessionId, playerName.trim());
    setPlayer(p);
  };

  const handleCreateTeam = async () => {
    if (!sessionId || !session) return;
    const teamIndex = session.teams?.length || 0;
    if (teamIndex >= 4) return;
    await api.createTeam(
      sessionId,
      TEAM_NAMES[teamIndex],
      TEAM_COLORS[teamIndex]
    );
    refreshSession();
  };

  const handleJoinTeam = async (teamId: string) => {
    if (!sessionId || !player) return;
    await api.joinTeam(sessionId, teamId, player.id);
    setPlayer({ ...player, team_id: teamId });
    refreshSession();
  };

  const handleStart = async () => {
    if (!sessionId) return;
    await api.startGame(sessionId);
    navigate(`/session/${sessionId}/play`);
  };

  if (!session) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-600">Loading...</div>
      </div>
    );
  }

  // Not yet joined
  if (!player) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-start sm:items-center justify-center p-4">
        <div className="max-w-md w-full bg-white rounded-lg shadow p-4 sm:p-6">
          <h2 className="text-xl font-bold mb-2">Join Game</h2>
          <p className="text-sm text-gray-500 mb-4">
            Code: <span className="font-mono font-bold">{session.code}</span>
          </p>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Your Name
            </label>
            <input
              type="text"
              value={playerName}
              onChange={(e) => setPlayerName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleJoin()}
              placeholder="Enter your name"
              maxLength={20}
              className="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button
            onClick={handleJoin}
            disabled={!playerName.trim()}
            className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            Join
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-4xl mx-auto px-4 py-3 sm:py-4 flex items-center justify-between">
          <div>
            <h1 className="text-lg sm:text-xl font-bold">Waiting Room</h1>
            <p className="text-sm text-gray-500">
              Code:{' '}
              <span className="font-mono font-bold text-base sm:text-lg text-blue-600">
                {session.code}
              </span>
            </p>
          </div>
          <div className="text-xs sm:text-sm text-gray-500">
            {session.players?.length || 0}/8 players
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-4 sm:py-8">
        {/* Teams */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Teams</h2>
            {(session.teams?.length || 0) < 4 && (
              <button
                onClick={handleCreateTeam}
                className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 text-sm transition-colors"
              >
                + Add Team
              </button>
            )}
          </div>

          {!session.teams?.length ? (
            <p className="text-gray-500 text-sm">
              No teams yet. Create teams to get started.
            </p>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {session.teams.map((team) => {
                const teamPlayers =
                  session.players?.filter((p) => p.team_id === team.id) || [];
                const isOnTeam = player.team_id === team.id;

                return (
                  <div
                    key={team.id}
                    className="bg-white rounded-lg shadow p-4 border-l-4"
                    style={{ borderLeftColor: team.color }}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="font-medium">{team.name}</h3>
                      {!isOnTeam && (
                        <button
                          onClick={() => handleJoinTeam(team.id)}
                          className="px-3 py-1 text-sm bg-blue-50 text-blue-600 rounded hover:bg-blue-100 transition-colors"
                        >
                          Join
                        </button>
                      )}
                      {isOnTeam && (
                        <span className="text-sm text-green-600 font-medium">
                          Your Team
                        </span>
                      )}
                    </div>
                    <div className="space-y-1">
                      {teamPlayers.map((p) => (
                        <div key={p.id} className="text-sm text-gray-600">
                          {p.name}
                          {p.id === player.id && ' (you)'}
                        </div>
                      ))}
                      {teamPlayers.length === 0 && (
                        <div className="text-sm text-gray-400 italic">
                          No players yet
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Unassigned players */}
        {session.players?.some((p) => !p.team_id) && (
          <div className="mb-8">
            <h2 className="text-lg font-semibold mb-2">Unassigned Players</h2>
            <div className="flex flex-wrap gap-2">
              {session.players
                ?.filter((p) => !p.team_id)
                .map((p) => (
                  <span
                    key={p.id}
                    className="px-3 py-1 bg-gray-100 rounded-full text-sm"
                  >
                    {p.name}
                    {p.id === player.id && ' (you)'}
                  </span>
                ))}
            </div>
          </div>
        )}

        {/* Start button */}
        <button
          onClick={handleStart}
          disabled={
            !session.teams?.length ||
            session.teams.length < 2 ||
            !session.players?.some((p) => p.team_id)
          }
          className="w-full px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 font-semibold text-lg transition-colors"
        >
          Start Game
        </button>
      </main>
    </div>
  );
}
