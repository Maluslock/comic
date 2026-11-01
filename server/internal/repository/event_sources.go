package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 抓取源常量 —— 与 migrations/000040_event_sources.up.sql 里的
// comic_events_source_check 白名单保持一致。两处必须同步改。
const (
	SourceAllcpp   = "allcpp"
	SourceBilibili = "bilibili"
	SourceNyato    = "nyato"
	SourceManual   = "manual"
)

// EventSources 是 comic_events.source 允许的全部取值（用于 admin 页面渲染与入参校验）。
var EventSources = []string{SourceAllcpp, SourceBilibili, SourceNyato, SourceManual}

// 运行状态常量 —— 与 event_source_runs_status_check 白名单保持一致。
const (
	RunStatusRunning = "running"
	RunStatusOK      = "ok"
	RunStatusPartial = "partial"
	RunStatusFailed  = "failed"
	RunStatusEmpty   = "empty"
)

// ErrSourceRunNotFound 表示 FinishRun 的目标 run id 不存在（被清理或 id 拼错）。
var ErrSourceRunNotFound = errors.New("event source run: not found")

// ErrInvalidSource 表示传入的来源不在白名单内。
var ErrInvalidSource = errors.New("event source: invalid source")

// SourceRun 是 event_source_runs 的一行：一轮抓取的可观测记录。
// 字段名与 JSON 标签面向 admin「数据源状态」页，前端直接吃。
type SourceRun struct {
	ID         int64      `json:"id"`
	Source     string     `json:"source"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Status     string     `json:"status"`
	Items      int32      `json:"items"`
	Inserted   int32      `json:"inserted"`
	Updated    int32      `json:"updated"`
	Errors     int32      `json:"errors"`
	DurationMs *int32     `json:"durationMs,omitempty"`
	Message    *string    `json:"message,omitempty"`
}

// SourceCount 是「某个源在 comic_events 里有几条」的聚合。
// Total 含已下架（del_flag=true），Active 只算在架的 —— 两者差值是清理掉的过期场次，
// admin 页同时显示才能看出「源还在产出但都被判定过期」这种异常。
type SourceCount struct {
	Source string `json:"source"`
	Total  int64  `json:"total"`
	Active int64  `json:"active"`
}

// FinishRunArg 是关闭一轮抓取所需的全部字段。
// DurationMs 为 nil 时由 SQL 用 NOW() - started_at 现算（推荐：调用方不必自己掐表，
// 也就不会因为时区/单位算错）。
type FinishRunArg struct {
	ID         int64
	Status     string
	Items      int
	Inserted   int
	Updated    int
	Errors     int
	DurationMs *int32
	Message    string
}

// EventSourcesRepo 是 event_source_runs 的手写仓储（不走 sqlc）。
// 与 internal/repository/admin_stats.go 同风格：裸 pgxpool，无 Queries 包装，
// 因为这些查询都要么是「一轮抓取」的写入，要么是 admin 聚合，不参与 sqlc 生成流程。
type EventSourcesRepo struct {
	pool *pgxpool.Pool
}

func NewEventSourcesRepo(pool *pgxpool.Pool) *EventSourcesRepo {
	return &EventSourcesRepo{pool: pool}
}

// runColumns 是 SourceRun 的扫描列顺序，三处 SELECT 共用，避免列序漂移。
const runColumns = `
	id, source, started_at, finished_at, status,
	items, inserted, updated, errors, duration_ms, message`

// scanSourceRun 按 runColumns 的顺序扫描一行。
func scanSourceRun(row interface {
	Scan(dest ...any) error
}) (SourceRun, error) {
	var r SourceRun
	err := row.Scan(&r.ID, &r.Source, &r.StartedAt, &r.FinishedAt, &r.Status,
		&r.Items, &r.Inserted, &r.Updated, &r.Errors, &r.DurationMs, &r.Message)
	return r, err
}

// validSource 校验来源名是否在白名单内（大小写与空白已被规整）。
func validSource(source string) bool {
	for _, s := range EventSources {
		if s == source {
			return true
		}
	}
	return false
}

// StartRun 插入一条 status='running' 的运行记录并返回它的 id。
// 编排服务拿到 id 后跑抓取，结束时调用 FinishRun 收尾 —— 即使进程中途崩溃，
// 那一行也会永远停在 running，admin 页据此能看出「上一轮跑挂了没回来」。
func (r *EventSourcesRepo) StartRun(ctx context.Context, source string) (int64, error) {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if !validSource(normalized) {
		return 0, fmt.Errorf("%w: %q (allowed: %s)",
			ErrInvalidSource, source, strings.Join(EventSources, ", "))
	}

	var id int64
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO event_source_runs (source) VALUES ($1) RETURNING id`,
		normalized).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// FinishRun 关闭一轮抓取：写状态、计数、耗时与消息。
