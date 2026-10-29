DROP INDEX IF EXISTS idx_order_events_unique_system_event;

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_events_unique_system_event
  ON order_events(booking_id, event_type)
  WHERE actor_role IN ('system', 'admin');
