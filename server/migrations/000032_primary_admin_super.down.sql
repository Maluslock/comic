-- 回滚：主管理员降回普通 admin（管理员账号管理接口随之对所有 admin 关闭）。
UPDATE admins SET role = 'admin' WHERE id = 1;
