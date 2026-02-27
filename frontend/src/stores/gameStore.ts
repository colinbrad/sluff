import { create } from 'zustand';
import type { Session, Round, ScoreDetails, GamePhase, TeamRoute } from '../types/game';

interface Cursor {
  playerId: string;
  lat: number;
  lng: number;
}

interface GameState {
  session: Session | null;
  currentRound: Round | null;
  phase: GamePhase | null;
  timeRemaining: number;
  teamCursors: Map<string, Cursor>;
  teamDrawing: GeoJSON.LineString | null;
  teamScores: Array<{ team_id: string; score: ScoreDetails }>;
  routeResults: TeamRoute[];

  setSession: (session: Session | null) => void;
  setCurrentRound: (round: Round | null) => void;
  setPhase: (phase: GamePhase) => void;
  setTimeRemaining: (t: number) => void;
  updateCursor: (playerId: string, lat: number, lng: number) => void;
  setTeamDrawing: (path: GeoJSON.LineString | null) => void;
  setTeamScores: (scores: Array<{ team_id: string; score: ScoreDetails }>) => void;
  setRouteResults: (routes: TeamRoute[]) => void;
  reset: () => void;
}

export const useGameStore = create<GameState>((set) => ({
  session: null,
  currentRound: null,
  phase: null,
  timeRemaining: 0,
  teamCursors: new Map(),
  teamDrawing: null,
  teamScores: [],
  routeResults: [],

  setSession: (session) => set({ session }),
  setCurrentRound: (round) => set({ currentRound: round }),
  setPhase: (phase) => set({ phase }),
  setTimeRemaining: (t) => set({ timeRemaining: t }),
  updateCursor: (playerId, lat, lng) =>
    set((state) => {
      const cursors = new Map(state.teamCursors);
      cursors.set(playerId, { playerId, lat, lng });
      return { teamCursors: cursors };
    }),
  setTeamDrawing: (path) => set({ teamDrawing: path }),
  setTeamScores: (scores) => set({ teamScores: scores }),
  setRouteResults: (routes) => set({ routeResults: routes }),
  reset: () =>
    set({
      session: null,
      currentRound: null,
      phase: null,
      timeRemaining: 0,
      teamCursors: new Map(),
      teamDrawing: null,
      teamScores: [],
      routeResults: [],
    }),
}));
