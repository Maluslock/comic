-- 000043 摄影师对外展示名 vs 账号昵称：给 photographers 加「艺名已自定义」标记
--
-- ── 为什么要这一步 ──
-- 用户名下有两条独立的「名字」：
--   users.name         账号昵称（登录后「我的」页、聊天里的人名）
--   photographers.name 对外展示的摄影师名（列表 / 搜索 / 详情 / 预约卡片）
-- 此前两者各写各的：`PUT /api/v1/me/profile`（编辑资料页）只写 users.name，
-- 摄影师档案的 name 一直是注册时那份。线上症状：用户把账号昵称改成 MalusLock，
-- 「我的」页显示 MalusLock，但摄影师列表里还是「阿伟摄影」，搜 MalusLock 命中 0 条
-- （列表/搜索 SQL 只匹配 p.name / p.description）。
--
-- 修法是「一条单向同步 + 一个例外开关」：
--   默认 photographers.name 跟随 users.name（用户改了账号昵称，展示名跟着改）；
--   但用户一旦在「摄影师资料」页显式设过艺名，那个名字就是品牌资产，不能被账号昵称冲掉。
--
-- name_is_custom 就是这个开关：
--   false（默认）= 对外展示名跟随账号昵称，改昵称时同步覆盖；
--   true          = 用户显式设过艺名，账号昵称改动**不得**覆盖它。
--
-- ── 回填：保护现有艺名 ──
-- 加列之后不能让存量数据被第一次改昵称冲光。现有 6 行里 5 行是演示数据、
-- 1 行是真实用户（photographers#15），**每一行的 photographers.name 都不等于 users.name**。
-- 「两列已经不一致」就说明这个展示名不是账号昵称的镜像，一律视为用户自定义艺名 → true。
-- 这样后续的同步语句（WHERE name_is_custom = false）一行都不会命中，存量展示名原样保留；
-- 新注册用户走 Activate 建档时是 false，改账号昵称即可跟随。
--
-- ── 幂等 ──
-- ADD COLUMN IF NOT EXISTS；回填是纯 UPDATE（第一次跑把不一致的行置 true，第二次跑
-- 那些行已满足 true，SET 同值不报错、不影响其它行）。可重复执行。
--
-- ── down ──
-- 只删列，不回滚数据：name 列本身没有被本迁移改写，删开关不会丢任何用户数据。

BEGIN;

ALTER TABLE photographers ADD COLUMN IF NOT EXISTS name_is_custom boolean NOT NULL DEFAULT false;

-- 现有展示名与账号昵称不一致的行 = 用户自己起的艺名，标记为自定义，禁止被昵称同步覆盖。
-- 没有 user_id 的档案（平台模板 / 历史无主档案）没有可对照的昵称，保持 false 不影响任何行为
-- （同步语句按 user_id 匹配，无主档案永远不会被同步命中）。
UPDATE photographers p
SET name_is_custom = true
FROM users u
WHERE u.id = p.user_id
  AND p.name IS DISTINCT FROM u.name;

COMMIT;
