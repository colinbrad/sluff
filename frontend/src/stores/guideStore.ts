import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Guide } from '../types/game';

interface GuideState {
  guide: Guide | null;
  token: string | null;
  setAuth: (guide: Guide, token: string) => void;
  clearAuth: () => void;
}

export const useGuideStore = create<GuideState>()(
  persist(
    (set) => ({
      guide: null,
      token: null,
      setAuth: (guide, token) => set({ guide, token }),
      clearAuth: () => set({ guide: null, token: null }),
    }),
    { name: 'sluff-guide' }
  )
);
