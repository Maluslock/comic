package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	"github.com/jackc/pgx/v5/pgxpool"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/config"
	"github.com/Maluslock/comic/server/internal/eventsource"
	"github.com/Maluslock/comic/server/internal/repository"
)

// EventIngestService 是多源漫展入库的编排层。
//
// 职责边界：
//   - eventsource 只负责「请求 → 解析 → 规范化 → 去重」，不碰数据库；
//   - 本服务负责「按优先级逐源跑 → 逐条 upsert → 封面本地化 → 写运行日志
//     → 过期清理 → 清缓存」，并把每轮的可观测结果落进 event_source_runs。
//
// 硬约束（安全阀，改代码前先读完，历史上踩过坑）：
//
//	a) 任一源失败绝不影响其他源继续跑：单个源的错误只记进它自己那行
//	   event_source_runs（status=failed）与本轮报告，循环里不 return。
//	a2) 单个源的抓取失败也不会把该源已有的数据清掉 —— 缺席不等于消失。
//
//	b) 只有当本轮「至少一个源成功（ok/partial）且合计入库条数 > 0」时，
//	   才执行过期清理：UPDATE comic_events SET del_flag = true
//	   WHERE end_date < NOW() AND del_flag = false。
//	   理由：清理是「按时间自然过期」，与「本轮抓到几条」无关；但如果整轮
//	   全挂（上游全 404 / 网络中断），说明我们对世界的认知已经过期，
//	   这时候动 del_flag 只会把库清干净却拿不到新数据。
//
//	c) 绝不做「本轮没抓到 = 全表软删」的缺席删除。
//	   反例是老的 scripts/ingest-bilibili.js 里那条
//	   `UPDATE ... WHERE allcpp_id NOT IN (<本轮 id 列表>)`：上游翻页少返回一页
//	   就能把几百条正常展会标成已下架。本服务不存在任何以「本轮 id 集合」
//	   为条件的删除/软删语句，这是刻意的。
//
//	d) upsert 成功后要把历史软删的行复活：del_flag 置回 false，
//	   但只对 end_date >= NOW() 的行做（已经结束的展会不该因为上游还在列表里
//	   就重新上架）。
type EventIngestService struct {
	queries *repository.Queries
	runRepo *repository.EventSourcesRepo
	cfg     *config.Config

	// pool 用于 sqlc 没生成的原生 SQL：source 列回写、del_flag 复活、
	// 过期清理、以及「这一条是新增还是更新」的存在性预查。
	// 没有它这些步骤全都做不了，因此 IngestAll 会在 pool 为 nil 时直接报错，
	// 而不是静默降级成「只 upsert 不写 source」这种半吊子状态。
	pool *pgxpool.Pool

	// sources 是按名字过滤后的抓取源；nil/空 = 全部（与 eventsource.All() 同序）。
	sources []string

	// coverDir 是封面落盘目录（相对当前工作目录）。API 从 server/ 启动，
	// 所以默认 "static/remote/covers" 与 router.Static("/static", "./static") 对得上；
	// 从仓库根跑时自动退化到 "server/static/remote/covers"。
	coverDir string

	maxPages   int
	httpClient *http.Client

	// mu 让同一进程内的多轮抓取串行化：cron 3 小时一轮，但一轮真跑久了
	// 会和下一轮重叠，重叠的两轮会互相覆盖 event_source_runs 的计数口径。
	mu sync.Mutex
}

// 封面本地化的尺寸口径 —— 与 scripts/cache-allcpp-covers.py 保持一致：
//   - 小于 2KB 的响应基本是占位图/错误页，落盘只会污染封面目录；
//   - 大于 3MB 的原图走 CDN 直链更划算，不去抢磁盘。
const (
	coverMinBytes = 2000
	coverMaxBytes = 3 << 20
)

// coverURLPrefix 是写进 comic_events.cover_url 的前缀（API 的静态路由是 /static）。
const coverURLPrefix = "/static/remote/covers/"

// 封面目录候选：前者对应「CWD = server/」，后者对应「CWD = 仓库根」。
const (
	defaultCoverDir  = "static/remote/covers"
	fallbackCoverDir = "server/static/remote/covers"
)

