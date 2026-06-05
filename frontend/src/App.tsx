import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Home from './components/Home';
import { useGuideStore } from './stores/guideStore';

const GuideDashboard = lazy(() => import('./components/guide/GuideDashboard'));
const MapEditor = lazy(() => import('./components/guide/MapEditor'));
const ImportLabeler = lazy(() => import('./components/guide/ImportLabeler'));
const GuideLogin = lazy(() => import('./components/auth/GuideLogin'));
const CreateSession = lazy(() => import('./components/lobby/CreateSession'));
const JoinSession = lazy(() => import('./components/lobby/JoinSession'));
const WaitingRoom = lazy(() => import('./components/lobby/WaitingRoom'));
const GameView = lazy(() => import('./components/game/GameView'));
const SoloSetup = lazy(() => import('./components/solo/SoloSetup'));
const DemoSetup = lazy(() => import('./components/demo/DemoSetup'));

function GuideRoute({ children }: { children: React.ReactNode }) {
  const token = useGuideStore((s) => s.token);
  return token ? <>{children}</> : <Navigate to="/guide/login" replace />;
}

export default function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={null}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/guide/login" element={<GuideLogin />} />
          <Route
            path="/guide"
            element={
              <GuideRoute>
                <GuideDashboard />
              </GuideRoute>
            }
          />
          <Route
            path="/guide/maps/:mapId"
            element={
              <GuideRoute>
                <MapEditor />
              </GuideRoute>
            }
          />
          <Route
            path="/guide/import"
            element={
              <GuideRoute>
                <ImportLabeler />
              </GuideRoute>
            }
          />
          <Route path="/demo" element={<DemoSetup />} />
          <Route
            path="/solo"
            element={
              <GuideRoute>
                <SoloSetup />
              </GuideRoute>
            }
          />
          <Route
            path="/create"
            element={
              <GuideRoute>
                <CreateSession />
              </GuideRoute>
            }
          />
          <Route path="/join" element={<JoinSession />} />
          <Route path="/session/:sessionId/lobby" element={<WaitingRoom />} />
          <Route path="/session/:sessionId/play" element={<GameView />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
