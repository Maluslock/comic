package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Maluslock/comic/server/internal/config"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Connect to database
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := repository.New(pool)
	syncer := service.NewEventSyncer(queries, cfg)

	fmt.Println("Starting event sync from allcpp.cn...")
	start := time.Now()

	result, err := syncer.Sync(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sync complete in %v\n", time.Since(start))
	fmt.Printf("Total events: %d\n", result.Total)
	fmt.Printf("Synced: %d\n", result.New)
	fmt.Printf("Errors: %d\n", result.Errors)
}
