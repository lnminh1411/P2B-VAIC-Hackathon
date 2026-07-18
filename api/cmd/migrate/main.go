package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}
	connection, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect database:", err)
		os.Exit(1)
	}
	defer connection.Close(context.Background())
	fmt.Println("Database connection ready. Apply api/migrations in lexical order in deployment job.")
}
