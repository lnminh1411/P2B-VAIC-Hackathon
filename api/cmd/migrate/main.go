package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/p2b/p2b/migrations"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	connection, err := pgx.Connect(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect database:", err)
		os.Exit(1)
	}
	defer connection.Close(context.Background())
	if err = migrations.Run(ctx, connection); err != nil {
		fmt.Fprintln(os.Stderr, "apply migrations:", err)
		os.Exit(1)
	}
	fmt.Println("Database migrations applied successfully.")
}
