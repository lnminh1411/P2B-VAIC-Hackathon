package main

import (
	"log/slog"
	"time"
)

func main() {
	slog.Info("P2B crawler scheduler ready", "policy", "review-before-publish")
	for {
		time.Sleep(6 * time.Hour)
		slog.Info("crawler tick", "mode", "allowlist-only")
	}
}
