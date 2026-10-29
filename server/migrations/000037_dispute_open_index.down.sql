DROP INDEX IF EXISTS idx_disputes_booking_open;

CREATE UNIQUE INDEX IF NOT EXISTS idx_disputes_booking_open
  ON disputes(booking_id)
  WHERE status <> 'closed';
