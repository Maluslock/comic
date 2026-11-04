-- 000042 一次性数据修正：把摄影师档案里的「占位名」换成真实昵称
--
-- ── 为什么要这一步 ──
-- 线上 photographers#15 的 name 是 '11'（纯数字），users#147 的 name 是微信授权
-- 默认值「微信用户」。列表页展示的是 photographers.name，所以全站看到的是 '11'；
-- 而用户自己的账号页显示「微信用户」——两条数据源各说各话。
--
-- 这本来由 Activate 的幂等分支（refreshActivatedProfile + isPlaceholderName）在
-- 「用户再点一次开通摄影师」时修掉。但真实的存量用户早就激活过了，他日常不会再去点
-- 那个按钮，所以那条代码路径永远走不到。存量脏数据需要一次显式的、可审计的修正。
--
-- ── 修什么、不修什么 ──
-- 修：photographers.name 是**占位值**的行 —— 空白 / 纯数字 / 「微信用户」，
--     且该行确实挂着 user_id，且对应 users.name 本身**不是**占位值。
--     这时用 users.name 覆盖 photographers.name。
-- 不靠真名修：users.name 也是占位值的行 —— 两个占位符互换等于没修，还会把仅有的
--     '11' 这类痕迹抹掉，让后续人工排查失去线索。这种行交给下面的第二段处理。
-- 不修：description / location / mode / avatar / mutual_intro —— 用户 description
--     也是 '11' 那是他自己写着玩的备注，不属于「对外展示名」，迁移无权代改。
-- 不修：没有 user_id 的档案（平台模板/历史无主档案），没有可依据的真实昵称。
-- 不修：已经是真人艺名的行 —— 艺名是摄影师的核心品牌资产（详见
--     internal/service/photographer_service.go 的 isPlaceholderName 注释）。
--
-- ── 第二段：两边都是占位值时的「认输式回落」──
-- 线上还有第二种死锁：photographers#15.name='11'，users#147.name='微信用户'。
-- 两个都是占位值 → 第一段按设计跳过（互换占位符没意义），但结果是这行**永远
-- 修不掉**：他早就是摄影师了，不会再点「开通摄影师」，users.name 也不会自己
-- 变成真名（他不填就一直是「微信用户」）。
--
-- 解法不是凭空造一个人名（那是伪造数据），而是给一个**明确的、可识别的系统
-- 默认占位**：'摄影师' || id，例如「摄影师15」。它比 '11' 强在三处：
--   (a) 用户一眼看出这是系统给的默认名，知道该去改；
--   (b) 列表里不再是莫名其妙的 '11'；
--   (c) 一旦他哪天填了真名，internal/service/photographer_service.go 的
--       isPlaceholderName 认得 ^摄影师[0-9]+$ 这个模式，于是允许被覆盖。
--
-- 触发条件：p.name 是占位值 AND (u.name 也是占位值 OR 根本没有 user_id)。
-- 有真名的情况已被第一段吃掉了，所以这里的 u.name 必然不可用。
--
-- ── 幂等 ──
-- 两段都是「幂等吸收」的：第一次跑完占位名就变成真名 / '摄影师N'，第二次跑时
-- photographers.name 已不满足「是占位值」的条件，WHERE 不命中，不会二次改写。
-- 全程参数无关（纯 UPDATE），重复执行不报错、不产生新数据。
--
-- ── 为什么放迁移而不是 Go 启动钩子 ──
-- 这类修正是「一次性的数据事实订正」，不是业务逻辑。写进启动流程会变成隐式行为：
-- 每次重启都可能悄悄改数据，出问题没人能追溯是哪次启动干的。迁移是显式的、
-- 有版本号的、可被 psql 复现的。Go 侧只保留「用户主动操作时」的修正路径。

BEGIN;

UPDATE photographers p
SET name = u.name,
    updated_at = NOW()
FROM users u
WHERE u.id = p.user_id
  -- 只动占位名：去空白后为空 / 全数字 / 微信默认昵称 / 本迁移第二段落下的 '摄影师N'
  AND (
        btrim(p.name) = ''
        OR p.name = '微信用户'
        OR btrim(p.name) ~ '^[0-9]+$'
        OR btrim(p.name) ~ '^摄影师[0-9]+$'
      )
  -- 真名字必须本身可用，否则两个占位符互换毫无意义
  AND btrim(u.name) <> ''
  AND u.name <> '微信用户'
  AND btrim(u.name) !~ '^[0-9]+$'
  -- 已经一样就没必要写
  AND p.name <> u.name;

-- 第二段：两边都是占位值（或没有 user_id）时的认输式回落 → '摄影师' || id
UPDATE photographers p
SET name = '摄影师' || p.id,
    updated_at = NOW()
WHERE (
        btrim(p.name) = ''
        OR p.name = '微信用户'
        OR btrim(p.name) ~ '^[0-9]+$'
      )
  -- '摄影师N' 已经是我们的兜底产物，不再重复改写（幂等的关键）
  AND btrim(p.name) !~ '^摄影师[0-9]+$'
  AND (
        p.user_id IS NULL
        OR NOT EXISTS (
          SELECT 1 FROM users u
          WHERE u.id = p.user_id
            AND btrim(u.name) <> ''
            AND u.name <> '微信用户'
            AND btrim(u.name) !~ '^[0-9]+$'
        )
      );

COMMIT;
