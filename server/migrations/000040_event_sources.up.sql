-- 000040 多源漫展入库：comic_events.source 来源列 + event_source_runs 抓取运行日志表
--
-- 背景：comic_events 之前只有一个隐含来源（allcpp.cn，早期还混入过 nyato 抓取），
-- 行级来源痕迹只剩 source_url 一列（206 行为空 = 老抓取，42 行是 bilibili 详情页）。
-- 「多源批量入库 + 失败自动切备用源 + 可观测运行日志」要求：
--   1) 每行漫展知道自己来自哪个源；
--   2) 每一轮抓取留下一条可查询的运行记录（开始/结束/状态/条数/耗时/错误）。
-- 本迁移补齐这两件事，语义变更仅此，不动抓取与 upsert 逻辑。
--
-- ── 为什么 allcpp_id 继续做跨源全局 upsert key，而不是改成 (source, source_id) 复合键 ──
-- 现有读路径与写路径全都绑定单列唯一键 allcpp_id：
--   * server/internal/db/query/events.sql 的 UpsertEvent / UpsertEventFromIngest 都是
--     `ON CONFLICT (allcpp_id) DO UPDATE`（生成文件 internal/repository/events.sql.go 同）；
--   * GetEventByAllcppId 被 service/handler 用来按外部 id 反查本地行；
--   * 唯一约束 comic_events_allcpp_id_key 是这条链路的硬保证。
-- 改成 (source, source_id) 复合键意味着所有 upsert / 反查 / 测试桩一起改，且外部 id 语义
-- 要拆成两列（allcpp 用真实 id、bilibili 用真实 project_id、nyato 用 ingest.AllcppID 的
-- FNV-1a 哈希，见 server/internal/ingest/hash.go），跨源去重口径也要重定义 —— 属于纯破坏性
-- 改造，收益为零：三个源的外部 id 空间本来就互不重叠（FNV 折叠进 int32 后仍保持确定性），
-- 单列唯一键已经能同时满足「同源更新」与「跨源不撞车」。
-- 因此 allcpp_id 保持不变：它就是「外部源 id」这一抽象本身，不是「allcpp 专用 id」；
-- source 列只回答「这行来自哪个源」，不参与唯一性判定。
--
-- 迁移可重复执行：ADD COLUMN IF NOT EXISTS / CREATE ... IF NOT EXISTS / 幂等 UPDATE /
-- 先 DROP CONSTRAINT IF EXISTS 再 ADD，重复跑不报错、不产生重复数据。
--
-- 注意（给抓取侧的约束）：source 默认 'allcpp' 只用于让历史行与新写入保持兼容。
-- 多源接入后，每个写入方必须显式写自己那一路的 source（bilibili/nyato/manual）；
-- 否则 bilibili 抓到的行会被默认值标成 allcpp —— 迁移无法替业务判断来源。

-- ─────────────────────────────── 1. 来源列 ───────────────────────────────
ALTER TABLE comic_events
  ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'allcpp';

-- 取值白名单：allcpp / bilibili / nyato / manual。
-- 先 DROP 再 ADD，保证重复执行不因「约束已存在」失败；名称沿用本仓约束命名习惯
-- （<table>_<column>_check，见 000036 credit_events_reason_check）。
ALTER TABLE comic_events
  DROP CONSTRAINT IF EXISTS comic_events_source_check;

ALTER TABLE comic_events
  ADD CONSTRAINT comic_events_source_check
  CHECK (source IN ('allcpp', 'bilibili', 'nyato', 'manual'));

-- 管理端「数据源状态」要按来源统计条目数，按来源过滤 event 列表也走这列。
CREATE INDEX IF NOT EXISTS idx_events_source ON comic_events(source);

-- ─────────────────────────── 2. 历史数据回填来源 ───────────────────────────
-- 口径（真库实测，2026-11-01）：248 行 = 42 行 bilibili 详情页 + 206 行 source_url 为空
-- （早期 nyato/allcpp 抓取，其中 8 行已被 expire 清理标 del_flag=true）。
--   42 行 source_url LIKE 'https://show.bilibili.com/platform/detail.html?id=...' → bilibili
--   206 行 source_url IS NULL 或空串                                       → allcpp（默认值）
--
-- 幂等：只更新「当前仍是 allcpp 且 source_url 命中 bilibili」的行。
-- 重复执行时这些行已是 bilibili，条件不再命中 → 影响 0 行，不会把数据改回去。
-- 不写 updated_at：这是元数据回填，不是内容变更；改 updated_at 会让同步时间失真。
UPDATE comic_events
   SET source = 'bilibili'
 WHERE source = 'allcpp'
   AND source_url ILIKE '%bilibili%';

-- ───────────────────────── 3. 抓取运行日志表 ─────────────────────────
-- 一轮抓取 = 一行。status 生命周期：
--   running  —— StartRun 插入的初始态
--   ok       —— 全部成功
--   partial  —— 部分成功（有错误但仍有入库）
--   failed   —— 整体失败（自动切备用源的触发依据）
--   empty    —— 抓取成功但 0 条（用于区分「源挂了」和「源没新数据」）
CREATE TABLE IF NOT EXISTS event_source_runs (
  id BIGSERIAL PRIMARY KEY,
  source VARCHAR(20) NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMPTZ,
  status VARCHAR(16) NOT NULL DEFAULT 'running',
  items INT NOT NULL DEFAULT 0,
  inserted INT NOT NULL DEFAULT 0,
  updated INT NOT NULL DEFAULT 0,
  errors INT NOT NULL DEFAULT 0,
  duration_ms INT,
  message TEXT
);

-- status 取值同样是白名单，防止拼错的 'succes' 悄悄落库。
ALTER TABLE event_source_runs
  DROP CONSTRAINT IF EXISTS event_source_runs_status_check;

ALTER TABLE event_source_runs
  ADD CONSTRAINT event_source_runs_status_check
  CHECK (status IN ('running', 'ok', 'partial', 'failed', 'empty'));

-- source 也钉同一份白名单：日志表里出现 'weibo' 这种未接入的源，只可能是调用方拼错名，
-- 而拼错的运行记录会污染「每源最新一轮」。repository.EventSourcesRepo.StartRun 已做
-- 同样的校验，这里加一层 DB 兜底，保证绕过仓储直接写 SQL 也写不进脏源。
ALTER TABLE event_source_runs
  DROP CONSTRAINT IF EXISTS event_source_runs_source_check;

ALTER TABLE event_source_runs
  ADD CONSTRAINT event_source_runs_source_check
  CHECK (source IN ('allcpp', 'bilibili', 'nyato', 'manual'));

-- 「某源最近 N 轮」与「每源最新一轮」的唯一查询路径。
CREATE INDEX IF NOT EXISTS idx_event_source_runs_source_started
  ON event_source_runs(source, started_at DESC);

-- 「全局最近 N 轮」（admin 运行日志时间线）不按 source 过滤，靠 started_at 排序，
-- 上面的复合索引对无 source 条件的前缀扫描用不上，补一个时间倒序索引避免全表扫。
CREATE INDEX IF NOT EXISTS idx_event_source_runs_started
  ON event_source_runs(started_at DESC);
