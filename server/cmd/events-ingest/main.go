// Command events-ingest 是多源漫展数据入库的独立入口。
//
// 两种用法：
//
//	events-ingest -once                     # 跑一轮就退出（手动补数 / 演练 / 验证）
//	events-ingest -schedule "0 3 * * *"     # 常驻，按 cron 表达式循环跑（one.sh 里的常驻进程）
//
// 它与老的 cmd/ingest 是并存关系，不是替代：
//   - cmd/ingest 走 internal/ingest 的 HTML 抓取（nyato）+ 老的单项任务开关；
//   - 本命令走 internal/eventsource 适配器 → service.EventIngestService 编排，
//     每轮都会往 event_source_runs 写运行日志，供 admin「数据源状态」页读。
//
// 日志一律带 [events-ingest] 前缀：one.sh 把它重定向到 /logs/events-ingest.log，
// grep 一个前缀就能把这个进程的输出从容器混合日志里挑出来。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"github.com/Maluslock/comic/server/internal/config"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// logf 统一前缀，保证每行都能被 grep 到（包括 log 包自己的时间戳前缀之后）。
func logf(format string, args ...any) {
	log.Printf("[events-ingest] "+format, args...)
}

func main() {
	var (
		once     = flag.Bool("once", false, "跑一轮全量入库后退出")
		schedule = flag.String("schedule", "", "cron 表达式（5 段：分 时 日 月 周），常驻循环；与 -once 互斥")
		sources  = flag.String("sources", "allcpp,bilibili", "逗号分隔的抓取源，按给定顺序（allcpp,bilibili）")
		maxPages = flag.Int("max-pages", 0, "单源最多翻页数，0 = 适配器默认（10）")
		coverDir = flag.String("cover-dir", "", "封面落盘目录（默认自动探测 static/remote/covers）")
		dsn      = flag.String("dsn", "", "PostgreSQL DSN，空则用 config.Load() 的 .env 配置")
		timeout  = flag.Duration("timeout", 20*time.Minute, "-once 模式的总超时")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags)

	cfg, err := config.Load()
	if err != nil {
		logf("加载配置失败: %v", err)
		os.Exit(1)
	}

	pool, err := openPool(*dsn, cfg)
	if err != nil {
		logf("连接数据库失败: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	svc := service.NewEventIngestService(repository.New(pool), repository.NewEventSourcesRepo(pool), cfg).
		WithPool(pool).
		WithSources(splitList(*sources)).
		WithMaxPages(*maxPages)
	if *coverDir != "" {
		svc.WithCoverDir(*coverDir)
	}

	switch {
	case *once:
		runOnce(svc, *timeout)
	case strings.TrimSpace(*schedule) != "":
		runSchedule(svc, *schedule, *timeout)
	default:
		logf("用法：events-ingest -once | -schedule \"0 3 * * *\" [-sources allcpp,bilibili] [-max-pages N]")
		flag.PrintDefaults()
		os.Exit(2)
	}
}

// openPool 建立连接池：优先用 -dsn，其次 config 的 DSN()。
func openPool(dsn string, cfg *config.Config) (*pgxpool.Pool, error) {
	target := strings.TrimSpace(dsn)
	if target == "" {
		target = cfg.DSN()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, target)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// runOnce 跑一轮就退出。退出码语义（给 cron 之外的调用方判断用）：
//
//	0 = 至少一个源成功（部分源失败也给 0，因为数据进库了）
//	1 = 所有源都失败，或整轮报错
func runOnce(svc *service.EventIngestService, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	report, err := svc.IngestAll(ctx)
	if report != nil {
		logf("一轮结果: %s", report.Line())
		if len(report.Warnings) > 0 {
			logf("告警: %s", strings.Join(report.Warnings, " ;; "))
		}
	}
	if err != nil {
		logf("本轮失败: %v", err)
		os.Exit(1)
	}
}

// runSchedule 常驻：按 cron 表达式循环跑，Ctrl+C / SIGTERM 优雅退出。
//
// 用 cron.WithChain(SkipIfStillRunning)：上一轮没跑完就跳过本轮，
// 避免两个轮次互相覆盖 event_source_runs 的计数口径（服务内还有一道 TryLock，
// 这里是给 cron 层的可视化日志）。
func runSchedule(svc *service.EventIngestService, expr string, timeout time.Duration) {
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))

	job := func() {
		start := time.Now()
		logf("cron 触发，开始一轮（%s）", start.Format("2006-01-02 15:04:05"))

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		report, err := svc.IngestAll(ctx)
		if report != nil {
			logf("一轮结果: %s", report.Line())
			if len(report.Warnings) > 0 {
				logf("告警: %s", strings.Join(report.Warnings, " ;; "))
			}
		}
		if err != nil {
			logf("本轮失败（下一轮继续）: %v", err)
			return
		}
		logf("本轮完成，用时 %s", time.Since(start).Round(time.Millisecond))
	}

	entryID, err := c.AddFunc(expr, job)
	if err != nil {
		logf("cron 表达式 %q 非法: %v", expr, err)
		os.Exit(1)
	}

	c.Start()
	defer func() {
		stopCtx := c.Stop()
		select {
		case <-stopCtx.Done():
		case <-time.After(30 * time.Second):
			logf("等待在跑的一轮超时，强制退出")
		}
	}()

	logf("cron 已启动：schedule=%q entry=%d", expr, entryID)
	for _, ent := range c.Entries() {
		logf("cron next run: %s（每 %s）", ent.Next.Format("2006-01-02 15:04:05 MST"), ent.Next.Sub(time.Now()).Round(time.Second))
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	got := <-sig
	logf("收到信号 %s，停止调度并退出", got)
}

// splitList 把 "a, b ,c" 拆成 ["a","b","c"]。
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

var _ = fmt.Sprintf // 保留 fmt 供后续调试时快速插入格式化输出
