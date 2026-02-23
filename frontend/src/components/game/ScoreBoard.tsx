import type { Team, ScoreDetails } from '../../types/game';

interface ScoreBoardProps {
  teamScores: Array<{ team_id: string; score: ScoreDetails }>;
  teams: Team[];
  onContinue: () => void;
}

export default function ScoreBoard({
  teamScores,
  teams,
  onContinue,
}: ScoreBoardProps) {
  const sorted = [...teamScores].sort(
    (a, b) => b.score.final_score - a.score.final_score
  );

  const getTeam = (teamId: string) =>
    teams.find((t) => t.id === teamId);

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center">
      <div className="max-w-2xl w-full px-4">
        <h1 className="text-3xl font-bold text-white text-center mb-8">
          Round Results
        </h1>

        <div className="space-y-4 mb-8">
          {sorted.map((entry, index) => {
            const team = getTeam(entry.team_id);
            return (
              <div
                key={entry.team_id}
                className="bg-gray-800 rounded-lg p-6 flex items-center gap-6"
              >
                <div
                  className={`text-4xl font-bold ${
                    index === 0
                      ? 'text-yellow-400'
                      : index === 1
                      ? 'text-gray-400'
                      : 'text-orange-700'
                  }`}
                >
                  #{index + 1}
                </div>

                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <div
                      className="w-4 h-4 rounded-full"
                      style={{ backgroundColor: team?.color || '#888' }}
                    />
                    <span className="text-white font-semibold text-lg">
                      {team?.name || 'Unknown Team'}
                    </span>
                  </div>

                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <span className="text-gray-400">In corridor</span>
                      <div className="text-white font-medium">
                        {entry.score.percent_in_corridor}%
                      </div>
                    </div>
                    <div>
                      <span className="text-gray-400">Route length</span>
                      <div className="text-white font-medium">
                        {entry.score.route_length_km} km
                      </div>
                    </div>
                    <div>
                      <span className="text-gray-400">Max deviation</span>
                      <div className="text-white font-medium">
                        {entry.score.max_deviation_m} m
                      </div>
                    </div>
                  </div>

                  <div className="flex gap-2 mt-2">
                    {entry.score.connects_start && (
                      <span className="text-xs px-2 py-0.5 bg-green-900 text-green-300 rounded">
                        Start
                      </span>
                    )}
                    {entry.score.connects_end && (
                      <span className="text-xs px-2 py-0.5 bg-green-900 text-green-300 rounded">
                        End
                      </span>
                    )}
                  </div>
                </div>

                <div className="text-right">
                  <div className="text-4xl font-bold text-white">
                    {Math.round(entry.score.final_score)}
                  </div>
                  <div className="text-gray-400 text-sm">/ 1000</div>
                </div>
              </div>
            );
          })}
        </div>

        <button
          onClick={onContinue}
          className="w-full px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold text-lg transition-colors"
        >
          Continue
        </button>
      </div>
    </div>
  );
}
