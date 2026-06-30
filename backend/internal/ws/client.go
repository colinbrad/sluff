// Package ws implements the WebSocket hub and per-connection client used for
// real-time cursor and drawing updates between teammates within a session.
package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/coder/websocket"

	"github.com/colinbradley/sluff/internal/model"
)

const (
	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second
	maxMsgSize = 64 * 1024 // 64KB
)

// Client is a single WebSocket connection associated with a player in a session.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	SessionID string
	PlayerID  string
	TeamID    string
}

// NewClient wraps a websocket.Conn for a player within a session and team.
func NewClient(hub *Hub, conn *websocket.Conn, sessionID, playerID, teamID string) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		SessionID: sessionID,
		PlayerID:  playerID,
		TeamID:    teamID,
	}
}

// ReadPump reads inbound WebSocket messages until ctx is cancelled or the peer
// closes. It runs as a long-lived goroutine started by the connection handler.
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(maxMsgSize)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				slog.Info("websocket closed", "player_id", c.PlayerID, "err", err)
			}
			return
		}

		var msg model.WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}
}

// WritePump writes outbound messages and periodic pings until ctx is cancelled
// or a write fails.
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()
			if err != nil {
				return
			}

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) handleMessage(msg model.WSMessage) {
	switch msg.Type {
	case model.MsgCursorMove:
		var payload model.CursorMovePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		c.hub.BroadcastToTeam(c.SessionID, c.TeamID, model.WSMessage{
			Type: model.MsgCursorUpdate,
			Payload: model.MustMarshal(model.CursorUpdatePayload{
				PlayerID: c.PlayerID,
				Lat:      payload.Lat,
				Lng:      payload.Lng,
			}),
		}, c.PlayerID)

	case model.MsgDrawingUpdate:
		var payload model.DrawingUpdatePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		payload.PlayerID = c.PlayerID
		payload.TeamID = c.TeamID
		c.hub.BroadcastToTeam(c.SessionID, c.TeamID, model.WSMessage{
			Type:    model.MsgDrawingUpdate,
			Payload: model.MustMarshal(payload),
		}, c.PlayerID)

	case model.MsgPing:
		// No-op, connection activity is enough
	}
}