// coverExts 是 findExistingCover 会去找的扩展名，顺序影响复用优先级。
// webp 排最前优先命中已优化封面。
var coverExts = []string{"webp", "jpg", "jpeg", "png", "gif"}

// sourceFetchTimeout 是单个源单轮抓取的整体超时（适配器会用它包 ctx）。
// 10 页 × (请求 + 300~500ms 翻页间隔) 远用不到这么久，留足余量给上游抽风。
const sourceFetchTimeout = 3 * time.Minute

// applySourceSQL 在 upsert 之后回写来源列，并顺手复活历史软删的未结束展会。
//
// 为什么不改 UpsertEventFromIngest 的 SQL：那条语句是 sqlc 从 events.sql 生成的，
// 加 source 参数要动 sql/、querier.go 与全部调用点；而 ON CONFLICT (allcpp_id)
// 的语义必须保持（allcpp_id 是跨源唯一 key）。这里用一条独立 UPDATE 补上，
// 保持 upsert 的唯一冲突键不变，也避免把 eventsource 的 source 概念漏进 sqlc 层。
//
// updated_at 不动：upsert 已经写过一次，这里只是补列，不该再抬一次时间戳。
const applySourceSQL = `
	UPDATE comic_events
	   SET source   = $1,
	       del_flag = CASE WHEN end_date >= NOW() THEN false ELSE del_flag END
	 WHERE allcpp_id = $2`

// expireCleanupSQL 是唯一的软删语句（约束 b）：只按时间过期，不看本轮抓到什么。
const expireCleanupSQL = `
	UPDATE comic_events
	   SET del_flag = true
	 WHERE end_date < NOW()
	   AND del_flag = false`

// staleRunGrace 是「这行 running 还活着吗」的判定阈值。
//
// 进程被 SIGKILL / 容器重启发 / OOM 杀掉时，FinishRun 来不及执行，
// event_source_runs 会永远留一行 status='running'。admin 的「数据源状态」页
// 按「每源最新一轮」展示，僵尸行会盖掉真实结果，让人以为一直在跑。
// 单源抓取上限是 sourceFetchTimeout(3min)；取 30 分钟做兜底，正常在跑的轮次不会被误伤。
const staleRunGrace = 30 * time.Minute

// sweepStaleRunsSQL 把明显不可能还在跑的 running 行收成 failed。
const sweepStaleRunsSQL = `
	UPDATE event_source_runs
	   SET status      = 'failed',
	       finished_at = NOW(),
	       duration_ms = GREATEST(0, (EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::int),
	       message     = COALESCE(NULLIF(message, ''), '进程异常退出，运行记录未收尾（自动标记为 failed）')
	 WHERE status = 'running'
	   AND started_at < NOW() - ($1::int * INTERVAL '1 second')`

// SourceIngestReport 是「单个源这一轮」的结果，与 event_source_runs 的一行一一对应。
type SourceIngestReport struct {
	Source     string             `json:"source"`
	RunID      int64              `json:"runId"`
	Status     string             `json:"status"`
	Kept       int                `json:"kept"`
	Inserted   int                `json:"inserted"`
	Updated    int                `json:"updated"`
	Errors     int                `json:"errors"`
	DurationMs int64              `json:"durationMs"`
	Message    string             `json:"message,omitempty"`
	Fetch      eventsource.Report `json:"fetch"`
}

// IngestReport 是一轮多源抓取的完整结果，给 -once 的日志与 cron 的收尾汇总用。
type IngestReport struct {
	StartedAt        time.Time            `json:"startedAt"`
	FinishedAt       time.Time            `json:"finishedAt"`
	DurationMs       int64                `json:"durationMs"`
	Sources          []SourceIngestReport `json:"sources"`
	Kept             int                  `json:"kept"`
	Inserted         int                  `json:"inserted"`
	Updated          int                  `json:"updated"`
	Errors           int                  `json:"errors"`
	SuccessSources   int                  `json:"successSources"`
	FailedSources    int                  `json:"failedSources"`
	ExpiredMarked    int64                `json:"expiredMarked"`
	CacheInvalidated bool                 `json:"cacheInvalidated"`
	Warnings         []string             `json:"warnings,omitempty"`
}

