import { useState } from 'react';
import type { Player, Team, TeamRoute, Round } from '../../types/game';
import * as api from '../../services/api';

interface GuideSessionControlsProps {
  sessionId: string;
  players: Player[];
  teams: Team[];
  routeResults: TeamRoute[];
  currentRound: Round | null;
  phase: string;
  onEndRound: () => void;
  onRefreshSession: () => void;
}

export default function GuideSessionControls({
  sessionId,
  players,
  teams,
  routeResults,
  currentRound,
  phase,
  onEndRound,
  onRefreshSession,
}: GuideSessionControlsProps) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  const getTeamName = (teamId: string) => teams.find((t) => t.id === teamId)?.name ?? 'Unknown';

  const handleKick = async (playerId: string) => {
    setBusy(true);
    try {
      await api.kickPlayer(sessionId, playerId);
      onRefreshSession();
    } catch {
      // non-fatal
    } finally {
      setBusy(false);
    }
  };

  const handleClearRoute = async (teamId: string) => {
    if (!currentRound) return;
    setBusy(true);
    try {
      await api.clearRoute(sessionId, currentRound.id, teamId);
      onRefreshSession();
    } catch {
      // non-fatal
    } finally {
      setBusy(false);
    }
  };

  const handleEndRound = async () => {
    setBusy(true);
    try {
      await onEndRound();
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="absolute top-2 right-2 z-20">
      {/* Toggle button */}
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 px-3 py-1.5 bg-purple-700 text-white rounded-lg text-xs font-semibold shadow hover:bg-purple-800 transition-colors"
      >
        <span>Guide</span>
        <span>{open ? '▲' : '▼'}</span>
      </button>

      {/* Panel */}
      {open && (
        <div className="mt-1 w-64 bg-gray-900/95 backdrop-blur-sm rounded-lg shadow-xl border border-gray-700 text-sm overflow-hidden">
          {/* End round early */}
          {phase === 'playing' && (
            <div className="p-3 border-b border-gray-700">
              <button
                onClick={handleEndRound}
                disabled={busy}
                className="w-full py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 font-semibold transition-colors"
              >
                End Round &amp; Reveal Scores
              </button>
            </div>
          )}

          {/* Players — kick */}
          {players.length > 0 && (
            <div className="p-3 border-b border-gray-700">
              <p className="text-gray-400 text-xs font-medium uppercase tracking-wide mb-2">
                Players
              </p>
              <div className="space-y-1.5">
                {players.map((p) => (
                  <div key={p.id} className="flex items-center justify-between gap-2">
                    <span className="text-white truncate">
                      {p.name}
                      {p.team_id && (
                        <span className="text-gray-400 text-xs ml-1">
                          ({getTeamName(p.team_id)})
                        </span>
                      )}
                    </span>
                    <button
                      onClick={() => handleKick(p.id)}
                      disabled={busy}
                      className="shrink-0 px-2 py-0.5 bg-red-700 text-white rounded text-xs hover:bg-red-800 disabled:opacity-50 transition-colors"
                    >
                      Kick
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Route submissions — clear */}
          {routeResults.length > 0 && currentRound && (
            <div className="p-3">
              <p className="text-gray-400 text-xs font-medium uppercase tracking-wide mb-2">
                Submissions
              </p>
              <div className="space-y-1.5">
                {routeResults.map((r) => (
                  <div key={r.id} className="flex items-center justify-between gap-2">
                    <span className="text-white truncate">{getTeamName(r.team_id)}</span>
                    <button
                      onClick={() => handleClearRoute(r.team_id)}
                      disabled={busy}
                      className="shrink-0 px-2 py-0.5 bg-yellow-700 text-white rounded text-xs hover:bg-yellow-800 disabled:opacity-50 transition-colors"
                    >
                      Clear
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
