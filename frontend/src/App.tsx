import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Home from './components/Home';
import AdminDashboard from './components/admin/AdminDashboard';
import MapEditor from './components/admin/MapEditor';
import CreateSession from './components/lobby/CreateSession';
import JoinSession from './components/lobby/JoinSession';
import WaitingRoom from './components/lobby/WaitingRoom';
import GameView from './components/game/GameView';
import SoloSetup from './components/solo/SoloSetup';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/solo" element={<SoloSetup />} />
        <Route path="/admin" element={<AdminDashboard />} />
        <Route path="/admin/maps/:mapId" element={<MapEditor />} />
        <Route path="/create" element={<CreateSession />} />
        <Route path="/join" element={<JoinSession />} />
        <Route path="/session/:sessionId/lobby" element={<WaitingRoom />} />
        <Route path="/session/:sessionId/play" element={<GameView />} />
      </Routes>
    </BrowserRouter>
  );
}
