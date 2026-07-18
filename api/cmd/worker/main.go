package main

import (
	"log/slog"
	"time"
)

func main() {
	slog.Info("P2B worker ready", "queue", "postgres", "poll_interval", "2s")
	for {
		time.Sleep(30 * time.Second)
		slog.Info("worker heartbeat")
	}
}
