package main

import (
	"context"
	"log"
	"os"

	"mcp-enterprise-registry/internal/db"
	"mcp-enterprise-registry/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	pool, err := db.New(ctx)
	if err != nil {
		log.Printf("warning: failed to init db: %v", err)
	}
	defer func() {
		if pool != nil {
			pool.Close()
		}
	}()

	srv := server.New(ctx, pool, log.Default())
	if err := srv.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}
