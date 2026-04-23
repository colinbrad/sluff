package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/coder/websocket"

	"github.com/colinbradley/sluff/internal/model"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	maxMsgSize = 64 * 1024 // 64KB
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	SessionID string
	PlayerID  string
	TeamID    string
}

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

func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(maxMsgSize)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Printf("WebSocket closed for player %s: %v", c.PlayerID, err)
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

func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "")
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
		// Broadcast cursor to teammates
		c.hub.BroadcastToTeam(c.SessionID, c.TeamID, model.WSMessage{
			Type: model.MsgCursorUpdate,
			Payload: mustMarshal(model.CursorUpdatePayload{
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
		// Broadcast drawing to teammates
		c.hub.BroadcastToTeam(c.SessionID, c.TeamID, model.WSMessage{
			Type:    model.MsgDrawingUpdate,
			Payload: mustMarshal(payload),
		}, c.PlayerID)

	case model.MsgPing:
		// No-op, connection activity is enough
	}
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
