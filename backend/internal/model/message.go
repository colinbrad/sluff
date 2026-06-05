package model

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// SafeMarshal returns the JSON encoding of v for use as a WSMessage payload.
// The well-typed payload structs in this package cannot legitimately fail to
// marshal; if one ever does we log and substitute JSON null rather than
// crashing the broadcasting goroutine.
func SafeMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("payload marshal failed", "err", err, "type", fmt.Sprintf("%T", v))
		return json.RawMessage("null")
	}
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
	MsgDrawingSubmit = "drawing_submit"
	MsgPing          = "ping"
)

// Server-to-client message type constants.
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

// PlayerEventPayload wraps a player record for join/leave broadcasts.
type PlayerEventPayload struct {
	Player Player `json:"player"`
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

// ErrorPayload carries a human-readable error message to the client.
type ErrorPayload struct {
	Message string `json:"message"`
}