// Line 把一轮结果压成一行，方便 /logs/events-ingest.log 里 grep。
func (r *IngestReport) Line() string {
	if r == nil {
		return "本轮无报告"
	}
	parts := make([]string, 0, len(r.Sources))
	for _, s := range r.Sources {
		parts = append(parts, fmt.Sprintf("%s=%s(k=%d,i=%d,u=%d,e=%d)",
			s.Source, s.Status, s.Kept, s.Inserted, s.Updated, s.Errors))
	}
	return fmt.Sprintf("合计 kept=%d inserted=%d updated=%d errors=%d 成功源=%d 失败源=%d 过期标记=%d 缓存=%v 用时=%dms [%s]",
		r.Kept, r.Inserted, r.Updated, r.Errors, r.SuccessSources, r.FailedSources,
		r.ExpiredMarked, r.CacheInvalidated, r.DurationMs, strings.Join(parts, " "))
}

// NewEventIngestService 构造编排服务。
//
// 注意：仓储类型实际叫 *repository.EventSourcesRepo（t1 的命名），
// 这里用真实类型而不是任务书里写的 EventSourceRepo。
// 必须再调用 WithPool 注入连接池，IngestAll 才有办法写 source 列与做过期清理。
func NewEventIngestService(
	queries *repository.Queries,
	runRepo *repository.EventSourcesRepo,
	cfg *config.Config,
) *EventIngestService {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &EventIngestService{
		queries: queries,
		runRepo: runRepo,
		cfg:     cfg,
	}
}

// WithPool 注入数据库连接池（必需）。
func (s *EventIngestService) WithPool(pool *pgxpool.Pool) *EventIngestService {
	s.pool = pool
	if s.queries == nil && pool != nil {
		s.queries = repository.New(pool)
	}
	return s
}

// WithSources 限定本轮要跑的源（名字，大小写与空白会被规整）；空 = 全部。
func (s *EventIngestService) WithSources(names []string) *EventIngestService {
	s.sources = normalizeSourceNames(names)
	return s
}

// WithCoverDir 覆盖封面落盘目录（相对 CWD）。
func (s *EventIngestService) WithCoverDir(dir string) *EventIngestService {
	s.coverDir = strings.TrimSpace(dir)
	return s
}

// WithMaxPages 覆盖单源最多翻页数（0 = 适配器默认）。
func (s *EventIngestService) WithMaxPages(n int) *EventIngestService {
	if n > 0 {
		s.maxPages = n
	}
	return s
}

// WithHTTPClient 覆盖下载/抓取用的 http client（主要给测试与代理场景）。
func (s *EventIngestService) WithHTTPClient(c *http.Client) *EventIngestService {
	s.httpClient = c
	return s
}

