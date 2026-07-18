package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/extraction"
	"github.com/p2b/p2b/internal/pipeline"
	storageadapter "github.com/p2b/p2b/internal/storage"
	workerprocessor "github.com/p2b/p2b/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("worker configuration invalid", "error", "DATABASE_URL is required")
		os.Exit(1)
	}
	database, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("worker database connection failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	if err = database.Ping(ctx); err != nil {
		slog.Error("worker database unavailable", "error", err)
		os.Exit(1)
	}
	storage, err := storageadapter.NewSupabaseSigner(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SECRET_KEY"), env("SUPABASE_STORAGE_BUCKET", "p2b-private"), nil)
	if err != nil {
		slog.Error("worker storage configuration invalid", "error", err)
		os.Exit(1)
	}
	model := env("GEMINI_MODEL", extraction.GeminiStableModel)
	gemini, err := extraction.NewGeminiExtractor(os.Getenv("GEMINI_API_KEY"), model, "", nil)
	if err != nil {
		slog.Error("worker Gemini configuration invalid", "error", err)
		os.Exit(1)
	}
	store := pipeline.NewStore(database)
	processor := workerprocessor.Processor{
		Store: store, Downloader: storage,
		Converter: extraction.MarkItDownConverter{Executable: env("MARKITDOWN_BIN", "markitdown")},
		Extractor: gemini, Model: model,
	}
	slog.Info("P2B extraction worker ready", "queue", "postgres", "model", model, "concurrency", 1)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("P2B extraction worker stopped")
			return
		case <-ticker.C:
			job, claimErr := store.Claim(ctx)
			if errors.Is(claimErr, pipeline.ErrNotFound) {
				continue
			}
			if claimErr != nil {
				slog.Error("claim extraction job failed", "error", claimErr)
				continue
			}
			jobContext, cancel := context.WithTimeout(ctx, 12*time.Minute)
			processErr := processor.Process(jobContext, job)
			cancel()
			if processErr != nil {
				slog.Error("extraction job failed", "job_id", job.ID, "attempt", job.Attempts, "error", processErr)
				continue
			}
			slog.Info("extraction job succeeded", "job_id", job.ID)
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
