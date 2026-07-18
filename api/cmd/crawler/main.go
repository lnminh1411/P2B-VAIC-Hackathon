package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/crawler"
	"github.com/p2b/p2b/internal/pipeline"
)

func main() {
	slog.Info("P2B crawler scheduler starting...")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("crawler configuration invalid: DATABASE_URL is required")
		os.Exit(1)
	}

	database, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("crawler database connection failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err = database.Ping(ctx); err != nil {
		slog.Error("crawler database unavailable", "error", err)
		os.Exit(1)
	}
	slog.Info("P2B crawler scheduler connected to database")

	store := pipeline.NewStore(database)

	// Run crawler pass immediately on startup so the system loads initial alerts
	crawler.RunCrawler(ctx, database, store)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("P2B crawler scheduler stopped")
			return
		case <-ticker.C:
			crawler.RunCrawler(ctx, database, store)
		}
	}
}
