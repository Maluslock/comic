UPDATE photographers p SET user_id = u.id
FROM users u
WHERE p.user_id IS NULL AND u.phone = '1000000000' || p.id;

UPDATE photographers p SET user_id = u.id
FROM users u
WHERE p.user_id IS NULL AND p.id = 5 AND u.phone = '13700000005';
