-- 000040 回滚：drop event_source_runs + comic_events.source
--
-- 只回滚本迁移自己造的东西：
--   * event_source_runs 表及其索引/约束；
--   * comic_events 的 source 列、CHECK 约束与 idx_events_source。
--
-- 刻意不碰的：
--   * comic_events_allcpp_id_key 唯一约束 —— 它是跨源 upsert 的全局 key（见 up 文件头部），
--     从 000001 就存在，不属于本次改动；
--   * 回填过的 source 值随 DROP COLUMN 一起消失，无需手工恢复；
--   * 运行日志数据（event_source_runs 行）随表丢弃，不做备份 —— 它是运行观测数据，
--     不是业务事实。

-- 先索引后表：索引随表一起消失，显式 DROP 只是让 down 在半途失败时可重跑。
DROP INDEX IF EXISTS idx_event_source_runs_started;
DROP INDEX IF EXISTS idx_event_source_runs_source_started;
DROP TABLE IF EXISTS event_source_runs;

DROP INDEX IF EXISTS idx_events_source;

ALTER TABLE comic_events
  DROP CONSTRAINT IF EXISTS comic_events_source_check;

ALTER TABLE comic_events
  DROP COLUMN IF EXISTS source;
