import type { GamePhase, Player, Round, ScoreDetails } from './game';

export interface WSMessage {
  type: string;
  payload: unknown;
}

// Client -> Server
export interface CursorMovePayload {
  lat: number;
  lng: number;
}

export interface DrawingUpdatePayload {
  team_id: string;
  path: GeoJSON.LineString;
}

export interface DrawingSubmitPayload {
  team_id: string;
  path: GeoJSON.LineString;
}

// Server -> Client
export interface CursorUpdatePayload {
  player_id: string;
  lat: number;
  lng: number;
}

export interface DrawingUpdateFromServer {
  player_id: string;
  team_id: string;
  path: GeoJSON.LineString;
}

export interface GameStatePayload {
  phase: GamePhase;
  current_round: number;
  time_remaining: number;
}

export interface PlayerEventPayload {
  player: Player;
}

export interface RoundStartPayload extends Round {}

export interface ScoresPayload {
  team_scores: Array<{
    team_id: string;
    score: ScoreDetails;
  }>;
}

export interface ErrorPayload {
  message: string;
}
