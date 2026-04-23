export type GamePhase = 'waiting' | 'playing' | 'scoring' | 'finished';

export interface Guide {
  id: string;
  username: string;
  created_at: string;
}

export interface GameMap {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
  rounds?: Round[];
}

export interface Round {
  id: string;
  map_id: string;
  round_number: number;
  name: string;
  start_point: GeoJSON.Point;
  end_point: GeoJSON.Point;
  corridor: GeoJSON.Polygon;
  no_go_zones?: GeoJSON.Polygon[];
}

export interface Session {
  id: string;
  map_id: string;
  guide_id?: string;
  code: string;
  phase: GamePhase;
  current_round: number;
  time_limit_sec: number;
  is_solo: boolean;
  created_at: string;
  teams?: Team[];
  players?: Player[];
}

export interface Team {
  id: string;
  session_id: string;
  name: string;
  color: string;
}

export interface Player {
  id: string;
  session_id: string;
  team_id: string;
  name: string;
  is_online: boolean;
}

export interface TeamRoute {
  id: string;
  session_id: string;
  round_id: string;
  team_id: string;
  path: string;
  score: number | null;
  details: ScoreDetails | null;
  submitted_at: string | null;
}

export interface ScoreDetails {
  total_points: number;
  points_in_corridor: number;
  percent_in_corridor: number;
  route_length_km: number;
  max_deviation_m: number;
  connects_start: boolean;
  connects_end: boolean;
  points_in_no_go_zone?: number;
  no_go_zone_penalty?: number;
  final_score: number;
}
