-- 000044 C 端首启风格偏好：user_style_prefs 表 + users.style_pref_chosen 开关
--
-- ── 为什么是关联表而不是 users 上的数组列 ──
-- 首页推荐后续要按用户勾选的风格加权/统计，关联表直接 GROUP BY tag_id 就能出结果；
-- 数组列还得 unnest，且没法对单个 tag 建有效索引。FK 双向都挂（user_id/tag_id），
-- 保证写不进脏 tag_id / 已注销用户的偏好行随 CASCADE 清掉，不留孤儿数据。
--
-- ── chosen 的语义 ──
-- users.style_pref_chosen 表示「用户明确做过一次选择（含主动清空）」。
-- 首启弹窗靠它判断「要不要再弹」：tags 为空可能是没选过，也可能是主动清空，
-- 所以 chosen 必须独立持久化，不能靠 tags 非空推断。默认 false = 从未选择。
--
-- ── 幂等 ──
-- CREATE TABLE / CREATE INDEX / ADD COLUMN 全部 IF NOT EXISTS。本项目迁移靠手工
-- psql 应用（没有 schema_migrations 表），重复执行不报错。
--
-- ── 数据兼容 ──
-- 纯新增：不修改任何现有列/行，存量 bookings/services/photographers/users 计数不变。

BEGIN;

CREATE TABLE IF NOT EXISTS user_style_prefs (
  user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tag_id     bigint NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, tag_id)
);

-- 按风格统计/加权推荐时的反查索引（谁选了某个风格）。
CREATE INDEX IF NOT EXISTS idx_user_style_prefs_tag ON user_style_prefs(tag_id);

ALTER TABLE users ADD COLUMN IF NOT EXISTS style_pref_chosen boolean NOT NULL DEFAULT false;

COMMIT;