// IngestAll 跑一轮全量多源入库，返回本轮报告。
//
// 返回的 error 只表示「本轮整体失败」（所有被选中的源都失败了），
// 单源失败已经记在报告的 Sources 里 —— 调用方据此决定退出码，
// 但不该因为一个源挂了就丢掉其他源的成果。
func (s *EventIngestService) IngestAll(ctx context.Context) (*IngestReport, error) {
	if s.pool == nil {
		return nil, errors.New("event ingest: 未注入数据库连接池（先调用 WithPool）")
	}
	if s.runRepo == nil {
		return nil, errors.New("event ingest: 未注入 event_source_runs 仓储")
	}
	if s.queries == nil {
		s.queries = repository.New(s.pool)
	}

	// 上一轮还没跑完就别开新一轮：两轮并行会把同一批行反复 upsert，
	// event_source_runs 里也会出现两行 running，看日志的人会以为卡死了。
	if !s.mu.TryLock() {
		return nil, errors.New("event ingest: 上一轮还在跑，跳过本轮")
	}
	defer s.mu.Unlock()

	sources, err := s.selectedSources()
	if err != nil {
		return nil, err
	}

	report := &IngestReport{StartedAt: time.Now()}
	log.Printf("开始多源入库：源=%s 最多页数=%d", joinSourceNames(sources), s.maxPagesOrDefault())

	// 先把上一轮被强杀留下的僵尸 running 行收掉：否则 admin 的「每源最新一轮」
	// 会一直显示 running，看日志的人分不清「在跑」和「死掉了」。
	if swept, sweepErr := s.sweepStaleRuns(ctx); sweepErr != nil {
		log.Printf("清理僵尸运行记录失败（不影响本轮）: %v", sweepErr)
	} else if swept > 0 {
		log.Printf("清理僵尸运行记录：%d 行 running 超过 %s 未收尾，已标记 failed", swept, staleRunGrace)
	}

	for _, src := range sources {
		if ctx.Err() != nil {
			report.Warnings = append(report.Warnings, "ctx 已取消，剩余源未执行: "+ctx.Err().Error())
			break
		}
		res := s.ingestSource(ctx, src)
		report.Sources = append(report.Sources, res)

		report.Kept += res.Kept
		report.Inserted += res.Inserted
		report.Updated += res.Updated
		report.Errors += res.Errors
		switch res.Status {
		case repository.RunStatusOK, repository.RunStatusPartial:
			report.SuccessSources++
		case repository.RunStatusFailed:
			report.FailedSources++
		}
		if len(res.Fetch.Warnings) > 0 {
			report.Warnings = append(report.Warnings, res.Source+": "+strings.Join(res.Fetch.Warnings, " | "))
		}
	}

	// 约束 b：至少一个源成功、且本轮确实拿到了数据，才动过期清理。
	if report.SuccessSources > 0 && report.Kept > 0 {
		marked, err := s.markExpired(ctx)
		if err != nil {
			log.Printf("过期清理失败（不影响本轮入库）: %v", err)
			report.Warnings = append(report.Warnings, "过期清理失败: "+err.Error())
		} else {
			report.ExpiredMarked = marked
			if marked > 0 {
				log.Printf("过期清理：标记 %d 条已结束展会为 del_flag=true", marked)
			}
		}
	} else {
		log.Printf("跳过过期清理：成功源=%d 本轮入库=%d（整轮失败或空轮不动 del_flag）",
			report.SuccessSources, report.Kept)
		report.Warnings = append(report.Warnings, "跳过过期清理：整轮失败或空轮")
	}

	// 只要库里可能有变化就清缓存（含过期清理改了可见性）。
	if report.Inserted+report.Updated > 0 || report.ExpiredMarked > 0 {
		report.CacheInvalidated = s.invalidateCache(ctx)
	}

	report.FinishedAt = time.Now()
	report.DurationMs = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
	log.Printf("多源入库结束：%s", report.Line())

	// 全源失败才向上抛错：让 -once 能用非 0 退出码报警，而部分失败照常成功退出。
	if len(report.Sources) > 0 && report.SuccessSources == 0 {
		msgs := make([]string, 0, len(report.Sources))
		for _, res := range report.Sources {
			msgs = append(msgs, res.Source+": "+res.Message)
		}
		return report, fmt.Errorf("event ingest: 所有源都失败（%s）", strings.Join(msgs, "; "))
	}
	return report, nil
}

// selectedSources 按优先级返回本轮要跑的源。
//
// 顺序与 eventsource.All() 一致（allcpp 在前，bilibili 垫底），
// 但这里是逐个显式构造的：要把 config 里的 ALLCPP_BASE_URL 与自建 http client
// 传下去（断源演练就是靠把 base URL 指到 404 完成的）。
func (s *EventIngestService) selectedSources() ([]eventsource.Source, error) {
	known := s.buildSources()

	want := make(map[string]struct{}, len(s.sources))
	for _, name := range s.sources {
		want[name] = struct{}{}
	}

	out := make([]eventsource.Source, 0, len(known))
	for _, src := range known {
		if len(want) > 0 {
			if _, ok := want[src.Name()]; !ok {
				continue
			}
		}
		out = append(out, src)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("event ingest: 没有可用的抓取源（请求=%v，已知=%s）",
			s.sources, joinSourceNames(known))
	}
	if len(want) > 0 {
		for name := range want {
			found := false
			for _, src := range out {
				if src.Name() == name {
					found = true
					break
				}
			}
			if !found {
				log.Printf("未知抓取源 %q，已忽略（可选：%s）", name, joinSourceNames(known))
			}
		}
	}
	return out, nil
}

