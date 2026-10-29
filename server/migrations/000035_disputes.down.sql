-- 担保交易 P0：回滚 disputes。
DROP INDEX IF EXISTS idx_dispute_events_dispute_time;

ALTER TABLE dispute_events
  DROP CONSTRAINT IF EXISTS dispute_events_event_type_check,
  DROP CONSTRAINT IF EXISTS dispute_events_actor_role_check;

DROP TABLE IF EXISTS dispute_events;

DROP INDEX IF EXISTS idx_disputes_status_evidence_deadline;
DROP INDEX IF EXISTS idx_disputes_status_respond_deadline;
DROP INDEX IF EXISTS idx_disputes_booking_open;

ALTER TABLE disputes
  DROP CONSTRAINT IF EXISTS disputes_status_check,
  DROP CONSTRAINT IF EXISTS disputes_type_check,
  DROP CONSTRAINT IF EXISTS disputes_opened_by_role_check;

DROP TABLE IF EXISTS disputes;
