package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/p2b/p2b/internal/httpapi"
	"github.com/p2b/p2b/internal/platform"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	server := &http.Server{
		Addr:              address,
		Handler:           httpapi.NewServerWithConfig(platform.NewDemoService(), httpapi.EnvConfig()),
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
