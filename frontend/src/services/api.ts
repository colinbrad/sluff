import type { GameMap, Round, Session, Player, Team, TeamRoute } from '../types/game';

const API_URL = import.meta.env.VITE_API_URL || '';
const BASE = `${API_URL}/api`;

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

// Guide - Maps
export const createMap = (name: string, description: string) =>
  request<GameMap>('/guide/maps', {
    method: 'POST',
    body: JSON.stringify({ name, description }),
  });

export const listMaps = () => request<GameMap[]>('/guide/maps');

export const getMap = (id: string) => request<GameMap>(`/guide/maps/${id}`);

export const updateMap = (id: string, data: { name?: string; description?: string }) =>
  request<GameMap>(`/guide/maps/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });

export const deleteMap = (id: string) =>
  request<void>(`/guide/maps/${id}`, { method: 'DELETE' });

// Guide - Rounds
export const createRound = (mapId: string, data: {
  round_number: number;
  name: string;
  start_point: GeoJSON.Geometry;
  end_point: GeoJSON.Geometry;
  corridor: GeoJSON.Geometry;
}) =>
  request<Round>(`/guide/maps/${mapId}/rounds`, {
    method: 'POST',
    body: JSON.stringify(data),
  });

export const updateRound = (mapId: string, roundId: string, data: Partial<{
  round_number: number;
  name: string;
  start_point: GeoJSON.Geometry;
  end_point: GeoJSON.Geometry;
  corridor: GeoJSON.Geometry;
}>) =>
  request<Round>(`/guide/maps/${mapId}/rounds/${roundId}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });

export const deleteRound = (mapId: string, roundId: string) =>
  request<void>(`/guide/maps/${mapId}/rounds/${roundId}`, { method: 'DELETE' });

// Sessions
export const createSession = (mapId: string, timeLimitSec?: number) =>
  request<Session>('/sessions', {
    method: 'POST',
    body: JSON.stringify({ map_id: mapId, time_limit_sec: timeLimitSec }),
  });

export const getSession = (id: string) => request<Session>(`/sessions/${id}`);

export const getSessionByCode = (code: string) =>
  request<Session>(`/sessions/code/${code}`);

export const joinSession = (sessionId: string, name: string) =>
  request<Player>(`/sessions/${sessionId}/join`, {
    method: 'POST',
    body: JSON.stringify({ name }),
  });

export const createTeam = (sessionId: string, name: string, color: string) =>
  request<Team>(`/sessions/${sessionId}/teams`, {
    method: 'POST',
    body: JSON.stringify({ name, color }),
  });

export const joinTeam = (sessionId: string, teamId: string, playerId: string) =>
  request<{ status: string }>(`/sessions/${sessionId}/teams/${teamId}/join`, {
    method: 'POST',
    body: JSON.stringify({ player_id: playerId }),
  });

export const startGame = (sessionId: string) =>
  request<Session>(`/sessions/${sessionId}/start`, { method: 'POST' });

export const submitRoute = (sessionId: string, roundId: string, teamId: string, path: GeoJSON.Geometry) =>
  request<TeamRoute>(`/sessions/${sessionId}/rounds/${roundId}/submit`, {
    method: 'POST',
    body: JSON.stringify({ team_id: teamId, path }),
  });

export const getScores = (sessionId: string, roundId: string) =>
  request<TeamRoute[]>(`/sessions/${sessionId}/rounds/${roundId}/scores`);

// Solo mode
export interface SoloSessionResponse {
  session: Session;
  player: Player;
  team: Team;
}

export const createSoloSession = (mapId: string, playerName: string, timeLimitSec?: number) =>
  request<SoloSessionResponse>('/sessions/solo', {
    method: 'POST',
    body: JSON.stringify({ map_id: mapId, player_name: playerName, time_limit_sec: timeLimitSec }),
  });
