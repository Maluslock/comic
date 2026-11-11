-- 000044 回滚：删掉风格偏好表与「已做过选择」开关列
--
-- up 只新增表 + 列，没有改写任何既有行，down 直接删干净即可回到旧结构。
-- 用户偏好数据随之丢弃（down 本来就是不可逆操作）。
--
-- 先 DROP COLUMN users.style_pref_chosen 再 DROP TABLE：顺序无依赖，但保持
-- 「列在前、表在后」与 up 的创建顺序对称。
--
-- IF EXISTS 保证可重复执行（本项目迁移靠手工 psql 应用，没有 schema_migrations 表）。

BEGIN;

ALTER TABLE users DROP COLUMN IF EXISTS style_pref_chosen;

DROP TABLE IF EXISTS user_style_prefs;

COMMIT;
