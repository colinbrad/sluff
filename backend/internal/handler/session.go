package handler

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/colinbradley/sluff/internal/middleware"
	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
)

type SessionHandler struct {
	store store.Store
}

func NewSessionHandler(s store.Store) *SessionHandler {
	return &SessionHandler{store: s}
}

func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MapID        string `json:"map_id"`
		TimeLimitSec int    `json:"time_limit_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MapID == "" {
		writeError(w, http.StatusBadRequest, "map_id is required")
		return
	}

	// Verify map exists
	m, err := h.store.GetMap(req.MapID)
	if err != nil || m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}

	timeLimitSec := req.TimeLimitSec
	if timeLimitSec <= 0 {
		timeLimitSec = 300
	}

	sess := &model.Session{
		ID:           uuid.New().String(),
		MapID:        req.MapID,
		GuideID:      middleware.GuideIDFromContext(r.Context()),
		Code:         generateCode(6),
		Phase:        model.PhaseWaiting,
		CurrentRound: 0,
		TimeLimitSec: timeLimitSec,
		CreatedAt:    time.Now(),
	}

	if err := h.store.CreateSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusCreated, sess)
}

func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sessionID")
	sess, err := h.store.GetSession(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *SessionHandler) GetSessionByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	sess, err := h.store.GetSessionByCode(code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *SessionHandler) JoinSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if sess.Phase != model.PhaseWaiting {
		writeError(w, http.StatusBadRequest, "game has already started")
		return
	}

	// Check player limit
	players, err := h.store.GetPlayersBySession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check players")
		return
	}
	if len(players) >= 8 {
		writeError(w, http.StatusBadRequest, "game is full (max 8 players)")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	player := &model.Player{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Name:      req.Name,
	}

	if err := h.store.CreatePlayer(player); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create player")
		return
	}

	writeJSON(w, http.StatusCreated, player)
}

func (h *SessionHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	teams, err := h.store.GetTeamsBySession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check teams")
		return
	}
	if len(teams) >= 4 {
		writeError(w, http.StatusBadRequest, "max 4 teams allowed")
		return
	}

	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Color == "" {
		req.Color = "#3B82F6" // default blue
	}

	team := &model.Team{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Name:      req.Name,
		Color:     req.Color,
	}

	if err := h.store.CreateTeam(team); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create team")
		return
	}

	writeJSON(w, http.StatusCreated, team)
}

func (h *SessionHandler) JoinTeam(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")

	var req struct {
		PlayerID string `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PlayerID == "" {
		writeError(w, http.StatusBadRequest, "player_id is required")
		return
	}

	if err := h.store.UpdatePlayerTeam(req.PlayerID, teamID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to join team")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h *SessionHandler) CreateSoloSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MapID        string `json:"map_id"`
		TimeLimitSec int    `json:"time_limit_sec"`
		PlayerName   string `json:"player_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MapID == "" || req.PlayerName == "" {
		writeError(w, http.StatusBadRequest, "map_id and player_name are required")
		return
	}

	m, err := h.store.GetMap(req.MapID)
	if err != nil || m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}

	timeLimitSec := req.TimeLimitSec
	if timeLimitSec <= 0 {
		timeLimitSec = 300
	}

	sess := &model.Session{
		ID:           uuid.New().String(),
		MapID:        req.MapID,
		GuideID:      middleware.GuideIDFromContext(r.Context()),
		Code:         generateCode(6),
		Phase:        model.PhaseWaiting,
		CurrentRound: 0,
		TimeLimitSec: timeLimitSec,
		IsSolo:       true,
		CreatedAt:    time.Now(),
	}
	if err := h.store.CreateSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	team := &model.Team{
		ID:        uuid.New().String(),
		SessionID: sess.ID,
		Name:      "Solo",
		Color:     "#3B82F6",
	}
	if err := h.store.CreateTeam(team); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create team")
		return
	}

	player := &model.Player{
		ID:        uuid.New().String(),
		SessionID: sess.ID,
		TeamID:    team.ID,
		Name:      req.PlayerName,
	}
	if err := h.store.CreatePlayer(player); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create player")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"session": sess,
		"player":  player,
		"team":    team,
	})
}

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I/O/0/1 to avoid confusion

func generateCode(length int) string {
	code := make([]byte, length)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		code[i] = codeChars[n.Int64()]
	}
	return string(code)
}
