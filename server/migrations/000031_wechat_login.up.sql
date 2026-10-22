-- 微信小程序登录：手机号不再必填（微信登录用户可能没有绑定手机号），
-- 并新增 openid / unionid 唯一标识。openid 是同一小程序内的用户唯一键，
-- unionid 仅在绑定开放平台后返回，可为空。
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;

ALTER TABLE users ADD COLUMN IF NOT EXISTS wx_openid VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS wx_unionid VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_users_wx_openid
  ON users (wx_openid);
CREATE INDEX IF NOT EXISTS idx_users_wx_unionid
  ON users (wx_unionid) WHERE wx_unionid IS NOT NULL;
