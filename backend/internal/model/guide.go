// Package model defines the core domain types: guides, maps, rounds, sessions,
// teams, players, and the WebSocket message envelope shared by client and server.
package model

import "time"

// Guide is an authenticated user who owns maps and runs game sessions.
type Guide struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
