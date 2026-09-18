-- Soft-delete flag for expired events.
ALTER TABLE comic_events ADD COLUMN IF NOT EXISTS del_flag BOOLEAN DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_events_del_flag ON comic_events(del_flag) WHERE del_flag = false;
