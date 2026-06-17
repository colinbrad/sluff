package handler

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

// WSHandler implements the WebSocket upgrade endpoint, validating the player
// against the session and starting the read and write goroutines.
type WSHandler struct {
	store *store.SQLiteStore
	hub   *ws.Hub
}

// NewWSHandler constructs a WSHandler backed by the given store and hub.
func NewWSHandler(s *store.SQLiteStore, hub *ws.Hub) *WSHandler {
	return &WSHandler{store: s, hub: hub}
}

// HandleWebSocket upgrades the connection and binds it to the player and session.
func (h *WSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	playerID := r.URL.Query().Get("player_id")

	if playerID == "" {
		writeError(w, http.StatusBadRequest, "player_id query parameter required")
		return
	}

	// Validate player exists in session
	player, err := h.store.GetPlayer(playerID)
	if err != nil || player == nil || player.SessionID != sessionID {
		writeError(w, http.StatusNotFound, "player not found in this session")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*"},
	})
	if err != nil {
		slog.Error("websocket accept failed", "err", err)
		return
	}

	client := ws.NewClient(h.hub, conn, sessionID, playerID, player.TeamID)
	h.hub.Register(client)

	ctx := r.Context()
	go client.WritePump(ctx)
	go client.ReadPump(ctx)
}
