// Command oneshot runs a single escrow background task once, without waiting for
// the cron schedule. It exists so ops can verify a timer end-to-end after a fix.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

func main() {
	task := flag.String("task", "noshow", "task to run once: noshow|autoaccept|autocancel")
	flag.Parse()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env("DB_USER", "comic"), env("DB_PASSWORD", "comic123"),
		env("DB_HOST", "127.0.0.1"), env("DB_PORT", "5433"), env("DB_NAME", "comic"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	queries := repository.New(pool)
	cron := service.NewCronService(queries)

	switch *task {
	case "noshow":
		n, err := cron.AutoNoShow(ctx)
		report("noshow", n, err)
	case "autoaccept":
		err := cron.AutoAcceptDeliveries(ctx)
		report("autoaccept", 0, err)
	case "autocancel":
		n, err := cron.AutoCancelPending(ctx)
		report("autocancel", n, err)
	default:
		log.Fatalf("unknown task %q", *task)
	}
}

func report(name string, n int, err error) {
	if err != nil {
		log.Fatalf("[oneshot:%s] FAILED: %v", name, err)
	}
	log.Printf("[oneshot:%s] ok, processed=%d", name, n)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
