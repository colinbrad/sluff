// Package handler implements the HTTP handlers for the sluff API: auth,
// guide-owned map and round CRUD, session and team management, game flow,
// and the WebSocket upgrade endpoint.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/colinbradley/sluff/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("response encode failed", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// safeMarshal is an alias for model.SafeMarshal kept for call-site brevity.
func safeMarshal(v any) json.RawMessage { return model.SafeMarshal(v) }
