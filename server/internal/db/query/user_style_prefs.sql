-- sqlc 源文件。仓库里实际使用的是 internal/repository/user_style_prefs.sql.go（手工维护，
-- 因为构建环境无法联网跑 sqlc）。两边的方法签名与列清单必须逐字一致：任何一次只改一边，
-- 下一次有人跑 sqlc 就会把另一边覆盖掉。

-- name: ListStylePrefsByUser :many
SELECT tag_id
FROM user_style_prefs
WHERE user_id = $1
ORDER BY tag_id ASC;

-- name: ReplaceStylePrefs :execrows
DELETE FROM user_style_prefs WHERE user_id = $1;
-- （INSERT 侧由手工文件在事务内逐条执行，sqlc 无法表达「事务内循环插」，
--   因此这里只登记 DELETE 半段；INSERT SQL 以手工文件里的 insertStylePref 为准。）

-- name: ListStyleTagsByUser :many
SELECT t.id, t.name
FROM user_style_prefs p
JOIN tags t ON t.id = p.tag_id
WHERE p.user_id = $1
ORDER BY t.id ASC;

-- name: MarkStylePrefChosen :execrows
UPDATE users SET style_pref_chosen = true WHERE id = $1;

-- name: GetStylePrefChosen :one
SELECT style_pref_chosen FROM users WHERE id = $1;

-- name: CountTags :one
SELECT COUNT(*)::int FROM tags WHERE id = ANY($1::bigint[]);
