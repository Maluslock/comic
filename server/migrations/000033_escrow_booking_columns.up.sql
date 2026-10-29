-- 担保交易 P0：为 bookings 增加履约节点字段。
-- 主状态（pending/confirmed/completed/cancelled）保持不变，用 sub_status 承载细节。
-- 约束：
--   - 不改动 uniq_bookings_active_slot；
--   - 历史行 sub_status 保持 NULL；
--   - 不伪造 delivered_at。
ALTER TABLE bookings
  ADD COLUMN IF NOT EXISTS sub_status VARCHAR(20),
  ADD COLUMN IF NOT EXISTS checkin_code VARCHAR(6),
  ADD COLUMN IF NOT EXISTS checkin_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS accept_deadline TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS delivery_evidence TEXT[],
  ADD COLUMN IF NOT EXISTS cancel_policy VARCHAR(10) NOT NULL DEFAULT 'standard',
  ADD COLUMN IF NOT EXISTS cancelled_by VARCHAR(20),
  ADD COLUMN IF NOT EXISTS noshow_party VARCHAR(20);

-- sub_status 只接受文档约定的值；NULL 表示旧行或尚未进入节点。
ALTER TABLE bookings
  DROP CONSTRAINT IF EXISTS bookings_sub_status_check;
ALTER TABLE bookings
  ADD CONSTRAINT bookings_sub_status_check
  CHECK (sub_status IS NULL OR sub_status IN ('in_service', 'delivered', 'disputed'));

-- cancelled_by / noshow_party 若是非空，必须是约定角色。
ALTER TABLE bookings
  DROP CONSTRAINT IF EXISTS bookings_cancelled_by_check;
ALTER TABLE bookings
  ADD CONSTRAINT bookings_cancelled_by_check
  CHECK (cancelled_by IS NULL OR cancelled_by IN ('coser', 'photographer', 'system'));

ALTER TABLE bookings
  DROP CONSTRAINT IF EXISTS bookings_noshow_party_check;
ALTER TABLE bookings
  ADD CONSTRAINT bookings_noshow_party_check
  CHECK (noshow_party IS NULL OR noshow_party IN ('coser', 'photographer', 'both'));

-- cancel_policy 当前仅标准版。
ALTER TABLE bookings
  DROP CONSTRAINT IF EXISTS bookings_cancel_policy_check;
ALTER TABLE bookings
  ADD CONSTRAINT bookings_cancel_policy_check
  CHECK (cancel_policy IN ('standard'));

-- 为 cron 扫描 pending 自动取消、delivered 自动验收建索引。
CREATE INDEX IF NOT EXISTS idx_bookings_status_created_at ON bookings(status, created_at);
CREATE INDEX IF NOT EXISTS idx_bookings_sub_status_delivered_at ON bookings(sub_status, delivered_at);
