package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/colinbradley/sluff/internal/config"
	"github.com/colinbradley/sluff/internal/server"
	"github.com/colinbradley/sluff/internal/store"
)

func main() {
	cfg := config.Load()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	db, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	srv := server.New(db, cfg)
	log.Fatal(srv.Start(cfg.Host + ":" + cfg.Port))
}
