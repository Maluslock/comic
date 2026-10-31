-- 摄影师申请入驻某个漫展。
-- 背景：event detail 的「参展摄影师」此前是 GetRecommendedPhotographers(4)（全站推荐），
-- 与漫展毫无关系。本表把「谁在某场漫展接单」变成真实数据，由摄影师申请 + admin 审核驱动。
--
-- status 取值：pending / approved / rejected / withdrawn
--   pending   —— 已提交，等审核
--   approved  —— 审核通过，进入 event detail 的参展摄影师列表
--   rejected  —— 审核驳回（可在同一漫展重新申请，故不进唯一索引）
--   withdrawn —— 摄影师自己撤回（保留历史行，不删除）
--
-- user_id / admin_id 有意冗余存 BIGINT（不加外键约束）：user_id 是申请时的身份快照，
-- 让 C 端「我的申请」不必再 join users 就能按 token 身份过滤；admin_id 仅审计用。
CREATE TABLE IF NOT EXISTS event_photographers (
  id BIGSERIAL PRIMARY KEY,
  event_id BIGINT NOT NULL REFERENCES comic_events(id) ON DELETE CASCADE,
  photographer_id BIGINT NOT NULL REFERENCES photographers(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  message TEXT,
  review_reason TEXT,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  admin_id BIGINT
);

-- 硬保证：同一摄影师在同一漫展只能有一条活跃申请（rejected 放行，允许重新申请）。
-- 并发重复提交靠这条索引撞 23505 → 409，而不是先查后插的竞态判断。
-- 注意：withdrawn 仍算「活跃」（索引口径是 status <> 'rejected'）。所以「撤回 → 再申请」
-- 不会新插一行，而是在 repository 层用 ReviveWithdrawnEventApplication 把旧行改回 pending
-- （保留 history，不破坏索引）。撞 23505 时先尝试复用 withdrawn 行，复用不到才是真重复。
CREATE UNIQUE INDEX IF NOT EXISTS uniq_event_photographer_active
  ON event_photographers(event_id, photographer_id)
  WHERE status <> 'rejected';

-- event detail 的主查询路径：按漫展取 approved 列表
CREATE INDEX IF NOT EXISTS idx_event_photographers_event
  ON event_photographers(event_id, status);

-- C 端「我的申请」：按用户取全部申请（含已驳回/已撤回的历史）
CREATE INDEX IF NOT EXISTS idx_event_photographers_user
  ON event_photographers(user_id, applied_at DESC);
