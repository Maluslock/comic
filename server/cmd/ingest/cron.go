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

	appconfig "github.com/Maluslock/comic/server/internal/config"
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
	// 与 cmd/ingest/main.go 同一口径：按 end_date 判定「已结束」，
	// 这样跨过开场时间但仍在办的多日漫展不会被一并软删。
	res, err := pool.Exec(ctx, "UPDATE comic_events SET del_flag = true WHERE end_date < NOW() AND del_flag = false")
	if err != nil {
		log.Printf("expire-cleanup: %v", err)
		return
	}
	log.Printf("expire-cleanup: marked %d expired events as deleted", res.RowsAffected())
}

// runEventsIngest 跑一轮多源编排（eventsource 适配器 + EventIngestService）。
//
// 与老的 nyato 抓取是两条并存的路径：nyato 走 internal/ingest 的 HTML 抓取，
// 这里走结构化接口（allcpp / bilibili），两者写同一张 comic_events，靠
// allcpp_id 唯一键 upsert，不会互相覆盖。每轮都会往 event_source_runs 写运行日志。
//
// 老入口（cmd/ingest -config config/cron.yaml）能继续用，只是把 source: events
// 的任务派发到新编排器上；新代码（cmd/events-ingest）是同一套服务的独立常驻进程。
func runEventsIngest(ctx context.Context, pool *pgxpool.Pool) {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Printf("task events: 加载配置失败: %v", err)
		return
	}
	svc := service.NewEventIngestService(repository.New(pool), repository.NewEventSourcesRepo(pool), cfg).
		WithPool(pool)
	report, err := svc.IngestAll(ctx)
	if report != nil {
		log.Printf("task events: %s", report.Line())
	}
	if err != nil {
		log.Printf("task events: 本轮失败: %v", err)
	}
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
			case "events":
				// 多源编排一轮（allcpp → bilibili，单源失败自动跳过继续）。
				// 抓取+封面下载+upsert 可能几分钟，给 20 分钟上限。
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
				defer cancel()
				runEventsIngest(ctx, pool)
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
