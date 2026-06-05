package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/colinbradley/sluff/internal/middleware"
	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

// GuideAdminHandler implements guide-only moderation endpoints: kick player,
// clear a team's submitted route.
type GuideAdminHandler struct {
	store store.Store
	hub   *ws.Hub
}

// NewGuideAdminHandler constructs a GuideAdminHandler backed by the given
// store and hub.
func NewGuideAdminHandler(s store.Store, hub *ws.Hub) *GuideAdminHandler {
	return &GuideAdminHandler{store: s, hub: hub}
}

// KickPlayer removes a player from the session and broadcasts their departure.
func (h *GuideAdminHandler) KickPlayer(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	playerID := chi.URLParam(r, "playerID")
	guideID := middleware.GuideIDFromContext(r.Context())

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.GuideID != guideID {
		writeError(w, http.StatusForbidden, "not your session")
		return
	}

	if err := h.store.DeletePlayer(playerID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to kick player")
		return
	}

	h.hub.BroadcastToSession(sessionID, model.WSMessage{
		Type:    "player_left",
		Payload: safeMarshal(map[string]string{"player_id": playerID}),
	})

	w.WriteHeader(http.StatusNoContent)
}

// ClearRoute deletes a team's route submission for the current round, allowing
// them to resubmit.
func (h *GuideAdminHandler) ClearRoute(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	roundID := chi.URLParam(r, "roundID")
	teamID := chi.URLParam(r, "teamID")
	guideID := middleware.GuideIDFromContext(r.Context())

	sess, err := h.store.GetSession(sessionID)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.GuideID != guideID {
		writeError(w, http.StatusForbidden, "not your session")
		return
	}

	if err := h.store.DeleteTeamRoute(roundID, teamID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear route")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
