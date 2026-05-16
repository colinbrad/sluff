import { useNavigate } from 'react-router-dom';
import { useGuideStore } from '../stores/guideStore';

export default function Home() {
  const navigate = useNavigate();
  const guide = useGuideStore((s) => s.guide);

  return (
    <div className="min-h-screen bg-gradient-to-b from-blue-50 to-white flex items-center justify-center p-4">
      <div className="text-center max-w-lg w-full">
        <h1 className="text-4xl sm:text-6xl font-bold text-gray-900 mb-2">Sluff</h1>
        <p className="text-lg sm:text-xl text-gray-600 mb-8">
          Draw a safe backcountry ski tour
        </p>

        <div className="flex flex-col gap-3 sm:gap-4">
          <button
            onClick={() => navigate('/demo')}
            className="px-6 py-3 sm:px-8 sm:py-4 bg-green-600 text-white rounded-xl text-base sm:text-lg font-semibold hover:bg-green-700 shadow-lg transition-colors"
          >
            Try Demo
          </button>
          <button
            onClick={() => navigate('/join')}
            className="px-6 py-3 sm:px-8 sm:py-4 bg-blue-600 text-white rounded-xl text-base sm:text-lg font-semibold hover:bg-blue-700 shadow-lg transition-colors"
          >
            Join Game
          </button>
          <button
            onClick={() => navigate(guide ? '/guide' : '/guide/login')}
            className="px-6 py-3 sm:px-8 sm:py-4 bg-white text-gray-700 rounded-xl text-base sm:text-lg font-semibold hover:bg-gray-50 shadow border transition-colors"
          >
            {guide ? `Guide Panel (${guide.username})` : 'Guide Sign In'}
          </button>
        </div>
      </div>
    </div>
  );
}