//
// duration_ms 走 COALESCE：arg.DurationMs 非空就用调用方给的，否则用 NOW() - started_at 现算
// （GREATEST 兜底，防止系统时钟回拨得到负数）。
// 不校验 status 白名单由 DB 的 CHECK 兜住 —— 非法值会直接报错而不是悄悄落库。
func (r *EventSourcesRepo) FinishRun(ctx context.Context, arg FinishRunArg) error {
	normalizedStatus := strings.ToLower(strings.TrimSpace(arg.Status))
	if normalizedStatus == "" {
		normalizedStatus = RunStatusOK
	}

	var message *string
	if msg := strings.TrimSpace(arg.Message); msg != "" {
		message = &msg
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE event_source_runs
		   SET status      = $2,
		       finished_at = NOW(),
		       items       = $3,
		       inserted    = $4,
		       updated     = $5,
		       errors      = $6,
		       duration_ms = COALESCE($7::int,
		                              GREATEST(0, (EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::int)),
		       message     = $8
		 WHERE id = $1`,
		arg.ID, normalizedStatus, arg.Items, arg.Inserted, arg.Updated, arg.Errors,
		arg.DurationMs, message)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSourceRunNotFound
	}
	return nil
}

// RecentRuns 返回全局最近的 limit 轮运行（不区分来源），给 admin 运行日志时间线用。
// limit <= 0 时用 20，上限 500 —— 这是给页面看的，不是导出接口。
func (r *EventSourcesRepo) RecentRuns(ctx context.Context, limit int) ([]SourceRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+runColumns+`
		   FROM event_source_runs
		  ORDER BY started_at DESC, id DESC
		  LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SourceRun, 0, limit)
	for rows.Next() {
		item, err := scanSourceRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// LatestRunPerSource 每个来源只返回最新一轮（admin「数据源状态」卡片）。
// 用 DISTINCT ON 而不是 N 次单源查询：一次扫描、一次往返；
// 排序键必须是 (source, started_at DESC, id DESC) 才能让 DISTINCT ON 取到同秒插入里更晚的那条。
func (r *EventSourcesRepo) LatestRunPerSource(ctx context.Context) ([]SourceRun, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (source) `+runColumns+`
		   FROM event_source_runs
		  ORDER BY source ASC, started_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SourceRun, 0, len(EventSources))
	for rows.Next() {
		item, err := scanSourceRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CountBySource 按来源统计 comic_events 的条目数（含 / 不含已下架的过期行）。
// 只返回库里真实存在的来源 —— 一个从未入库的源不该在统计里凭空出现 0 行；
// admin 页要展示「全部已知源」时用 EventSources 补零。
func (r *EventSourcesRepo) CountBySource(ctx context.Context) ([]SourceCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT source,
		       COUNT(*)                                  AS total,
		       COUNT(*) FILTER (WHERE del_flag = false)  AS active
		  FROM comic_events
		 GROUP BY source
		 ORDER BY total DESC, source ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SourceCount, 0, len(EventSources))
	for rows.Next() {
		var c SourceCount
		if err := rows.Scan(&c.Source, &c.Total, &c.Active); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}
