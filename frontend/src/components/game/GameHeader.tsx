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
    <header className="bg-white shadow-sm border-b px-4 py-3 flex items-center justify-between z-10">
      <div className="flex items-center gap-4">
        <button
          onClick={onExit}
          className="text-gray-500 hover:text-gray-700 text-sm"
        >
          &larr; Exit
        </button>
        <div>
          <span className="text-sm text-gray-500">Round {roundNumber}</span>
          {roundName && (
            <span className="text-sm text-gray-400 ml-2">{roundName}</span>
          )}
        </div>
      </div>

      <div className="flex items-center gap-4">
        {phase === 'playing' && (
          <div
            className={`font-mono text-2xl font-bold ${
              isLow ? 'text-red-600 animate-pulse' : 'text-gray-900'
            }`}
          >
            {minutes}:{seconds.toString().padStart(2, '0')}
          </div>
        )}

        {submitted && (
          <span className="text-sm text-green-600 font-medium">Submitted</span>
        )}

        {phase === 'waiting' && (
          <span className="text-sm text-gray-500">Waiting to start...</span>
        )}
        {phase === 'scoring' && (
          <span className="text-sm text-blue-600 font-medium">Scoring...</span>
        )}
        {phase === 'finished' && (
          <span className="text-sm text-purple-600 font-medium">
            Game Over
          </span>
        )}
      </div>
    </header>
  );
}
