package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/colinbradley/sluff/internal/game"
	"github.com/colinbradley/sluff/internal/middleware"
	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

// GameHandler implements endpoints that advance game flow: starting rounds,
// submitting routes, and fetching scores.
type GameHandler struct {
	store *store.SQLiteStore
	hub   *ws.Hub
}

// NewGameHandler constructs a GameHandler backed by the given store and hub.
func NewGameHandler(s *store.SQLiteStore, hub *ws.Hub) *GameHandler {
	return &GameHandler{store: s, hub: hub}
}

// StartGame advances the session to its next round (or to PhaseFinished if the
// last round is complete) and broadcasts the new round to all connected clients.
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

	// sess.Teams was already loaded by GetSession; no need for a second query.
	minTeams := 2
	if sess.IsSolo {
		minTeams = 1
	}
	if len(sess.Teams) < minTeams {
		writeError(w, http.StatusBadRequest, "not enough teams")
		return
	}

	// Advance to next round
	nextRound := sess.CurrentRound + 1
	if nextRound > len(rounds) {
		sess.Phase = model.PhaseFinished
		sess.CurrentRound = len(rounds)
		if err := h.store.UpdateSession(sess); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update session")
			return
		}
		// Notify connected players the game is over so they leave the
		// "waiting for guide" state.
		h.hub.BroadcastToSession(sessionID, model.WSMessage{
			Type: model.MsgGameState,
			Payload: model.MustMarshal(model.GameStatePayload{
				Phase:        model.PhaseFinished,
				CurrentRound: sess.CurrentRound,
			}),
		})
		writeJSON(w, http.StatusOK, sess)
		return
	}

	sess.Phase = model.PhasePlaying
	sess.CurrentRound = nextRound
	// Record the authoritative deadline; the ticker ends the round when it passes.
	if err := h.store.StartRound(sessionID, nextRound, time.Now().Add(time.Duration(sess.TimeLimitSec)*time.Second)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	// Broadcast round start to all connected players.
	round := rounds[nextRound-1]
	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type:    model.MsgRoundStart,
		Payload: model.MustMarshal(round),
	})
	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type: model.MsgGameState,
		Payload: model.MustMarshal(model.GameStatePayload{
			Phase:         model.PhasePlaying,
			CurrentRound:  nextRound,
			TimeRemaining: sess.TimeLimitSec,
		}),
	})

	writeJSON(w, http.StatusOK, sess)
}

// SubmitRoute scores a team's GeoJSON LineString against the round's corridor
// and persists the result. Only valid during PhasePlaying; rejects duplicates.
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

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	// Scoring is also accepted so clients can auto-submit their in-progress
	// route when the round ends.
	if sess.Phase != model.PhasePlaying && sess.Phase != model.PhaseScoring {
		writeError(w, http.StatusBadRequest, "session is not accepting submissions")
		return
	}

	round, err := h.store.GetRound(roundID)
	if err != nil || round == nil {
		writeError(w, http.StatusNotFound, "round not found")
		return
	}
	if round.MapID != sess.MapID {
		writeError(w, http.StatusBadRequest, "round does not belong to this session's map")
		return
	}

	// sess.Teams was already loaded by GetSession; no need for a second query.
	validTeam := false
	for _, t := range sess.Teams {
		if t.ID == req.TeamID {
			validTeam = true
			break
		}
	}
	if !validTeam {
		writeError(w, http.StatusForbidden, "team does not belong to this session")
		return
	}

	existing, err := h.store.GetTeamRoute(roundID, req.TeamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check existing route")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "route already submitted for this team and round")
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

	// Multiplayer live-scoring broadcasts. Solo/demo have no WS clients.
	if !sess.IsSolo {
		if sess.Phase == model.PhaseScoring {
			// Round already ended; stream updated scores as each (auto-)submission lands.
			h.broadcastScores(sessionID, sess)
		} else {
			// During play, announce progress without leaking the route, then end the
			// round early once every team with players has submitted.
			h.hub.BroadcastToSession(sessionID, model.WSMessage{
				Type:    model.MsgTeamSubmitted,
				Payload: model.MustMarshal(model.TeamSubmittedPayload{TeamID: req.TeamID}),
			})
			if routes, err := h.store.GetRoutesByRound(roundID); err == nil && allTeamsSubmitted(sess, routes) {
				h.endRound(sessionID)
			}
		}
	}

	writeJSON(w, http.StatusCreated, route)
}

// GetScores returns all submitted team routes (with scores) for a round.
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

