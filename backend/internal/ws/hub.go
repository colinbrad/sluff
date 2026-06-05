package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/colinbradley/sluff/internal/model"
)

// Hub manages WebSocket connections grouped by session.
type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[string]*Client // sessionID -> playerID -> client
	register   chan *Client
	unregister chan *Client
}

// NewHub constructs an empty Hub. Call Run in a goroutine to start dispatching.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run processes register and unregister events until the program exits.
// It is intended to be called once in a dedicated goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.SessionID] == nil {
				h.rooms[client.SessionID] = make(map[string]*Client)
			}
			h.rooms[client.SessionID][client.PlayerID] = client
			h.mu.Unlock()
			slog.Info("player joined", "player_id", client.PlayerID, "session_id", client.SessionID)

		case client := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.SessionID]; ok {
				// Only evict if this *Client is still the registered one for
				// the player. If the player reconnected, a newer *Client
				// occupies the slot and must not be removed when an older
				// goroutine's defer fires this Unregister.
				if existing, ok := room[client.PlayerID]; ok && existing == client {
					delete(room, client.PlayerID)
					close(client.send)
					if len(room) == 0 {
						delete(h.rooms, client.SessionID)
					}
				}
			}
			h.mu.Unlock()
			slog.Info("player left", "player_id", client.PlayerID, "session_id", client.SessionID)
		}
	}
}

// Register enqueues a client for registration. Safe for concurrent use.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister enqueues a client for removal. Safe for concurrent use.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// BroadcastToSession sends a message to all clients in a session.
func (h *Hub) BroadcastToSession(sessionID string, msg model.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.rooms[sessionID]
	if !ok {
		return
	}

	for _, client := range room {
		select {
		case client.send <- data:
		default:
			// Client buffer full, skip
		}
	}
}

// BroadcastToTeam sends a message to all clients in a team within a session.
func (h *Hub) BroadcastToTeam(sessionID, teamID string, msg model.WSMessage, excludePlayerID string) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.rooms[sessionID]
	if !ok {
		return
	}

	for _, client := range room {
		if client.TeamID == teamID && client.PlayerID != excludePlayerID {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}
