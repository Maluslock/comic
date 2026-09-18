-- Event follows (关注漫展) + event subscriptions (开赛提醒订阅, 预留)
CREATE TABLE IF NOT EXISTS event_follows (
  id BIGSERIAL PRIMARY KEY,
  user_id VARCHAR(100) NOT NULL,
  event_id INTEGER NOT NULL REFERENCES comic_events(id),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (user_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_follows_user ON event_follows(user_id);

-- 订阅消息记录（demo 预留: 正式版需 AppID + 模板 ID）
CREATE TABLE IF NOT EXISTS event_subscriptions (
  id BIGSERIAL PRIMARY KEY,
  user_id VARCHAR(100) NOT NULL,
  event_id INTEGER NOT NULL REFERENCES comic_events(id),
  template_id VARCHAR(100) NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'accepted',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (user_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON event_subscriptions(user_id);