// GetCurrentRound returns the round that is currently in progress for a session,
// serialized as GeoJSON for the frontend.
func (h *GameHandler) GetCurrentRound(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.CurrentRound == 0 {
		writeError(w, http.StatusBadRequest, "no round in progress")
		return
	}

	rounds, err := h.store.GetRoundsByMap(sess.MapID)
	if err != nil || len(rounds) < sess.CurrentRound {
		writeError(w, http.StatusNotFound, "round not found")
		return
	}

	writeJSON(w, http.StatusOK, rounds[sess.CurrentRound-1])
}

// DemoNextRound is the public, unauthenticated round advancement endpoint used
// by demo sessions; it does not broadcast over WebSocket.
func (h *GameHandler) DemoNextRound(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if !sess.IsSolo || sess.GuideID != "" {
		writeError(w, http.StatusForbidden, "not a demo session")
		return
	}

	rounds, err := h.store.GetRoundsByMap(sess.MapID)
	if err != nil || len(rounds) == 0 {
		writeError(w, http.StatusBadRequest, "map has no rounds")
		return
	}

	nextRound := sess.CurrentRound + 1
	if nextRound > len(rounds) {
		sess.Phase = model.PhaseFinished
		sess.CurrentRound = len(rounds)
		if err := h.store.UpdateSession(sess); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update session")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": sess, "round": nil})
		return
	}

	sess.Phase = model.PhasePlaying
	sess.CurrentRound = nextRound
	if err := h.store.UpdateSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session": sess,
		"round":   rounds[nextRound-1],
	})
}

// EndRound lets the guide end the current round early and reveal scores.
func (h *GameHandler) EndRound(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.GuideID != middleware.GuideIDFromContext(r.Context()) {
		writeError(w, http.StatusForbidden, "not your session")
		return
	}

	h.endRound(sessionID)
	w.WriteHeader(http.StatusNoContent)
}

// RunRoundTicker ends multiplayer rounds whose deadline has passed. It runs as
// a single goroutine until ctx is cancelled.
func (h *GameHandler) RunRoundTicker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := h.store.ExpiredPlayingSessions(time.Now())
			if err != nil {
				slog.Error("round ticker query failed", "err", err)
				continue
			}
			for _, id := range ids {
				h.endRound(id)
			}
		}
	}
}

// endRound transitions a playing session to scoring and reveals the scores so
// far. Idempotent: the ticker, the guide action, and the final submission can
// all call it, but only the caller that flips the phase row broadcasts.
func (h *GameHandler) endRound(sessionID string) {
	ended, err := h.store.EndRoundIfPlaying(sessionID)
	if err != nil || !ended {
		return
	}
	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		return
	}
	h.hub.BroadcastToSession(sessionID, model.WSMessage{Type: model.MsgRoundEnd})
	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type: model.MsgGameState,
		Payload: model.MustMarshal(model.GameStatePayload{
			Phase:        model.PhaseScoring,
			CurrentRound: sess.CurrentRound,
		}),
	})
	h.broadcastScores(sessionID, sess)
}

// broadcastScores sends the per-team score breakdown for the session's current
// round to every connected client.
func (h *GameHandler) broadcastScores(sessionID string, sess *model.Session) {
	rounds, err := h.store.GetRoundsByMap(sess.MapID)
	if err != nil || sess.CurrentRound < 1 || sess.CurrentRound > len(rounds) {
		return
	}
	routes, err := h.store.GetRoutesByRound(rounds[sess.CurrentRound-1].ID)
	if err != nil {
		return
	}
	entries := make([]model.TeamScoreEntry, 0, len(routes))
	for _, rt := range routes {
		if rt.Details != nil {
			entries = append(entries, model.TeamScoreEntry{TeamID: rt.TeamID, Score: *rt.Details})
		}
	}
	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type:    model.MsgScores,
		Payload: model.MustMarshal(model.ScoresPayload{TeamScores: entries}),
	})
}

// allTeamsSubmitted reports whether every team with at least one player has
// submitted for the round. Empty teams do not block the early round end.
func allTeamsSubmitted(sess *model.Session, routes []model.TeamRoute) bool {
	withPlayers := map[string]bool{}
	for _, p := range sess.Players {
		if p.TeamID != "" {
			withPlayers[p.TeamID] = true
		}
	}
	if len(withPlayers) == 0 {
		return false
	}
	submitted := map[string]bool{}
	for _, rt := range routes {
		submitted[rt.TeamID] = true
	}
	for tid := range withPlayers {
		if !submitted[tid] {
			return false
		}
	}
	return true
}
