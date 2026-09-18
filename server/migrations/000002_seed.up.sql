DO $$
BEGIN
IF NOT EXISTS (SELECT 1 FROM tags) THEN

-- Tags (13 from mock)
INSERT INTO tags (name) VALUES
('日系'), ('古风'), ('暗黑'), ('清新'), ('科幻'), ('赛博朋克'),
('哥特'), ('少女'), ('汉服'), ('仙侠'), ('游戏'), ('动漫'), ('影视');

-- Services (4 global tiers from mock)
INSERT INTO services (name, price, description, duration) VALUES
('基础套餐', 399, '2小时拍摄，10张精修', 120),
('进阶套餐', 699, '4小时拍摄，20张精修', 240),
('精品套餐', 1299, '全天拍摄，40张精修，含妆造', 480),
('漫展跟拍', 599, '漫展当日跟拍，15张精修', 360);

-- Photographers (4 from mock)
INSERT INTO photographers (name, avatar, description, location, rating, review_count, order_count) VALUES
('光影行者', 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer1&backgroundColor=b6e3f4', '专注ACG摄影8年，擅长捕捉角色神韵，曾为多个知名coser拍摄官方宣传照。', '北京', 4.9, 234, 567),
('樱花落', 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer2&backgroundColor=ffd5dc', '日系小清新风格，擅长利用自然光营造梦幻氛围，少女心满满~', '上海', 4.8, 186, 423),
('暗夜骑士', 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer3&backgroundColor=c0aede', '专攻暗黑、哥特风格，用光影诠释角色的另一面，带你进入不一样的世界。', '广州', 4.7, 156, 312),
('古风公子', 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer4&backgroundColor=d1d4f9', '古风摄影大师，精通汉服、仙侠题材，还原古典美学极致。', '杭州', 4.9, 298, 678);

-- Photographer-tag associations
-- 光影行者: 日系(1), 古风(2), 科幻(5)
INSERT INTO photographer_tags (photographer_id, tag_id) VALUES (1,1),(1,2),(1,5);
-- 樱花落: 日系(1), 清新(4), 少女(8)
INSERT INTO photographer_tags (photographer_id, tag_id) VALUES (2,1),(2,4),(2,8);
-- 暗夜骑士: 暗黑(3), 哥特(7), 赛博朋克(6)
INSERT INTO photographer_tags (photographer_id, tag_id) VALUES (3,3),(3,7),(3,6);
-- 古风公子: 古风(2), 汉服(9), 仙侠(10)
INSERT INTO photographer_tags (photographer_id, tag_id) VALUES (4,2),(4,9),(4,10);

-- Works (4 from mock)
INSERT INTO works (photographer_id, title, images) VALUES
(1, '原神 - 雷电将军', ARRAY['https://picsum.photos/seed/coswork1/600/450', 'https://picsum.photos/seed/coswork2/600/450']),
(1, '鬼灭之刃 - 祢豆子', ARRAY['https://picsum.photos/seed/coswork3/600/450']),
(2, '魔卡少女樱', ARRAY['https://picsum.photos/seed/coswork4/600/450']),
(4, '古风仙侠', ARRAY['https://picsum.photos/seed/coswork5/600/450']);

-- Reviews (2 from mock)
INSERT INTO reviews (photographer_id, user_id, user_name, user_avatar, rating, content) VALUES
(1, 1, '小狐狸', 'https://api.dicebear.com/7.x/avataaars/svg?seed=coser1&backgroundColor=ffdfbf', 5, '摄影师非常专业，拍出来的效果超出预期！沟通也很顺畅，下次还会合作~'),
(1, 2, '月华', 'https://api.dicebear.com/7.x/avataaars/svg?seed=coser2&backgroundColor=c9e9f6', 5, '光影处理太棒了，每张照片都像海报一样！强烈推荐！');

-- Banners (3 from event covers — real images + SVG branded covers, verified 2026-08-05)
INSERT INTO banners (image_url, title, link_type, link_id, sort_order) VALUES
('https://cms3.chinajoy.net/157/upload/resources/image/103312.jpg', 'ChinaJoy 2026', 'event', 1, 0),
('https://english.shanghai.gov.cn/cmsres/04/04acd83239cf4fa8a45c056e2b0f01c0/9c3f75159398444056f2b7e40b37e8ab.jpg', '2026 CCG EXPO', 'event', 2, 1),
('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC40%E5%B1%8A%E8%90%A4%E7%81%AB%E8%99%AB%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%B9%BF%E5%B7%9E%20%C2%B7%20%E4%BF%9D%E5%88%A9%E4%B8%96%E8%B4%B8%E5%8D%9A%E8%A7%88%E9%A6%86%3C%2Ftext%3E%3C%2Fsvg%3E', '第40届萤火虫漫展', 'event', 3, 2);

END IF;
END $$;
