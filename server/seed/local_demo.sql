-- Local demo seed — reproduces the demo state described in AGENTS.md on a fresh DB.
-- Idempotent-ish: users upsert on phone; events insert on id conflict-do-nothing.
-- Run AFTER migrations 000001-000017. Cover URLs for events 1-8 are applied by re-running 000003.

-- ---------- users (photographer accounts + coser + test accounts) ----------
INSERT INTO users (phone, name, avatar) VALUES
  ('10000000001', '光影行者', '/static/img/avatar-photographer1.svg'),
  ('10000000002', '樱花落',   '/static/img/avatar-photographer2.svg'),
  ('10000000003', '暗夜骑士', '/static/img/avatar-photographer3.svg'),
  ('10000000004', '古风公子', '/static/img/avatar-photographer4.svg'),
  ('13800138000', '用户8000', '/static/img/avatar-coser1b.svg'),
  ('13900001111', '测试用户', '/static/img/avatar-test1.svg'),
  ('13700000005', '测试摄影师', '/static/img/avatar-photographer5.svg')
ON CONFLICT (phone) DO NOTHING;

-- ---------- link photographers 1-4 to their accounts, mark them certified (黄V) ----------
UPDATE photographers p SET
  user_id       = u.id,
  certified     = true,
  mode          = 'both',
  mutual_intro  = '欢迎约拍，互勉 / 收费均可，棚拍外景皆可。',
  activated_at  = NOW()
FROM users u
WHERE u.phone = '1000000000' || p.id AND p.id BETWEEN 1 AND 4;

-- ---------- photographer 5: unverified 测试摄影师 (for the rejected cert-apply flow) ----------
INSERT INTO photographers (name, avatar, description, location, rating, review_count, order_count, user_id, mode, mutual_intro, certified, activated_at)
SELECT '测试摄影师',
       '/static/img/avatar-photographer5.svg',
       '用于认证申请演示的测试摄影师账号。',
       '深圳', 4.5, 12, 20, u.id, 'free', '互勉为主，欢迎交流。', false, NOW()
FROM users u
WHERE u.phone = '13700000005'
  AND NOT EXISTS (SELECT 1 FROM photographers WHERE name = '测试摄影师');

-- ---------- comic events 1-8 (covers applied by re-running 000003) ----------
INSERT INTO comic_events (id, allcpp_id, name, location, venue, address, start_date, end_date, tags, type_name, status, del_flag) VALUES
  (1, 1001, 'ChinaJoy 2026',                 '上海', '上海新国际博览中心',   '上海市浦东新区龙阳路2345号',        '2026-08-01T00:00:00+08:00', '2026-08-04T00:00:00+08:00', ARRAY['游戏','数码','cosplay','电竞'], '游戏', 'upcoming', false),
  (2, 1002, '第40届萤火虫漫展',                '广州', '保利世贸博览馆',       '广州市海珠区阅江中路380号',         '2026-08-14T00:00:00+08:00', '2026-08-17T00:00:00+08:00', ARRAY['动漫','游戏','cosplay'],        '综合', 'upcoming', false),
  (3, 1003, 'CP33 综合同人展',                 '杭州', '杭州大会展中心',       '杭州市萧山区市心北路',              '2026-09-12T00:00:00+08:00', '2026-09-15T00:00:00+08:00', ARRAY['同人','创作','cosplay'],        '同人', 'upcoming', false),
  (4, 1004, '西安梦乡动漫展',                  '西安', '西安国际会展中心',     '西安市灞桥区会展一路1399号',        '2026-09-11T00:00:00+08:00', '2026-09-14T00:00:00+08:00', ARRAY['动漫','同人','汉服'],           '综合', 'upcoming', false),
  (5, 1005, 'IJOY国际动漫节',                  '北京', '北京国家会议中心',     '北京市朝阳区天辰东路7号',           '2026-10-01T00:00:00+08:00', '2026-10-03T00:00:00+08:00', ARRAY['综合','国潮','cosplay'],        '综合', 'upcoming', false),
  (6, 1006, '成都第二十四届世界线动漫展',       '成都', '中国西部国际博览城',   '成都市天府新区福州路东段88号',      '2026-09-26T00:00:00+08:00', '2026-09-28T00:00:00+08:00', ARRAY['动漫','cosplay','游戏'],        '综合', 'upcoming', false),
  (7, 1007, 'COMICUP 33 新青年',               '杭州', '杭州大会展中心',       '杭州市萧山区市心北路',              '2026-10-10T00:00:00+08:00', '2026-10-12T00:00:00+08:00', ARRAY['同人','创作','动漫'],           '同人', 'upcoming', false),
  (8, 1008, '2026上海CCG EXPO',                '上海', '上海跨国采购会展中心', '上海市普陀区光复西路2739号',        '2026-10-15T00:00:00+08:00', '2026-10-18T00:00:00+08:00', ARRAY['综合','游戏','动漫'],           '综合', 'upcoming', false)
