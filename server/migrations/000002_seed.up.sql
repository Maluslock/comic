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

-- Banners (3 from event covers — seed for testing, real ones will be manually managed)
INSERT INTO banners (image_url, title, link_type, link_id, sort_order) VALUES
('https://picsum.photos/seed/comic1/750/360', '上海 CP30', 'event', 1, 0),
('https://picsum.photos/seed/comic2/750/360', '成都 CD28', 'event', 2, 1),
('https://picsum.photos/seed/comic3/750/360', '广州萤火虫', 'event', 3, 2);
