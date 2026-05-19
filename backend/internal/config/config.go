package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
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
		b := make([]byte, 32)
		rand.Read(b)
		jwtSecret = base64.StdEncoding.EncodeToString(b)
		log.Println("WARNING: JWT_SECRET not set — using ephemeral secret (sessions won't survive restart)")
	}

	return &Config{
		Host:        host,
		Port:        port,
		DBPath:      dbPath,
		CORSOrigins: corsOrigins,
		JWTSecret:   jwtSecret,
	}
}
