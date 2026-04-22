package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/colinbradley/sluff/internal/game"
	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

type GameHandler struct {
	store store.Store
	hub   *ws.Hub
}

func NewGameHandler(s store.Store, hub *ws.Hub) *GameHandler {
	return &GameHandler{store: s, hub: hub}
}

func (h *GameHandler) StartGame(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	allowedPhase := sess.Phase == model.PhaseWaiting || sess.Phase == model.PhaseScoring
	if sess.IsSolo {
		allowedPhase = allowedPhase || sess.Phase == model.PhasePlaying
	}
	if !allowedPhase {
		writeError(w, http.StatusBadRequest, "game cannot be started from current phase")
		return
	}

	// Get rounds for the map
	rounds, err := h.store.GetRoundsByMap(sess.MapID)
	if err != nil || len(rounds) == 0 {
		writeError(w, http.StatusBadRequest, "map has no rounds configured")
		return
	}

	// Check teams have players
	teams, err := h.store.GetTeamsBySession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check teams")
		return
	}
	minTeams := 2
	if sess.IsSolo {
		minTeams = 1
	}
	if len(teams) < minTeams {
		writeError(w, http.StatusBadRequest, "not enough teams")
		return
	}

	// Advance to next round
	nextRound := sess.CurrentRound + 1
	if nextRound > len(rounds) {
		sess.Phase = model.PhaseFinished
		sess.CurrentRound = len(rounds)
		h.store.UpdateSession(sess)
		writeJSON(w, http.StatusOK, sess)
		return
	}

	sess.Phase = model.PhasePlaying
	sess.CurrentRound = nextRound
	if err := h.store.UpdateSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	// Broadcast round start to all connected players.
	round := rounds[nextRound-1]
	payload := roundToGeoJSON(&round)

	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type:    model.MsgRoundStart,
		Payload: mustMarshal(payload),
	})
	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type: model.MsgGameState,
		Payload: mustMarshal(model.GameStatePayload{
			Phase:         model.PhasePlaying,
			CurrentRound:  nextRound,
			TimeRemaining: sess.TimeLimitSec,
		}),
	})

	writeJSON(w, http.StatusOK, sess)
}

func (h *GameHandler) SubmitRoute(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	roundID := chi.URLParam(r, "roundID")

	var req struct {
		TeamID string          `json:"team_id"`
		Path   json.RawMessage `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get the round for scoring
	round, err := h.store.GetRound(roundID)
	if err != nil || round == nil {
		writeError(w, http.StatusNotFound, "round not found")
		return
	}

	now := time.Now()
	route := &model.TeamRoute{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		RoundID:     roundID,
		TeamID:      req.TeamID,
		Path:        string(req.Path),
		SubmittedAt: &now,
	}

	// Score the route
	details, err := game.ScoreRoute(string(req.Path), round)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to score route: "+err.Error())
		return
	}

	route.Score = &details.FinalScore
	route.Details = &details

	if err := h.store.CreateTeamRoute(route); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save route")
		return
	}

	writeJSON(w, http.StatusCreated, route)
}

func (h *GameHandler) GetScores(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")

	routes, err := h.store.GetRoutesByRound(roundID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get scores")
		return
	}
	if routes == nil {
		routes = []model.TeamRoute{}
	}

	writeJSON(w, http.StatusOK, routes)
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
