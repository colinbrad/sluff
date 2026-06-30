package model

import "encoding/json"

// MustMarshal marshals v to JSON, discarding errors. The well-typed payload
// structs in this package cannot fail to marshal.
func MustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client-to-server message type constants.
const (
	MsgCursorMove    = "cursor_move"
	MsgDrawingUpdate = "drawing_update"
	MsgPing          = "ping"
)

// Server-to-client message type constants.
const (
	MsgPlayerLeft    = "player_left"
	MsgCursorUpdate  = "cursor_update"
	MsgGameState     = "game_state"
	MsgRoundStart    = "round_start"
	MsgRoundEnd      = "round_end"
	MsgScores        = "scores"
	MsgTeamSubmitted = "team_submitted"
)

// CursorMovePayload is sent by clients to share their cursor position.
type CursorMovePayload struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// CursorUpdatePayload is broadcast to teammates with an originating player ID.
type CursorUpdatePayload struct {
	PlayerID string  `json:"player_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

// DrawingUpdatePayload carries an in-progress route drawing for collaborative editing.
type DrawingUpdatePayload struct {
	PlayerID string          `json:"player_id,omitempty"`
	TeamID   string          `json:"team_id"`
	Path     json.RawMessage `json:"path"`
}

// GameStatePayload reports the session's current phase and round timing.
type GameStatePayload struct {
	Phase         GamePhase `json:"phase"`
	CurrentRound  int       `json:"current_round"`
	TimeRemaining int       `json:"time_remaining"`
}

// ScoresPayload is broadcast at the end of a round with per-team results.
type ScoresPayload struct {
	TeamScores []TeamScoreEntry `json:"team_scores"`
}

// TeamScoreEntry pairs a team ID with that team's score details.
type TeamScoreEntry struct {
	TeamID string       `json:"team_id"`
	Score  ScoreDetails `json:"score"`
}

// TeamSubmittedPayload announces (without leaking the route) that a team has
// submitted for the current round, driving the submission-progress indicator.
type TeamSubmittedPayload struct {
	TeamID string `json:"team_id"`
}
