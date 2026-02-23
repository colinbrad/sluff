package handler

import (
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

type WSHandler struct {
	store store.Store
	hub   *ws.Hub
}

func NewWSHandler(s store.Store, hub *ws.Hub) *WSHandler {
	return &WSHandler{store: s, hub: hub}
}

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
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, sessionID, playerID, player.TeamID)
	h.hub.Register(client)

	ctx := r.Context()
	go client.WritePump(ctx)
	go client.ReadPump(ctx)
}
