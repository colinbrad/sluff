package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// GamePhase enumerates the lifecycle states of a game session.
type GamePhase string

// Game phase constants.
const (
	PhaseWaiting  GamePhase = "waiting"
	PhasePlaying  GamePhase = "playing"
	PhaseScoring  GamePhase = "scoring"
	PhaseFinished GamePhase = "finished"
)

// Session is a single playthrough of a map by one or more teams.
type Session struct {
	ID           string    `json:"id"`
	MapID        string    `json:"map_id"`
	GuideID      string    `json:"guide_id"`
	Code         string    `json:"code"`
	Phase        GamePhase `json:"phase"`
	CurrentRound int       `json:"current_round"`
	TimeLimitSec int       `json:"time_limit_sec"`
	IsSolo       bool      `json:"is_solo"`
	CreatedAt    time.Time `json:"created_at"`
	Teams        []Team    `json:"teams,omitempty"`
	Players      []Player  `json:"players,omitempty"`
}

// Team groups players who collaborate on route submissions within a session.
type Team struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

// Player is a participant in a session, optionally assigned to a team.
type Player struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	TeamID    string `json:"team_id"`
	Name      string `json:"name"`
}

// TeamRoute is a team's submitted GeoJSON LineString for a round, along with its score.
type TeamRoute struct {
	ID          string        `json:"id"`
	SessionID   string        `json:"session_id"`
	RoundID     string        `json:"round_id"`
	TeamID      string        `json:"team_id"`
	Path        string        `json:"path"` // GeoJSON LineString
	Score       *float64      `json:"score"`
	Details     *ScoreDetails `json:"details"`
	SubmittedAt *time.Time    `json:"submitted_at"`
}

// ScoreDetails is the per-component breakdown produced when scoring a route.
type ScoreDetails struct {
	TotalPoints       int     `json:"total_points"`
	PointsInCorridor  int     `json:"points_in_corridor"`
	PercentInCorridor float64 `json:"percent_in_corridor"`
	RouteLengthKm     float64 `json:"route_length_km"`
	MaxDeviationM     float64 `json:"max_deviation_m"`
	ConnectsStart     bool    `json:"connects_start"`
	ConnectsEnd       bool    `json:"connects_end"`
	PointsInNoGoZone  int     `json:"points_in_no_go_zone"`
	NoGoZonePenalty   float64 `json:"no_go_zone_penalty"`
	FinalScore        float64 `json:"final_score"`
}

// ToJSON encodes the score details as a JSON string for database storage.
func (sd ScoreDetails) ToJSON() string {
	b, _ := json.Marshal(sd)
	return string(b)
}

// ScoreDetailsFromJSON decodes a JSON-encoded ScoreDetails. Returns (nil, nil)
// when s is empty.
func ScoreDetailsFromJSON(s string) (*ScoreDetails, error) {
	if s == "" {
		return nil, nil
	}
	var sd ScoreDetails
	if err := json.Unmarshal([]byte(s), &sd); err != nil {
		return nil, fmt.Errorf("decode score details: %w", err)
	}
	return &sd, nil
}
