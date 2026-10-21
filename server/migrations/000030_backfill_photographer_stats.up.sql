-- 把 photographers 的三个对外展示数字回填成真实值：
--   rating       ← reviews 的评分均值（无评价为 0）
--   review_count ← reviews 条数
--   order_count  ← 已完成的 bookings 数
--
-- 背景：这三个字段一直是种子数据里的死数字，与 reviews / bookings 表完全脱钩。
-- 例：光影行者显示 4.9 分 / 234 条评价 / 567 单，而库里真实只有 2 条评价、1 单完成；
-- 更要紧的是插入评价或订单流转从不更新它们（代码侧已修：RecomputePhotographerStats
-- 会在评价写入与订单状态流转后重算）。这条迁移负责把**历史**数据也拉齐，
-- 否则「显示 4.9 分」和「评价列表只有 2 条」会一直互相矛盾。
--
-- 注意：回填后这些数字会显著变小（不再有 234 条评价这种观感）。这是有意为之 —— 数字必须
-- 说真话。若要恢复演示观感，正确做法是补真实的评价/订单数据，而不是改回假数字。
--
-- 幂等：纯派生计算，可重复执行。
UPDATE photographers p
SET rating = COALESCE((SELECT ROUND(AVG(r.rating)::numeric, 1) FROM reviews r WHERE r.photographer_id = p.id), 0.0),
    review_count = (SELECT COUNT(*) FROM reviews r WHERE r.photographer_id = p.id),
    order_count = (SELECT COUNT(*) FROM bookings b WHERE b.photographer_id = p.id AND b.status = 'completed'),
    updated_at = NOW();
