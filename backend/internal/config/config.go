// Package config reads runtime configuration from environment variables.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Host        string
	Port        string
	DBPath      string
	CORSOrigins []string
	JWTSecret   string
}

// Load reads PORT, DB_PATH, CORS_ORIGINS, and JWT_SECRET from the environment,
// applying defaults for the first three. If JWT_SECRET is unset in production
// (HOST=0.0.0.0) the process exits; in development a random ephemeral secret
// is generated.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Default to loopback so the many unauthenticated endpoints (player join,
	// route submit, demo, WebSocket) are not exposed to the local network.
	// Render sets HOST=0.0.0.0 so the container accepts external traffic.
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/sluff.db"
	}

	corsOrigins := []string{"http://localhost:5173", "http://localhost:*"}
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		corsOrigins = strings.Split(env, ",")
		for i := range corsOrigins {
			corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if os.Getenv("HOST") == "0.0.0.0" {
			slog.Error("JWT_SECRET must be set when HOST=0.0.0.0 (production)")
			os.Exit(1)
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			slog.Error("failed to generate random JWT secret", "err", err)
			os.Exit(1)
		}
		jwtSecret = base64.StdEncoding.EncodeToString(b)
		slog.Warn("JWT_SECRET not set; using ephemeral secret (sessions will not survive restart)")
	}

	return &Config{
		Host:        host,
		Port:        port,
		DBPath:      dbPath,
		CORSOrigins: corsOrigins,
		JWTSecret:   jwtSecret,
	}
}
