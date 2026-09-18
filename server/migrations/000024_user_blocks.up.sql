CREATE TABLE IF NOT EXISTS user_blocks (
  user_id BIGINT NOT NULL,
  blocked_user_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, blocked_user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_blocks_blocked ON user_blocks(blocked_user_id);
