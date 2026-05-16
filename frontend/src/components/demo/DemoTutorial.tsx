export type DemoStep = 'welcome' | 'drawing' | 'ready';

interface DemoTutorialProps {
  step: DemoStep;
  onDismissWelcome: () => void;
}

export default function DemoTutorial({ step, onDismissWelcome }: DemoTutorialProps) {
  if (step === 'welcome') {
    return (
      <div className="fixed inset-0 z-50 bg-gray-900/80 flex items-center justify-center p-4">
        <div className="bg-white rounded-xl shadow-xl max-w-sm w-full p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">How to Play</h2>
          <ol className="space-y-3 text-gray-700 text-sm mb-6">
            <li>
              <span className="font-semibold text-green-600">1. Find the markers</span>
              {' '}— a green start and red end point mark your route targets.
            </li>
            <li>
              <span className="font-semibold text-blue-600">2. Draw a route</span>
              {' '}— click and drag to trace a freehand line from start to end. Press Enter or double-click to finish.
            </li>
            <li>
              <span className="font-semibold">3. Stay in the corridor</span>
              {' '}— the shaded area is the safe zone. Routes that stay inside score more points.
            </li>
            <li>
              <span className="font-semibold text-red-600">4. Avoid no-go zones</span>
              {' '}— red areas are hazards. Crossing them deducts points.
            </li>
            <li>
              <span className="font-semibold">5. Submit</span>
              {' '}— click Submit Route when you're happy with your line.
            </li>
          </ol>
          <button
            onClick={onDismissWelcome}
            className="w-full px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 font-semibold transition-colors"
          >
            Let's go!
          </button>
        </div>
      </div>
    );
  }

  const hints: Record<Exclude<DemoStep, 'welcome'>, string> = {
    drawing:
      'Draw your route — click and drag from the green start to the red end. Press Enter or double-click to finish.',
    ready: 'Route drawn! Click Submit Route when you\'re ready to score it.',
  };

  return (
    <div className="absolute top-3 left-0 right-0 z-20 px-4 pointer-events-none">
      <div className="max-w-md mx-auto">
        <div className="bg-gray-900/90 backdrop-blur-sm text-white text-sm rounded-lg px-4 py-2.5 text-center shadow-lg">
          {hints[step]}
        </div>
      </div>
    </div>
  );
}