// buildSources 构造已知适配器（顺序即优先级）。
func (s *EventIngestService) buildSources() []eventsource.Source {
	allcppCfg := eventsource.AllcppConfig{BaseURL: strings.TrimSpace(s.cfg.AllcppBaseURL)}
	biliCfg := eventsource.BilibiliConfig{}
	if s.httpClient != nil {
		allcppCfg.HTTPClient = s.httpClient
		biliCfg.HTTPClient = s.httpClient
	}
	return []eventsource.Source{
		eventsource.NewAllcppWithConfig(allcppCfg),
		eventsource.NewBilibiliWithConfig(biliCfg),
	}
}

// ingestSource 跑单个源：StartRun → Fetch → 逐条 upsert → FinishRun。
// 任何错误都退化成「这一源记 failed」，绝不 return 出去打断外层的其他源。
func (s *EventIngestService) ingestSource(ctx context.Context, src eventsource.Source) SourceIngestReport {
	name := src.Name()
	started := time.Now()
	out := SourceIngestReport{Source: name}

	runID, err := s.runRepo.StartRun(ctx, name)
	if err != nil {
		// 连运行日志都写不进去（库挂了/源名不在白名单），抓取也没意义：
		// 记一条本地失败即可，不影响其他源。
		out.Status = repository.RunStatusFailed
		out.Message = "启动运行记录失败: " + err.Error()
		out.DurationMs = time.Since(started).Milliseconds()
		log.Printf("源 %s: %s", name, out.Message)
		return out
	}
	out.RunID = runID
	log.Printf("源 %s: run#%d 开始抓取（最多 %d 页）", name, runID, s.maxPagesOrDefault())

	res, fetchErr := eventsource.FetchResult(ctx, src, s.fetchOptions())
	out.Fetch = res.Report
	out.Kept = len(res.Items)

	if fetchErr != nil {
		// 首页就失败：这个源本轮零产出（不会删任何旧数据，见约束 a2）。
		out.Status = repository.RunStatusFailed
		out.Message = "抓取失败: " + fetchErr.Error()
		out.DurationMs = time.Since(started).Milliseconds()
		log.Printf("源 %s: %s（已获取 %d 条，不影响其他源）", name, out.Message, out.Kept)
		s.finishRun(ctx, name, runID, out, started)
		return out
	}

	for _, item := range res.Items {
		if ctx.Err() != nil {
			out.Errors++
			out.Message = "ctx 取消，剩余 " + fmt.Sprint(len(res.Items)) + " 条未处理"
			break
		}
		inserted, err := s.persistItem(ctx, name, item)
		if err != nil {
			out.Errors++
			if out.Errors <= 5 { // 只打前 5 条，避免上游数据脏时刷屏
				log.Printf("源 %s: 写入 %d（%s）失败: %v", name, item.AllcppID, truncate(item.Name, 40), err)
			}
			continue
		}
		if inserted {
			out.Inserted++
		} else {
			out.Updated++
		}
	}

	switch {
	case out.Kept == 0 && out.Errors == 0:
		out.Status = repository.RunStatusEmpty
	case out.Errors > 0 || len(res.Report.Warnings) > 0:
		out.Status = repository.RunStatusPartial
	default:
		out.Status = repository.RunStatusOK
	}

	out.DurationMs = time.Since(started).Milliseconds()
	out.Message = buildSourceMessage(out, res.Report)
	log.Printf("源 %s: %s（%dms）", name, out.Message, out.DurationMs)
	s.finishRun(ctx, name, runID, out, started)
	return out
}

// finishRun 收尾写 event_source_runs：状态 + 全部计数 + 消息。
// 这里的错误只打日志 —— 运行日志写不进去不该让已经入库的数据白跑。
func (s *EventIngestService) finishRun(ctx context.Context, name string, runID int64, out SourceIngestReport, started time.Time) {
	durMs := int32(time.Since(started).Milliseconds())
	if err := s.runRepo.FinishRun(ctx, repository.FinishRunArg{
		ID:         runID,
		Status:     out.Status,
		Items:      out.Kept,
		Inserted:   out.Inserted,
		Updated:    out.Updated,
		Errors:     out.Errors,
		DurationMs: &durMs,
		Message:    out.Message,
	}); err != nil {
		log.Printf("源 %s: 写 event_source_runs#%d 失败: %v", name, runID, err)
	}
}

