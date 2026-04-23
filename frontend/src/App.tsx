import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Home from './components/Home';

const GuideDashboard = lazy(() => import('./components/guide/GuideDashboard'));
const MapEditor = lazy(() => import('./components/guide/MapEditor'));
const CreateSession = lazy(() => import('./components/lobby/CreateSession'));
const JoinSession = lazy(() => import('./components/lobby/JoinSession'));
const WaitingRoom = lazy(() => import('./components/lobby/WaitingRoom'));
const GameView = lazy(() => import('./components/game/GameView'));
const SoloSetup = lazy(() => import('./components/solo/SoloSetup'));

export default function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={null}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/solo" element={<SoloSetup />} />
          <Route path="/guide" element={<GuideDashboard />} />
          <Route path="/guide/maps/:mapId" element={<MapEditor />} />
          <Route path="/create" element={<CreateSession />} />
          <Route path="/join" element={<JoinSession />} />
          <Route path="/session/:sessionId/lobby" element={<WaitingRoom />} />
          <Route path="/session/:sessionId/play" element={<GameView />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