ON CONFLICT (id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('comic_events','id'), GREATEST((SELECT MAX(id) FROM comic_events), 8));

-- ---------- certification applications (approved / rejected history) ----------
-- photographer 2 樱花落 -> approved
INSERT INTO photographer_cert_applications (user_id, photographer_id, evidence_images, evidence_desc, status, admin_id, reviewed_at)
SELECT u.id, 2, ARRAY['/static/img/cert-1.jpg'], '日系人像样片若干，含棚拍与自然光。', 'approved', 1, NOW()
FROM users u WHERE u.phone = '10000000002'
  AND NOT EXISTS (SELECT 1 FROM photographer_cert_applications WHERE photographer_id = 2);
-- photographer 3 暗夜骑士 -> rejected (earlier)
INSERT INTO photographer_cert_applications (user_id, photographer_id, evidence_images, evidence_desc, status, review_reason, admin_id, reviewed_at)
SELECT u.id, 3, ARRAY['/static/img/cert-2.jpg'], '暗黑哥特风格样片。', 'rejected', '样片数量不足，请补充作品后重新申请。', 1, NOW()
FROM users u WHERE u.phone = '10000000003'
  AND NOT EXISTS (SELECT 1 FROM photographer_cert_applications WHERE photographer_id = 3);
-- photographer 3 暗夜骑士 -> approved (later)
INSERT INTO photographer_cert_applications (user_id, photographer_id, evidence_images, evidence_desc, status, admin_id, reviewed_at)
SELECT u.id, 3, ARRAY['/static/img/cert-2.jpg','/static/img/cert-2b.jpg'], '补充后的暗黑哥特风格样片。', 'approved', 1, NOW()
FROM users u WHERE u.phone = '10000000003'
  AND NOT EXISTS (SELECT 1 FROM photographer_cert_applications WHERE photographer_id = 3 AND status = 'approved');
-- photographer 4 古风公子 -> approved
INSERT INTO photographer_cert_applications (user_id, photographer_id, evidence_images, evidence_desc, status, admin_id, reviewed_at)
SELECT u.id, 4, ARRAY['/static/img/cert-3.jpg'], '汉服 / 仙侠古风样片。', 'approved', 1, NOW()
FROM users u WHERE u.phone = '10000000004'
  AND NOT EXISTS (SELECT 1 FROM photographer_cert_applications WHERE photographer_id = 4);
-- photographer 5 测试摄影师 -> rejected (drives the "已驳回 + 重新申请" demo)
INSERT INTO photographer_cert_applications (user_id, photographer_id, evidence_images, evidence_desc, status, review_reason, admin_id, reviewed_at)
SELECT u.id, p.id, ARRAY['/static/img/cert-5.jpg'], '测试用样片。', 'rejected', '样片数量不足。', 1, NOW()
FROM users u JOIN photographers p ON p.user_id = u.id
WHERE u.phone = '13700000005'
  AND NOT EXISTS (SELECT 1 FROM photographer_cert_applications WHERE photographer_id = p.id);

-- ---------- a few bookings for the order module ----------
INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price, remarks)
SELECT 1, u.id, 1, '2026-09-20', '14:00', 'pending',   399, '想拍日系风格，希望在漫展现场。' FROM users u WHERE u.phone='13800138000'
  AND NOT EXISTS (SELECT 1 FROM bookings WHERE remarks = '想拍日系风格，希望在漫展现场。');
INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price, remarks)
SELECT 2, u.id, 2, '2026-09-21', '10:00', 'confirmed', 699, '小清新外景，上午光线好。' FROM users u WHERE u.phone='13800138000'
  AND NOT EXISTS (SELECT 1 FROM bookings WHERE remarks = '小清新外景，上午光线好。');
INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price, remarks)
SELECT 3, u.id, 3, '2026-09-10', '16:00', 'completed', 999, '暗黑哥特风格，已完成拍摄。' FROM users u WHERE u.phone='13800138000'
  AND NOT EXISTS (SELECT 1 FROM bookings WHERE remarks = '暗黑哥特风格，已完成拍摄。');