// persistItem 写单条展会：存在性预查 → upsert → 回写 source / 复活 del_flag。
func (s *EventIngestService) persistItem(ctx context.Context, srcName string, it eventsource.EventItem) (bool, error) {
	name := strings.TrimSpace(it.Name)
	if name == "" || it.End.IsZero() {
		return false, fmt.Errorf("缺少必要字段（name 或 end_date 为空）")
	}

	start := it.Start
	if start.IsZero() {
		// 上游只给了结束时间时不要把 0001-01-01 写进 start_date（会污染排序与展示）。
		start = it.End
	}

	source := strings.TrimSpace(it.Source)
	if source == "" {
		source = srcName
	}

	// 先判存在：UpsertEventFromIngest 只回 id，不告诉我们这次是插还是更新，
	// 而报告与 event_source_runs 必须区分 inserted / updated。
	var existed bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM comic_events WHERE allcpp_id = $1)`,
		it.AllcppID).Scan(&existed); err != nil {
		return false, fmt.Errorf("查询是否已存在: %w", err)
	}

	cover := s.localizeCover(ctx, source, it)

	if _, err := s.queries.UpsertEventFromIngest(ctx, repository.UpsertEventFromIngestParams{
		AllcppID:  it.AllcppID,
		Name:      name,
		Location:  strPtr(strings.TrimSpace(it.City)),
		Venue:     strPtr(strings.TrimSpace(it.Venue)),
		Address:   strPtr(strings.TrimSpace(it.Address)),
		StartDate: start,
		EndDate:   it.End,
		CoverUrl:  strPtr(cover),
		Tags:      it.Tags,
		TypeName:  strPtr(strings.TrimSpace(it.TypeName)),
		SourceUrl: strPtr(strings.TrimSpace(it.SourceURL)),
	}); err != nil {
		return false, fmt.Errorf("upsert: %w", err)
	}

	// 约束 d：source 列必须显式写（sqlc 那条 SQL 不带 source，默认值 allcpp 只是历史兼容），
	// 同时把「历史被软删、但结束时间还没到」的行复活。
	if _, err := s.pool.Exec(ctx, applySourceSQL, source, it.AllcppID); err != nil {
		return false, fmt.Errorf("回写 source: %w", err)
	}

	return !existed, nil
}

// sweepStaleRuns 把僵死超过 staleRunGrace 的 running 行收成 failed，返回处理行数。
//
// 这是「可观测性自愈」而不是业务逻辑：进程被杀是常态（容器重启、OOM、kill -9），
// 运行日志表不能因此永久失真。
func (s *EventIngestService) sweepStaleRuns(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, sweepStaleRunsSQL, int(staleRunGrace.Seconds()))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// markExpired 是唯一的软删入口（约束 b/c），只按 end_date 判定。
func (s *EventIngestService) markExpired(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, expireCleanupSQL)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---------------------------------------------------------------------------
// 封面本地化
// ---------------------------------------------------------------------------

// localizeCover 把封面下载到 static/remote/covers/ 并返回可用的 cover_url。
//
// 三种结果：
//  1. 已有同名文件（上一轮抓过 / 老脚本 cache-allcpp-covers.py 下过 <id>.jpg）→ 直接复用；
//  2. 下载成功（>2KB 且 ≤3MB）→ 落盘并返回 /static/remote/covers/{source}-{id}.{ext}；
//  3. 任何失败 → 原样返回上游 http 链接。
//
// 绝不返回空字符串：cover_url 为空会让前端列表出现「无封面」空洞，
// 而一个能打开的外链永远比空值有用。
func (s *EventIngestService) localizeCover(ctx context.Context, source string, it eventsource.EventItem) string {
	raw := strings.TrimSpace(it.CoverURL)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(raw), "http") {
		// 已经是相对路径/本地路径（上游没给绝对地址），保持原样，别二次拼接。
		return raw
	}

	dir := s.resolveCoverDir()
	if dir == "" {
		return raw
	}

	base := fmt.Sprintf("%s-%d", source, it.AllcppID)
	if local := findExistingCover(dir, base); local != "" {
		return local
	}
	// 老脚本按 <allcpp_id>.jpg 落盘，同一个展会没必要重复下载一遍。
	if local := findExistingCover(dir, fmt.Sprintf("%d", it.AllcppID)); local != "" {
		return local
	}

	fileName, err := s.downloadCover(ctx, dir, base, raw)
	if err != nil {
		log.Printf("封面下载失败，保留原链 %s: %v", raw, err)
		return raw
	}
	return coverURLPrefix + fileName
}

// resolveCoverDir 决定封面目录，并保证它存在（失败时返回 ""，调用方退回原链）。
func (s *EventIngestService) resolveCoverDir() string {
	if s.coverDir != "" {
		if err := os.MkdirAll(s.coverDir, 0o755); err != nil {
			log.Printf("封面目录 %s 不可用: %v", s.coverDir, err)
			return ""
		}
		return s.coverDir
	}
	for _, dir := range []string{defaultCoverDir, fallbackCoverDir} {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			s.coverDir = dir
			return dir
		}
	}
	// 都不存在：按 API 的工作目录约定落 defaultCoverDir。
	if err := os.MkdirAll(defaultCoverDir, 0o755); err != nil {
		log.Printf("封面目录 %s 创建失败: %v", defaultCoverDir, err)
		return ""
	}
	s.coverDir = defaultCoverDir
	return defaultCoverDir
}

// findExistingCover 在 dir 下按 coverExts 找 base.ext，返回可供 DB 使用的 URL 路径。
func findExistingCover(dir, base string) string {
	for _, ext := range coverExts {
		name := base + "." + ext
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.IsDir() || st.Size() <= 0 {
			continue
		}
		return coverURLPrefix + name
	}
	return ""
}

// compressCoverToWebP 将图片按长边 <= maxSide 缩放并转为 webp quality (75)。
func compressCoverToWebP(data []byte, maxSide int, quality float32) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	longest := w
	if h > longest {
		longest = h
	}

	var target image.Image = img
	if longest > maxSide {
		scale := float64(maxSide) / float64(longest)
		newW := int(float64(w)*scale + 0.5)
		newH := int(float64(h)*scale + 0.5)
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		xdraw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)
		target = dst
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, target, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
}

// downloadCover 下载并落盘单个封面，返回文件名（不含目录）。
// 落盘前尝试压缩：长边 > 750 缩放、转 webp quality 75；压缩失败退回原图并记日志，不丢弃封面。
func (s *EventIngestService) downloadCover(ctx context.Context, dir, base, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", eventsource.BrowserUA)
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,*/*;q=0.8")
	req.Header.Set("Referer", rawURL)

	resp, err := s.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// 多读 1 字节用于判超限，避免把 50MB 的怪东西整个读进内存。
	// coverMaxBytes 超限判定在压缩前执行。
	data, err := io.ReadAll(io.LimitReader(resp.Body, coverMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > coverMaxBytes {
		return "", fmt.Errorf("响应体超过 %d 字节上限", coverMaxBytes)
	}
	if len(data) <= coverMinBytes {
		return "", fmt.Errorf("响应体只有 %d 字节，疑似占位图", len(data))
	}

	// 尝试压缩为 webp (长边 <= 750, quality 75)
	var finalData []byte
	var ext string
	if compressed, err := compressCoverToWebP(data, 750, 75); err == nil && len(compressed) > 0 {
		finalData = compressed
		ext = "webp"
	} else {
		log.Printf("ingest cover: compress %s failed (%v), fallback to original", rawURL, err)
		finalData = data
		ext = detectCoverExt(data, resp.Header.Get("Content-Type"), rawURL)
	}

	name := base + "." + ext
	tmp := filepath.Join(dir, name+".tmp")
	if err := os.WriteFile(tmp, finalData, 0o644); err != nil {
		return "", err
	}
	// 先写 .tmp 再 rename：cron 中途被杀不会留下半个文件被人当成封面。
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return name, nil
}

// detectCoverExt 判扩展名：magic bytes 优先（CDN 经常把 png 当 jpg 返回），
// 其次 Content-Type，最后才看 URL 后缀。
func detectCoverExt(data []byte, contentType, rawURL string) string {
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	switch strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(sniff), ";")[0])) {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	}

	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return "jpg"
	case strings.Contains(ct, "png"):
		return "png"
	case strings.Contains(ct, "webp"):
		return "webp"
	case strings.Contains(ct, "gif"):
		return "gif"
	}

	lower := strings.ToLower(rawURL)
	if idx := strings.IndexAny(lower, "?#"); idx >= 0 {
		lower = lower[:idx]
	}
	for _, ext := range []string{"jpg", "jpeg", "png", "webp", "gif"} {
		if strings.HasSuffix(lower, "."+ext) {
			if ext == "jpeg" {
				return "jpg"
			}
			return ext
		}
	}
	return "jpg"
}

// ---------------------------------------------------------------------------
// 缓存失效
// ---------------------------------------------------------------------------

// invalidateCache 清掉与漫展列表相关的读缓存。
//
// key 来源（逐个核过）：
//   - "home"                home_handler.go 的首页聚合（含 upcoming events）
//   - "events:" 前缀        event_handler.go 的 events:{location}:{status}:{page}:{size}
//   - "photographers:" 前缀 photograph*_handler.go 的列表/服务缓存（首页卡片的图与评分）
//
// Redis 不可用时 cache.NewRedisCache 返回 nil，所有方法 nil-safe，
// 因此这里不需要额外判断 —— 失效失败绝不能影响已入库的数据。
func (s *EventIngestService) invalidateCache(ctx context.Context) bool {
	rc := cache.NewRedisCache(s.cfg.RedisAddr(), "", 0)
	if rc == nil {
		log.Printf("缓存失效跳过：Redis %s 不可用", s.cfg.RedisAddr())
		return false
	}
	defer func() { _ = rc.Close() }()

	rc.Delete(ctx, "home")
	removed := rc.DeletePrefix(ctx, "events:") + rc.DeletePrefix(ctx, "photographers:")
	log.Printf("缓存失效：home + %d 个 events:/photographers: key", removed)
	return true
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

// fetchOptions 组装单次抓取参数。
func (s *EventIngestService) fetchOptions() eventsource.Options {
	return eventsource.Options{
		MaxPages: s.maxPages,
		Timeout:  sourceFetchTimeout,
	}
}

func (s *EventIngestService) maxPagesOrDefault() int {
	if s.maxPages > 0 {
		return s.maxPages
	}
	return eventsource.DefaultMaxPages
}

// client 返回下载封面用的 client（懒初始化，与适配器一样关掉 HTTP/2）。
func (s *EventIngestService) client() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	s.httpClient = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			ForceAttemptHTTP2:   false,
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 4,
			IdleConnTimeout:     60 * time.Second,
		},
	}
	return s.httpClient
}

// buildSourceMessage 组装 event_source_runs.message：一行关键计数 + 停止原因 + 告警。
func buildSourceMessage(out SourceIngestReport, rep eventsource.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "fetched=%d kept=%d inserted=%d updated=%d errors=%d pages=%d/%d dup=%d filtered=%d invalid=%d",
		rep.Fetched, out.Kept, out.Inserted, out.Updated, out.Errors,
		rep.Pages, rep.PageCount, rep.Duplicates, rep.Filtered, rep.Invalid)
	if rep.StopReason != "" {
		b.WriteString("; stop=")
		b.WriteString(rep.StopReason)
	}
	if out.Message != "" {
		b.WriteString("; note=")
		b.WriteString(out.Message)
	}
	if len(rep.Warnings) > 0 {
		b.WriteString("; warn=")
		b.WriteString(strings.Join(rep.Warnings, " | "))
	}
	return truncate(b.String(), 1000)
}

func normalizeSourceNames(names []string) []string {
	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, raw := range names {
		for _, part := range strings.Split(raw, ",") {
			name := strings.ToLower(strings.TrimSpace(part))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

func joinSourceNames(sources []eventsource.Source) string {
	names := make([]string, 0, len(sources))
	for _, src := range sources {
		names = append(names, src.Name())
	}
	return strings.Join(names, ",")
}

func truncate(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "…"
}
