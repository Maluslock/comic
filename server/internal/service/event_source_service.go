package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
)

// EventSourceRunLimit 是 admin「数据源状态」页运行历史返回的条数。
// 50 条覆盖最近两三天的三源轮次，够看出「谁刚跑挂过」，又不至于把页面拖慢。
const EventSourceRunLimit = 50

// eventSourceStatsWindow 是统计近 7 天成功率所用的「最多能回看多少轮」。
// 复用 t1 仓储的 RecentRuns 而不新增 7 天聚合 SQL：仓储层属 t1 的改动面，
// 本任务只允许动 service/handler。代价是窗口可能被截断 —— 15 分钟一轮时
// 7 天约 672 轮 > 500，此时统计退化为「最近 500 轮」；三源里最快的是 bilibili
// （cron 里 2 小时一轮），实际远低于 500，故当前不会截断。
const eventSourceStatsWindow = 500

// eventSourceStore 是 admin 数据源页需要的数据访问面，
// 由 *repository.EventSourcesRepo 实现（t1），单测用 fake 注入。
type eventSourceStore interface {
	RecentRuns(ctx context.Context, limit int) ([]repository.SourceRun, error)
	LatestRunPerSource(ctx context.Context) ([]repository.SourceRun, error)
	CountBySource(ctx context.Context) ([]repository.SourceCount, error)
}

// EventSourceStatus 是「单个来源」的状态卡片数据。
//
// LastRun/LastStatus/LastItems 取自最新一轮；SuccessRate7d 与 RunCount7d 是近 7 天聚合；
// TotalItems/ActiveItems 是库里现有的条目数（Total 含已下架）。
// 从未跑过、也从未入库的来源不会缺失 —— service 按 repository.EventSources 的顺序补零输出，
// 前端因此不用自己兜底，也能立刻看出「nyato 还没接过」。
type EventSourceStatus struct {
	Source        string     `json:"source"`
	LastRun       *time.Time `json:"lastRun"`
	LastStatus    string     `json:"lastStatus"`
	LastItems     int32      `json:"lastItems"`
	SuccessRate7d float64    `json:"successRate7d"`
	RunCount7d    int        `json:"runCount7d"`
	TotalItems    int64      `json:"totalItems"`
	ActiveItems   int64      `json:"activeItems"`
}

// EventSourceOverview 是 GET /api/admin/v1/event-sources 的完整响应。
// Runs 直接复用 repository.SourceRun（JSON 标签已面向本页设计），避免多一层搬运。
type EventSourceOverview struct {
	Sources     []EventSourceStatus    `json:"sources"`
	Runs        []repository.SourceRun `json:"runs"`
	Totals      map[string]int64       `json:"totals"`
	GeneratedAt time.Time              `json:"generatedAt"`
}

// EventSourceService 组装 admin 数据源状态页需要的全部读模型。
type EventSourceService struct {
	store eventSourceStore
	now   func() time.Time // 可注入的时钟，单测里把「近 7 天」钉死
}

func NewEventSourceService(store eventSourceStore) *EventSourceService {
	return &EventSourceService{store: store, now: time.Now}
}

// Overview 一次取三份数据并归并成页面契约：
//   - LatestRunPerSource：每源最新一轮（卡片「最近一次运行」）
//   - RecentRuns(50)：全局时间线（表格）
//   - CountBySource：每源在库条数（卡片「在库条数」+ totals）
//
// 近 7 天成功率由 RecentRuns(eventSourceStatsWindow) 在内存里聚合 —— 见 eventSourceStatsWindow 注释。
func (s *EventSourceService) Overview(ctx context.Context) (*EventSourceOverview, error) {
	latest, err := s.store.LatestRunPerSource(ctx)
	if err != nil {
		return nil, err
	}
	runs, err := s.store.RecentRuns(ctx, EventSourceRunLimit)
	if err != nil {
		return nil, err
	}
	counts, err := s.store.CountBySource(ctx)
	if err != nil {
		return nil, err
	}
	window, err := s.store.RecentRuns(ctx, eventSourceStatsWindow)
	if err != nil {
		return nil, err
	}

	now := s.now()
	rate, windowCount := successRate7d(window, now)

	latestBySource := make(map[string]repository.SourceRun, len(latest))
	for _, r := range latest {
		latestBySource[r.Source] = r
	}
	countBySource := make(map[string]repository.SourceCount, len(counts))
	totals := make(map[string]int64, len(counts))
	for _, c := range counts {
		countBySource[c.Source] = c
		totals[c.Source] = c.Total
	}

	// 顺序以 repository.EventSources 为准：页面卡片位置稳定，不随数据量抖动。
	// 库里出现白名单外的来源（历史脏数据）追加在末尾，不能静默吞掉。
	order := make([]string, 0, len(repository.EventSources)+len(counts))
	seen := make(map[string]bool, len(repository.EventSources))
	for _, src := range repository.EventSources {
		order = append(order, src)
		seen[src] = true
	}
	extra := make([]string, 0, len(counts))
	for _, c := range counts {
		if !seen[c.Source] {
			extra = append(extra, c.Source)
			seen[c.Source] = true
		}
	}
	sort.Strings(extra)
	order = append(order, extra...)

	sources := make([]EventSourceStatus, 0, len(order))
	for _, src := range order {
		st := EventSourceStatus{
			Source:        src,
			LastStatus:    "", // 从没跑过时显式空串，前端显示「未运行」
			SuccessRate7d: round4(rate[src]),
			RunCount7d:    windowCount[src],
		}
		if r, ok := latestBySource[src]; ok {
			started := r.StartedAt
			st.LastRun = &started
			st.LastStatus = r.Status
			st.LastItems = r.Items
		}
		if c, ok := countBySource[src]; ok {
			st.TotalItems = c.Total
			st.ActiveItems = c.Active
		}
		sources = append(sources, st)
	}

	if runs == nil {
		runs = []repository.SourceRun{}
	}

	return &EventSourceOverview{
		Sources:     sources,
		Runs:        runs,
		Totals:      totals,
		GeneratedAt: now,
	}, nil
}

// successRate7d 按来源聚合近 7 天的运行结果。
//
// 分子：status ∈ {ok, empty} —— empty 是「跑通了但这轮没新数据」，属于健康。
// 分母：status ∈ {ok, empty, partial, failed}，即已结束的轮次；
// running（进程还没回来或已崩溃卡住）不进分母，否则刚起跑的那一轮会把成功率拉低。
// 分母为 0 时该源不出现在 map 里，调用方按 0 处理。
func successRate7d(runs []repository.SourceRun, now time.Time) (map[string]float64, map[string]int) {
	cutoff := now.Add(-7 * 24 * time.Hour)
	ok := make(map[string]int)
	done := make(map[string]int)
	for _, r := range runs {
		if r.StartedAt.Before(cutoff) {
			continue
		}
		switch r.Status {
		case repository.RunStatusOK, repository.RunStatusEmpty:
			ok[r.Source]++
			done[r.Source]++
		case repository.RunStatusPartial, repository.RunStatusFailed:
			done[r.Source]++
		}
	}
	rate := make(map[string]float64, len(done))
	for src, total := range done {
		if total == 0 {
			continue
		}
		rate[src] = float64(ok[src]) / float64(total)
	}
	return rate, done
}

// round4 把比率收成 4 位小数，避免 0.6666666666666666 这种浮点噪声进 JSON。
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
