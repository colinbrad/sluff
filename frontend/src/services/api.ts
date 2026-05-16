import type { GameMap, Guide, Round, Session, Player, Team, TeamRoute } from '../types/game';
import { useGuideStore } from '../stores/guideStore';

const API_URL = import.meta.env.VITE_API_URL || '';
const BASE = `${API_URL}/api`;

function authHeaders(): HeadersInit {
  const token = useGuideStore.getState().token;
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  if (!res.ok) {
    if (res.status === 401) {
      useGuideStore.getState().clearAuth();
    }
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

function authRequest<T>(path: string, options?: RequestInit): Promise<T> {
  return request<T>(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...(options?.headers as Record<string, string> | undefined),
    },
  });
}

export interface AuthResponse {
  token: string;
  guide: Guide;
}

export const registerGuide = (username: string, password: string) =>
  request<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });

export const loginGuide = (username: string, password: string) =>
  request<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });

export const createMap = (name: string, description: string) =>
  authRequest<GameMap>('/guide/maps', {
    method: 'POST',
    body: JSON.stringify({ name, description }),
  });

export const listMaps = () => authRequest<GameMap[]>('/guide/maps');

export const getMap = (id: string) => authRequest<GameMap>(`/guide/maps/${id}`);

export const updateMap = (id: string, data: { name?: string; description?: string }) =>
  authRequest<GameMap>(`/guide/maps/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });

export const deleteMap = (id: string) =>
  authRequest<void>(`/guide/maps/${id}`, { method: 'DELETE' });

export const createRound = (mapId: string, data: {
  round_number: number;
  name: string;
  start_point: GeoJSON.Geometry;
  end_point: GeoJSON.Geometry;
  corridor: GeoJSON.Geometry;
  no_go_zones?: GeoJSON.Polygon[];
}) =>
  authRequest<Round>(`/guide/maps/${mapId}/rounds`, {
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
  authRequest<Round>(`/guide/maps/${mapId}/rounds/${roundId}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });

export const deleteRound = (mapId: string, roundId: string) =>
  authRequest<void>(`/guide/maps/${mapId}/rounds/${roundId}`, { method: 'DELETE' });

export const createSession = (mapId: string, timeLimitSec?: number) =>
  authRequest<Session>('/sessions', {
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
  authRequest<Session>(`/sessions/${sessionId}/start`, { method: 'POST' });

export const submitRoute = (sessionId: string, roundId: string, teamId: string, path: GeoJSON.Geometry) =>
  request<TeamRoute>(`/sessions/${sessionId}/rounds/${roundId}/submit`, {
    method: 'POST',
    body: JSON.stringify({ team_id: teamId, path }),
  });

export const getScores = (sessionId: string, roundId: string) =>
  request<TeamRoute[]>(`/sessions/${sessionId}/rounds/${roundId}/scores`);

export interface SoloSessionResponse {
  session: Session;
  player: Player;
  team: Team;
}

export const createSoloSession = (mapId: string, playerName: string, timeLimitSec?: number) =>
  authRequest<SoloSessionResponse>('/sessions/solo', {
    method: 'POST',
    body: JSON.stringify({ map_id: mapId, player_name: playerName, time_limit_sec: timeLimitSec }),
  });

export interface DemoNextRoundResponse {
  session: Session;
  round: Round | null;
}

export const createDemoSession = (playerName: string) =>
  request<SoloSessionResponse>('/sessions/demo', {
    method: 'POST',
    body: JSON.stringify({ player_name: playerName }),
  });

export const demoNextRound = (sessionId: string) =>
  request<DemoNextRoundResponse>(`/sessions/${sessionId}/demo/next`, { method: 'POST' });

export const getCurrentRound = (sessionId: string) =>
  request<Round>(`/sessions/${sessionId}/current-round`);

// Guide admin actions
export const kickPlayer = (sessionId: string, playerId: string) =>
  authRequest<void>(`/sessions/${sessionId}/players/${playerId}`, { method: 'DELETE' });

export const clearRoute = (sessionId: string, roundId: string, teamId: string) =>
  authRequest<void>(`/sessions/${sessionId}/rounds/${roundId}/routes/${teamId}`, { method: 'DELETE' });
