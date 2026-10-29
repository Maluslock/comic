package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

type TaskConfig struct {
	Name     string `yaml:"name"`
	Schedule string `yaml:"schedule"`
	Source   string `yaml:"source"`
	Pages    int    `yaml:"pages"`
	URL      string `yaml:"url"`
	Enabled  bool   `yaml:"enabled"`
}

type CronConfig struct {
	Tasks []TaskConfig `yaml:"tasks"`
}

func LoadCronConfig(path string) (*CronConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg CronConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

func openPool() (*pgxpool.Pool, error) {
	databaseURL := *dsn
	if databaseURL == "" {
		databaseURL = buildDSN()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func markExpired(pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := pool.Exec(ctx, "UPDATE comic_events SET del_flag = true WHERE start_date < NOW() AND del_flag = false")
	if err != nil {
		log.Printf("expire-cleanup: %v", err)
		return
	}
	log.Printf("expire-cleanup: marked %d expired events as deleted", res.RowsAffected())
}

func runEscrowCron(ctx context.Context, pool *pgxpool.Pool, name string) {
	start := time.Now()
	queries := repository.New(pool)
	svc := service.NewCronService(queries)
	var err error
	var count int
	switch name {
	case "booking-auto-cancel":
		var cancelCount int
		cancelCount, err = svc.AutoCancelPending(ctx)
		count = cancelCount
	case "booking-auto-accept":
		acceptErr := svc.AutoAcceptDeliveries(ctx)
		if acceptErr == nil {
			count = -1
		}
		err = acceptErr
	case "booking-noshow":
		var noshowCount int
		noshowCount, err = svc.AutoNoShow(ctx)
		count = noshowCount
	case "booking-default-resolve":
		var resolveCount int
		resolveCount, err = svc.DefaultResolveDisputes(ctx)
		count = resolveCount
	default:
		log.Printf("task %q: unknown escrow task", name)
		return
	}
	if err != nil {
		log.Printf("task %q: failed: %v", name, err)
		return
	}
	log.Printf("task %q: processed %d row(s) in %s", name, count, time.Since(start).Round(time.Millisecond))
}

func runCronMode(configPath string) {
	cfg, err := LoadCronConfig(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := openPool()
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	c := cron.New()
	registered := 0
	for _, task := range cfg.Tasks {
		if !task.Enabled {
			log.Printf("task %q: disabled, skipping", task.Name)
			continue
		}
		t := task
		_, err := c.AddFunc(t.Schedule, func() {
			start := time.Now()
			log.Printf("task %q: starting at %s", t.Name, start.Format("2006-01-02 15:04:05"))
			switch t.Source {
			case "expire":
				markExpired(pool)
			case "bilibili":
				runOnce(runParams{url: t.URL, pages: t.Pages, withBili: true})
			case "nyato", "":
				runOnce(runParams{url: t.URL, pages: t.Pages})
			case "escrow":
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				runEscrowCron(ctx, pool, t.Name)
			default:
				log.Printf("task %q: unknown source %q", t.Name, t.Source)
			}
			log.Printf("task %q: done in %s", t.Name, time.Since(start).Round(time.Millisecond))
		})
		if err != nil {
			log.Printf("task %q: bad schedule %q: %v", t.Name, t.Schedule, err)
			continue
		}
		log.Printf("task %q registered: schedule=%q source=%s pages=%d url=%s", t.Name, t.Schedule, t.Source, t.Pages, t.URL)
		registered++
	}

	if registered == 0 {
		log.Fatal("no tasks registered — check cron.yaml schedules and enabled flags")
	}

	c.Start()
	log.Printf("cron scheduler started with %d task(s); Ctrl+C to stop", registered)
	select {}
}
