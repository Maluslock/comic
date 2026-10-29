-- 担保交易 P0：回滚 bookings 扩列。
-- 先删索引，再删 CHECK 约束，最后按列依赖逆序 DROP COLUMN。
DROP INDEX IF EXISTS idx_bookings_sub_status_delivered_at;
DROP INDEX IF EXISTS idx_bookings_status_created_at;

ALTER TABLE bookings
  DROP CONSTRAINT IF EXISTS bookings_sub_status_check,
  DROP CONSTRAINT IF EXISTS bookings_cancelled_by_check,
  DROP CONSTRAINT IF EXISTS bookings_noshow_party_check,
  DROP CONSTRAINT IF EXISTS bookings_cancel_policy_check;

ALTER TABLE bookings
  DROP COLUMN IF EXISTS noshow_party,
  DROP COLUMN IF EXISTS cancelled_by,
  DROP COLUMN IF EXISTS cancel_policy,
  DROP COLUMN IF EXISTS delivery_evidence,
  DROP COLUMN IF EXISTS accept_deadline,
  DROP COLUMN IF EXISTS delivered_at,
  DROP COLUMN IF EXISTS checkin_at,
  DROP COLUMN IF EXISTS checkin_code,
  DROP COLUMN IF EXISTS sub_status;
