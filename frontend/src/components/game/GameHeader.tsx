import type { GamePhase } from '../../types/game';

interface GameHeaderProps {
  roundNumber: number;
  roundName: string;
  timeRemaining: number;
  phase: GamePhase;
  submitted: boolean;
  onExit: () => void;
}

export default function GameHeader({
  roundNumber,
  roundName,
  timeRemaining,
  phase,
  submitted,
  onExit,
}: GameHeaderProps) {
  const minutes = Math.floor(timeRemaining / 60);
  const seconds = timeRemaining % 60;
  const isLow = timeRemaining <= 30;

  return (
    <header className="bg-white shadow-sm border-b px-2 py-2 sm:px-4 sm:py-3 flex items-center justify-between z-10 gap-2">
      <div className="flex items-center gap-2 sm:gap-4 min-w-0">
        <button
          onClick={onExit}
          className="text-gray-500 hover:text-gray-700 text-sm shrink-0 py-1"
        >
          &larr; <span className="hidden sm:inline">Exit</span>
        </button>
        <div className="min-w-0">
          <span className="text-xs sm:text-sm text-gray-500">Rd {roundNumber}</span>
          {roundName && (
            <span className="text-xs sm:text-sm text-gray-400 ml-1 sm:ml-2 truncate">
              {roundName}
            </span>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2 sm:gap-4 shrink-0">
        {phase === 'playing' && (
          <div
            className={`font-mono text-lg sm:text-2xl font-bold ${
              isLow ? 'text-red-600 animate-pulse' : 'text-gray-900'
            }`}
          >
            {minutes}:{seconds.toString().padStart(2, '0')}
          </div>
        )}

        {submitted && (
          <span className="text-xs sm:text-sm text-green-600 font-medium">Submitted</span>
        )}

        {phase === 'waiting' && (
          <span className="text-xs sm:text-sm text-gray-500">Waiting...</span>
        )}
        {phase === 'scoring' && (
          <span className="text-xs sm:text-sm text-blue-600 font-medium">Scoring...</span>
        )}
        {phase === 'finished' && (
          <span className="text-xs sm:text-sm text-purple-600 font-medium">Game Over</span>
        )}
      </div>
    </header>
  );
}
