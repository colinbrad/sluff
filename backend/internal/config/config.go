package config

import (
	"os"
	"strings"
)

type Config struct {
	Port        string
	DBPath      string
	CORSOrigins []string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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

	return &Config{
		Port:        port,
		DBPath:      dbPath,
		CORSOrigins: corsOrigins,
	}
}
