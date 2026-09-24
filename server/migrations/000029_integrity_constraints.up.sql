-- 两条「不变量必须由 DB 保证」的完整性约束。
--
-- 1) bookings 时段唯一（仅未取消的订单）：
--    应用层的冲突检查是 CountConflictBookings(SELECT) → INSERT，是典型的 TOCTOU。实测把 12 个
--    请求用线程屏障对齐后同时发出，有 7 个返回 201、7 条订单落在同一 (photographer_id, date, time)
--    —— 一位摄影师同一档期被约了 7 次。顺序请求能正确 409，但并发打不穿，只有 DB 约束能保证。
--    WHERE status <> 'cancelled'：取消要释放时段，历史取消单允许堆叠。
--
-- 2) reviews 一人一摄影师一条：评价此前没有任何前置校验（没约过也能评、可无限重复），
--    这条唯一索引先兜住重复；「必须存在已完成订单」由服务层校验。
--
-- 幂等：可重复执行。建索引前已确认存量数据无冲突（非取消订单无重复时段；无同一人重复评价）。
CREATE UNIQUE INDEX IF NOT EXISTS uniq_bookings_active_slot
  ON bookings (photographer_id, date, time)
  WHERE status <> 'cancelled';

CREATE UNIQUE INDEX IF NOT EXISTS uniq_reviews_user_photographer
  ON reviews (photographer_id, user_id);
