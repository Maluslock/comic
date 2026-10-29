-- 000034 给 order_events 加了「system/admin 事件按 (booking_id, event_type) 唯一」的索引，
-- 本意是让三个 cron 幂等（auto_accept / auto_cancel / noshow），但它的作用域过大：
-- 同一个 booking 上第二次 admin 裁决（dispute_resolve）会撞唯一键 → 插入报错被日志吞掉，
-- 事件链断在 dispute_respond；credit_delta 同理，第二次加减分不留痕。
--
-- 把它收窄到真正需要幂等的那三类事件，其余 admin/system 事件按业务可重复。
DROP INDEX IF EXISTS idx_order_events_unique_system_event;

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_events_unique_system_event
  ON order_events(booking_id, event_type)
  WHERE actor_role IN ('system', 'admin')
    AND event_type IN ('auto_accept', 'auto_cancel', 'noshow');
