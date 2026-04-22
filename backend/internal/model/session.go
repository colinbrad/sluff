package model

import (
	"encoding/json"
	"time"
)

type GamePhase string

const (
	PhaseWaiting  GamePhase = "waiting"
	PhasePlaying  GamePhase = "playing"
	PhaseScoring  GamePhase = "scoring"
	PhaseFinished GamePhase = "finished"
)

type Session struct {
	ID           string    `json:"id"`
	MapID        string    `json:"map_id"`
	Code         string    `json:"code"`
	Phase        GamePhase `json:"phase"`
	CurrentRound int       `json:"current_round"`
	TimeLimitSec int       `json:"time_limit_sec"`
	IsSolo       bool      `json:"is_solo"`
	CreatedAt    time.Time `json:"created_at"`
	Teams        []Team    `json:"teams,omitempty"`
	Players      []Player  `json:"players,omitempty"`
}

type Team struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

type Player struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	TeamID    string `json:"team_id"`
	Name      string `json:"name"`
	IsOnline  bool   `json:"is_online"`
}

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

func (sd ScoreDetails) ToJSON() string {
	b, _ := json.Marshal(sd)
	return string(b)
}

func ScoreDetailsFromJSON(s string) (*ScoreDetails, error) {
	if s == "" {
		return nil, nil
	}
	var sd ScoreDetails
	err := json.Unmarshal([]byte(s), &sd)
	if err != nil {
		return nil, err
	}
	return &sd, nil
}
