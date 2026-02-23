import { useNavigate } from 'react-router-dom';

export default function Home() {
  const navigate = useNavigate();

  return (
    <div className="min-h-screen bg-gradient-to-b from-blue-50 to-white flex items-center justify-center">
      <div className="text-center max-w-lg px-4">
        <h1 className="text-6xl font-bold text-gray-900 mb-2">Sluff</h1>
        <p className="text-xl text-gray-600 mb-8">
          Draw a safe backcountry ski tour
        </p>

        <div className="flex flex-col gap-4">
          <button
            onClick={() => navigate('/solo')}
            className="px-8 py-4 bg-green-600 text-white rounded-xl text-lg font-semibold hover:bg-green-700 shadow-lg transition-colors"
          >
            Solo Play
          </button>
          <button
            onClick={() => navigate('/join')}
            className="px-8 py-4 bg-blue-600 text-white rounded-xl text-lg font-semibold hover:bg-blue-700 shadow-lg transition-colors"
          >
            Join Game
          </button>
          <button
            onClick={() => navigate('/create')}
            className="px-8 py-4 bg-white text-gray-700 rounded-xl text-lg font-semibold hover:bg-gray-50 shadow border transition-colors"
          >
            Create Session
          </button>
          <button
            onClick={() => navigate('/admin')}
            className="px-8 py-3 text-gray-500 hover:text-gray-700 text-sm transition-colors"
          >
            Admin Panel
          </button>
        </div>
      </div>
    </div>
  );
}
