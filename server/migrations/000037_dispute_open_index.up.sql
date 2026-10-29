-- 000035 把唯一索引写成 WHERE status <> 'closed'，但 disputes.status 的取值是
-- open / escalated / resolved / rejected —— 没有 'closed'，所以这个谓词恒为真，
-- 等于「一个 booking 一辈子只能提交一次争议」：争议 resolved 之后再也提不了第二次。
--
-- 正确语义是「同一时刻只允许一个未结案争议」：open / responding / escalated 期间唯一，
-- 一旦 resolved / closed 就不该再挡住新的争议。
DROP INDEX IF EXISTS idx_disputes_booking_open;

CREATE UNIQUE INDEX IF NOT EXISTS idx_disputes_booking_open
  ON disputes(booking_id)
  WHERE status IN ('open', 'responding', 'escalated');
