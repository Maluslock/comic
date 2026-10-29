-- 担保交易 P0：订单事件流水（append-only）。
-- 每个履约/争议/取消节点都写一条事件，用于时间线、通知去重、审计。
CREATE TABLE IF NOT EXISTS order_events (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  actor_id BIGINT,
  actor_role VARCHAR(20),
  event_type VARCHAR(40) NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 事件类型约束：与履约节点、争议、取消、信用对应。
ALTER TABLE order_events
  DROP CONSTRAINT IF EXISTS order_events_event_type_check;
ALTER TABLE order_events
  ADD CONSTRAINT order_events_event_type_check
  CHECK (event_type IN (
    'confirmed', 'checkin', 'deliver', 'accept', 'auto_accept', 'complete_legacy',
    'cancel', 'auto_cancel', 'noshow', 'dispute_open', 'dispute_respond',
    'dispute_escalate', 'dispute_resolve', 'credit_delta'
  ));

-- actor_role 若是非空，必须是约定角色。
ALTER TABLE order_events
  DROP CONSTRAINT IF EXISTS order_events_actor_role_check;
ALTER TABLE order_events
  ADD CONSTRAINT order_events_actor_role_check
  CHECK (actor_role IS NULL OR actor_role IN ('coser', 'photographer', 'system', 'admin'));

-- 按 booking + 时间拉时间线；event_type 用于快速筛选某类事件。
CREATE INDEX IF NOT EXISTS idx_order_events_booking_time
  ON order_events(booking_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_events_booking_type
  ON order_events(booking_id, event_type);

-- 通知去重：同一 booking 同一事件类型只应存在一条 system/admin 触发的事件（幂等）。
-- 用户多次上传交付凭证等业务事件允许重复，因此唯一键只覆盖系统/管理员自动事件。
CREATE UNIQUE INDEX IF NOT EXISTS idx_order_events_unique_system_event
  ON order_events(booking_id, event_type)
  WHERE actor_role IN ('system', 'admin');
