// Command sluff-server is the HTTP entry point for the sluff backend.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/colinbradley/sluff/internal/config"
	"github.com/colinbradley/sluff/internal/server"
	"github.com/colinbradley/sluff/internal/store"
)

func main() {
	installLogger()

	cfg := config.Load()

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o750); err != nil {
		slog.Error("create data directory failed", "err", err)
		os.Exit(1)
	}

	db, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		slog.Error("initialize database failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.New(ctx, db, cfg)
	if err := srv.Start(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// installLogger sets the default slog handler to JSON in production
// (HOST=0.0.0.0) and human-readable text otherwise.
func installLogger() {
	var h slog.Handler
	if os.Getenv("HOST") == "0.0.0.0" {
		h = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		h = slog.NewTextHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(h))
}
