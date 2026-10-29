-- 担保交易 P0：回滚 credit。
DROP INDEX IF EXISTS idx_credit_events_user_created;
DROP INDEX IF EXISTS idx_credit_events_unique_ref;

ALTER TABLE credit_events
  DROP CONSTRAINT IF EXISTS credit_events_reason_check,
  DROP CONSTRAINT IF EXISTS credit_events_ref_type_check;

DROP TABLE IF EXISTS credit_events;

ALTER TABLE credit_scores
  DROP CONSTRAINT IF EXISTS credit_scores_score_check;

DROP TABLE IF EXISTS credit_scores;
