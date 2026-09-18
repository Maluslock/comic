UPDATE photographers p SET user_id = NULL
FROM users u
WHERE p.user_id = u.id
  AND (u.phone = '1000000000' || p.id OR (p.id = 5 AND u.phone = '13700000005'));
