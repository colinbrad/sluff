import { describe, it, expect, beforeEach } from 'vitest';
import { useGameStore } from '../stores/gameStore';
import { usePlayerStore } from '../stores/playerStore';
import type { Session, Round, Player } from '../types/game';

describe('gameStore', () => {
  beforeEach(() => {
    useGameStore.getState().reset();
  });

  it('starts with null values', () => {
    const state = useGameStore.getState();
    expect(state.session).toBeNull();
    expect(state.currentRound).toBeNull();
    expect(state.phase).toBeNull();
    expect(state.timeRemaining).toBe(0);
    expect(state.teamScores).toEqual([]);
  });

  it('setSession updates session', () => {
    const session: Session = {
      id: 's1',
      map_id: 'm1',
      code: 'ABC123',
      phase: 'waiting',
      current_round: 0,
      time_limit_sec: 300,
      is_solo: false,
      created_at: '2024-01-01T00:00:00Z',
    };
    useGameStore.getState().setSession(session);
    expect(useGameStore.getState().session).toEqual(session);
  });

  it('setCurrentRound updates round', () => {
    const round: Round = {
      id: 'r1',
      map_id: 'm1',
      round_number: 1,
      name: 'Round 1',
      start_point: { type: 'Point', coordinates: [10, 47] },
      end_point: { type: 'Point', coordinates: [10.1, 47.1] },
      corridor: { type: 'Polygon', coordinates: [[[10, 47], [10.1, 47], [10.1, 47.1], [10, 47.1], [10, 47]]] },
    };
    useGameStore.getState().setCurrentRound(round);
    expect(useGameStore.getState().currentRound).toEqual(round);
  });

  it('setPhase updates phase', () => {
    useGameStore.getState().setPhase('playing');
    expect(useGameStore.getState().phase).toBe('playing');
  });

  it('setTimeRemaining updates time', () => {
    useGameStore.getState().setTimeRemaining(120);
    expect(useGameStore.getState().timeRemaining).toBe(120);
  });

  it('updateCursor adds cursor to map', () => {
    useGameStore.getState().updateCursor('p1', 47.0, 10.0);
    const cursors = useGameStore.getState().teamCursors;
    expect(cursors.get('p1')).toEqual({ playerId: 'p1', lat: 47.0, lng: 10.0 });
  });

  it('updateCursor overwrites existing cursor', () => {
    useGameStore.getState().updateCursor('p1', 47.0, 10.0);
    useGameStore.getState().updateCursor('p1', 48.0, 11.0);
    const cursors = useGameStore.getState().teamCursors;
    expect(cursors.get('p1')).toEqual({ playerId: 'p1', lat: 48.0, lng: 11.0 });
    expect(cursors.size).toBe(1);
  });

  it('setTeamDrawing stores line', () => {
    const line: GeoJSON.LineString = {
      type: 'LineString',
      coordinates: [[10, 47], [10.1, 47.1]],
    };
    useGameStore.getState().setTeamDrawing(line);
    expect(useGameStore.getState().teamDrawing).toEqual(line);
  });

  it('setTeamScores stores scores', () => {
    const scores = [{ team_id: 't1', score: { total_points: 800, points_in_corridor: 10, percent_in_corridor: 80, route_length_km: 1.2, max_deviation_m: 50, connects_start: true, connects_end: true, final_score: 800 } }];
    useGameStore.getState().setTeamScores(scores);
    expect(useGameStore.getState().teamScores).toEqual(scores);
  });

  it('reset clears all state', () => {
    useGameStore.getState().setPhase('playing');
    useGameStore.getState().setTimeRemaining(100);
    useGameStore.getState().updateCursor('p1', 47, 10);
    useGameStore.getState().reset();

    const state = useGameStore.getState();
    expect(state.session).toBeNull();
    expect(state.phase).toBeNull();
    expect(state.timeRemaining).toBe(0);
    expect(state.teamCursors.size).toBe(0);
  });
});

describe('playerStore', () => {
  beforeEach(() => {
    usePlayerStore.getState().setPlayer(null);
  });

  it('starts with null player', () => {
    expect(usePlayerStore.getState().player).toBeNull();
  });

  it('setPlayer stores player', () => {
    const player: Player = {
      id: 'p1',
      session_id: 's1',
      team_id: 't1',
      name: 'Test Player',
      is_online: true,
    };
    usePlayerStore.getState().setPlayer(player);
    expect(usePlayerStore.getState().player).toEqual(player);
  });

  it('setPlayer to null clears player', () => {
    usePlayerStore.getState().setPlayer({ id: 'p1', session_id: 's1', team_id: 't1', name: 'Test', is_online: true });
    usePlayerStore.getState().setPlayer(null);
    expect(usePlayerStore.getState().player).toBeNull();
  });
});
