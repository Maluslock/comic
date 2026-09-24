-- 000030 是**纯派生数据的重算**（rating / review_count / order_count 全部由 reviews 与
-- bookings 推导），它不破坏任何原始信息，因此没有有意义的逆操作。
--
-- 刻意留成空操作：把这三个字段「回滚」成原来的假种子数字没有意义，而清零会破坏真实聚合值。
-- 需要恢复任何数字，请改事实来源（补评价 / 补完成的订单），或调用
-- repository.RecomputePhotographerStats(photographerID) 重算。
SELECT 1;
