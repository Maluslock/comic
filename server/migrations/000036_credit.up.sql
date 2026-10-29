-- 担保交易 P0：信用分与信用事件。
-- 基准 100 分，由事实事件驱动加减。
CREATE TABLE IF NOT EXISTS credit_scores (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL UNIQUE,
  score INTEGER NOT NULL DEFAULT 100,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- score 约束：不得低于 0。
ALTER TABLE credit_scores
  DROP CONSTRAINT IF EXISTS credit_scores_score_check;
ALTER TABLE credit_scores
  ADD CONSTRAINT credit_scores_score_check
  CHECK (score >= 0);

-- 信用事件：每条加减分必须有据可查。
CREATE TABLE IF NOT EXISTS credit_events (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  delta INTEGER NOT NULL,
  reason VARCHAR(40) NOT NULL,
  ref_type VARCHAR(20),
  ref_id BIGINT,
  booking_id BIGINT REFERENCES bookings(id) ON DELETE SET NULL,
  description TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- reason 枚举：与方案口径一一对应。
ALTER TABLE credit_events
  DROP CONSTRAINT IF EXISTS credit_events_reason_check;
ALTER TABLE credit_events
  ADD CONSTRAINT credit_events_reason_check
  CHECK (reason IN (
    'order_completed', 'photographer_no_show', 'coser_no_show',
    'dispute_lost', 'late_cancel', 'delivery_overdue'
  ));

-- ref_type 若是非空，必须是约定类型。
ALTER TABLE credit_events
  DROP CONSTRAINT IF EXISTS credit_events_ref_type_check;
ALTER TABLE credit_events
  ADD CONSTRAINT credit_events_ref_type_check
  CHECK (ref_type IS NULL OR ref_type IN ('booking', 'dispute'));

-- 防刷：同一用户同一事实事件只计一次。
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_events_unique_ref
  ON credit_events(user_id, ref_type, ref_id, reason)
  WHERE ref_type IS NOT NULL AND ref_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_credit_events_user_created
  ON credit_events(user_id, created_at DESC);
