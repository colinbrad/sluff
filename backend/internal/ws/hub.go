package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/colinbradley/sluff/internal/model"
)

// Hub manages WebSocket connections grouped by session.
type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[string]*Client // sessionID -> playerID -> client
	register chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

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
			log.Printf("Player %s joined session %s", client.PlayerID, client.SessionID)

		case client := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.SessionID]; ok {
				if _, ok := room[client.PlayerID]; ok {
					delete(room, client.PlayerID)
					close(client.send)
					if len(room) == 0 {
						delete(h.rooms, client.SessionID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Player %s left session %s", client.PlayerID, client.SessionID)
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

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
