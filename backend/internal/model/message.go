package model

import "encoding/json"

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client -> Server message types
const (
	MsgCursorMove     = "cursor_move"
	MsgDrawingUpdate  = "drawing_update"
	MsgDrawingSubmit  = "drawing_submit"
	MsgPing           = "ping"
)

// Server -> Client message types
const (
	MsgPlayerJoined = "player_joined"
	MsgPlayerLeft   = "player_left"
	MsgCursorUpdate = "cursor_update"
	MsgGameState    = "game_state"
	MsgRoundStart   = "round_start"
	MsgRoundEnd     = "round_end"
	MsgScores       = "scores"
	MsgError        = "error"
)

type CursorMovePayload struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type CursorUpdatePayload struct {
	PlayerID string  `json:"player_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type DrawingUpdatePayload struct {
	PlayerID string          `json:"player_id,omitempty"`
	TeamID   string          `json:"team_id"`
	Path     json.RawMessage `json:"path"`
}

type GameStatePayload struct {
	Phase         GamePhase `json:"phase"`
	CurrentRound  int       `json:"current_round"`
	TimeRemaining int       `json:"time_remaining"`
}

type PlayerEventPayload struct {
	Player Player `json:"player"`
}

type ScoresPayload struct {
	TeamScores []TeamScoreEntry `json:"team_scores"`
}

type TeamScoreEntry struct {
	TeamID  string       `json:"team_id"`
	Score   ScoreDetails `json:"score"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}
