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

// SessionHandler implements session and team management endpoints.
type SessionHandler struct {
	store *store.SQLiteStore
}

// NewSessionHandler constructs a SessionHandler backed by the given store.
func NewSessionHandler(s *store.SQLiteStore) *SessionHandler {
	return &SessionHandler{store: s}
}

// CreateSession creates a multiplayer session for an authenticated guide's map.
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

// GetSession returns a session by ID.
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

// GetSessionByCode returns a session by its short join code.
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

// JoinSession registers a new player in a waiting session.
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

// CreateTeam creates a new team within a session, up to a maximum of four.
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

// JoinTeam assigns an existing player to a team.
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

// CreateSoloSession creates a single-player session with one team and one player
// pre-created for the requesting guide.
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

	h.createSoloLike(w, &model.Session{
		MapID:        m.ID,
		GuideID:      middleware.GuideIDFromContext(r.Context()),
		Phase:        model.PhaseWaiting,
		CurrentRound: 0,
		TimeLimitSec: timeLimitSec,
	}, "Solo", req.PlayerName)
}

// CreateDemoSession creates an unauthenticated single-player session against
// the first available map, used by the public demo flow.
func (h *SessionHandler) CreateDemoSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerName string `json:"player_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PlayerName == "" {
		writeError(w, http.StatusBadRequest, "player_name is required")
		return
	}

	maps, err := h.store.ListAllMaps()
	if err != nil || len(maps) == 0 {
		writeError(w, http.StatusServiceUnavailable, "no maps available for demo")
		return
	}
	m := maps[0]
	if len(m.Rounds) == 0 {
		writeError(w, http.StatusServiceUnavailable, "demo map has no rounds configured")
		return
	}

	h.createSoloLike(w, &model.Session{
		MapID:        m.ID,
		GuideID:      "", // demo: no guide owner
		Phase:        model.PhasePlaying,
		CurrentRound: 1,
		TimeLimitSec: 300,
	}, "You", req.PlayerName)
}

// createSoloLike persists sess (filling in ID/Code/IsSolo/CreatedAt), then
// creates a single team and player, and writes the trio as the response.
func (h *SessionHandler) createSoloLike(w http.ResponseWriter, sess *model.Session, teamName, playerName string) {
	sess.ID = uuid.New().String()
	sess.Code = generateCode(6)
	sess.IsSolo = true
	sess.CreatedAt = time.Now()

	if err := h.store.CreateSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	team := &model.Team{
		ID:        uuid.New().String(),
		SessionID: sess.ID,
		Name:      teamName,
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
		Name:      playerName,
	}
	if err := h.store.CreatePlayer(player); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create player")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
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
