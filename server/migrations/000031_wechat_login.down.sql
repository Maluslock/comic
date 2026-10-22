-- 回滚前先清掉没有手机号的微信用户，否则 phone NOT NULL 加不回去。
DELETE FROM users WHERE phone IS NULL;
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;
DROP INDEX IF EXISTS idx_users_wx_unionid;
DROP INDEX IF EXISTS uniq_users_wx_openid;
ALTER TABLE users DROP COLUMN IF EXISTS wx_unionid;
ALTER TABLE users DROP COLUMN IF EXISTS wx_openid;
