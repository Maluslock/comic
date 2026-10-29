-- 担保交易 P0：回滚 order_events。
DROP INDEX IF EXISTS idx_order_events_unique_system_event;
DROP INDEX IF EXISTS idx_order_events_booking_type;
DROP INDEX IF EXISTS idx_order_events_booking_time;

ALTER TABLE order_events
  DROP CONSTRAINT IF EXISTS order_events_event_type_check,
  DROP CONSTRAINT IF EXISTS order_events_actor_role_check;

DROP TABLE IF EXISTS order_events;
