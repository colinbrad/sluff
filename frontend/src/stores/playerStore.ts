import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Player } from '../types/game';

interface PlayerState {
  player: Player | null;
  setPlayer: (player: Player | null) => void;
}

export const usePlayerStore = create<PlayerState>()(
  persist(
    (set) => ({
      player: null,
      setPlayer: (player) => set({ player }),
    }),
    { name: 'sluff-player' },
  ),
);
