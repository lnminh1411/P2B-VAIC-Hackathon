package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/httpapi"
	"github.com/p2b/p2b/internal/platform"
	storageadapter "github.com/p2b/p2b/internal/storage"
	"github.com/p2b/p2b/internal/tenancy"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	config, err := httpapi.EnvConfig()
	if err != nil {
		slog.Error("invalid API configuration", "error", err)
		os.Exit(1)
	}
	var database *pgxpool.Pool
	if !config.DevAuth {
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			slog.Error("invalid API configuration", "error", "DATABASE_URL is required")
			os.Exit(1)
		}
		poolConfig, parseErr := pgxpool.ParseConfig(databaseURL)
		if parseErr != nil {
			slog.Error("invalid DATABASE_URL", "error", parseErr)
			os.Exit(1)
		}
		poolConfig.MaxConns = 10
		poolConfig.MinConns = 1
		poolConfig.MaxConnLifetime = 30 * time.Minute
		poolConfig.MaxConnIdleTime = 5 * time.Minute
		database, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			slog.Error("database pool failed", "error", err)
			os.Exit(1)
		}
		defer database.Close()
		pingContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = database.Ping(pingContext); err != nil {
			slog.Error("database unavailable", "error", err)
			os.Exit(1)
		}
		config.WorkspaceBootstrapper = tenancy.NewBootstrapper(database)
		config.ReadinessChecker = database
		uploadSigner, signerErr := storageadapter.NewSupabaseSigner(
			os.Getenv("SUPABASE_URL"),
			os.Getenv("SUPABASE_SECRET_KEY"),
			env("SUPABASE_STORAGE_BUCKET", "p2b-private"),
			nil,
		)
		if signerErr != nil {
			slog.Error("storage configuration failed", "error", signerErr)
			os.Exit(1)
		}
		config.UploadSigner = uploadSigner
	}
	server := &http.Server{
		Addr:              address,
		Handler:           httpapi.NewServerWithConfig(platform.NewDemoService(), config),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	slog.Info("P2B API listening", "address", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
