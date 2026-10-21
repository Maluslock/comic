-- 回滚 000029 加的两个完整性约束。
DROP INDEX IF EXISTS uniq_bookings_active_slot;
DROP INDEX IF EXISTS uniq_reviews_user_photographer;
