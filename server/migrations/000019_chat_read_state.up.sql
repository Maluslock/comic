CREATE TABLE IF NOT EXISTS chat_read_state (
  session_id   BIGINT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  user_id      BIGINT NOT NULL,
  last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (session_id, user_id)
);
