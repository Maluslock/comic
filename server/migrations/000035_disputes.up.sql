-- 担保交易 P0：争议工单与争议事件流水。
CREATE TABLE IF NOT EXISTS disputes (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  opened_by BIGINT NOT NULL,
  opened_by_role VARCHAR(20) NOT NULL,
  type VARCHAR(20) NOT NULL,
  reason TEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'open',
  respond_deadline TIMESTAMPTZ NOT NULL,
  evidence_deadline TIMESTAMPTZ NOT NULL,
  resolution TEXT,
  admin_id BIGINT REFERENCES admins(id),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 争议状态机：open -> responding -> escalated -> resolved -> closed。
ALTER TABLE disputes
  DROP CONSTRAINT IF EXISTS disputes_status_check;
ALTER TABLE disputes
  ADD CONSTRAINT disputes_status_check
  CHECK (status IN ('open', 'responding', 'escalated', 'resolved', 'closed'));

-- 争议类型。
ALTER TABLE disputes
  DROP CONSTRAINT IF EXISTS disputes_type_check;
ALTER TABLE disputes
  ADD CONSTRAINT disputes_type_check
  CHECK (type IN ('no_show', 'not_delivered', 'quality', 'other'));

-- 开单方角色。
ALTER TABLE disputes
  DROP CONSTRAINT IF EXISTS disputes_opened_by_role_check;
ALTER TABLE disputes
  ADD CONSTRAINT disputes_opened_by_role_check
  CHECK (opened_by_role IN ('coser', 'photographer'));

-- 一个 booking 同一时刻只能有一个未关闭争议。
CREATE UNIQUE INDEX IF NOT EXISTS idx_disputes_booking_open
  ON disputes(booking_id)
  WHERE status <> 'closed';

-- cron 默认裁决扫描索引。
CREATE INDEX IF NOT EXISTS idx_disputes_status_respond_deadline
  ON disputes(status, respond_deadline);
CREATE INDEX IF NOT EXISTS idx_disputes_status_evidence_deadline
  ON disputes(status, evidence_deadline);

-- 争议事件流水：记录提交、响应、升级、裁决、关闭。
CREATE TABLE IF NOT EXISTS dispute_events (
  id BIGSERIAL PRIMARY KEY,
  dispute_id BIGINT NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
  actor_id BIGINT,
  actor_role VARCHAR(20),
  event_type VARCHAR(40) NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE dispute_events
  DROP CONSTRAINT IF EXISTS dispute_events_event_type_check;
ALTER TABLE dispute_events
  ADD CONSTRAINT dispute_events_event_type_check
  CHECK (event_type IN ('open', 'respond', 'escalate', 'resolve', 'close', 'auto_resolve'));

ALTER TABLE dispute_events
  DROP CONSTRAINT IF EXISTS dispute_events_actor_role_check;
ALTER TABLE dispute_events
  ADD CONSTRAINT dispute_events_actor_role_check
  CHECK (actor_role IS NULL OR actor_role IN ('coser', 'photographer', 'system', 'admin'));

CREATE INDEX IF NOT EXISTS idx_dispute_events_dispute_time
  ON dispute_events(dispute_id, created_at DESC);
