-- 000043 回滚：删掉「艺名已自定义」标记列
--
-- up 只新增一列 + 回填标记，没有改写任何一行的 photographers.name / users.name，
-- 所以删列即可回到旧结构，不会丢数据（唯一的副作用是下次改账号昵称时，
-- 所有摄影师档案都会重新跟随昵称——这是旧行为，也正是本迁移要修的那个 bug）。
--
-- IF EXISTS 保证可重复执行（本项目迁移靠手工 psql 应用，没有 schema_migrations 表）。

BEGIN;

ALTER TABLE photographers DROP COLUMN IF EXISTS name_is_custom;

COMMIT;
