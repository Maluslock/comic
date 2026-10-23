-- 主管理员（id=1，种子 admin/admin123）升为 super：
-- 管理员账号管理（列表/新建/禁用/重置密码）在 000032 之后只允许 role='super' 调用，
-- 否则任何人都能被普通 admin 提权或顶掉会话（见 middleware.RequireSuper）。
--
-- 只动 id=1：其他历史行保持 'admin'，避免把演示账号一并提权。
UPDATE admins SET role = 'super' WHERE id = 1;
