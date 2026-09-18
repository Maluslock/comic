-- 008: 将外链图片改指向本地静态资源 /static/img/...
-- 目的：服务器无外网时图片仍可显示。
-- 静态文件由 scripts/cache-images.sh 抓取到 src/static/img（C 端随包）与 server/static/img（后端托管）。
-- 该迁移幂等：仅替换仍然指向外部域名的值。

-- ---------- 摄影师头像 ----------
UPDATE photographers SET avatar = '/static/img/avatar-photographer1.svg' WHERE avatar LIKE 'https://api.dicebear.com/%photographer1%';
UPDATE photographers SET avatar = '/static/img/avatar-photographer2.svg' WHERE avatar LIKE 'https://api.dicebear.com/%photographer2%';
UPDATE photographers SET avatar = '/static/img/avatar-photographer3.svg' WHERE avatar LIKE 'https://api.dicebear.com/%photographer3%';
UPDATE photographers SET avatar = '/static/img/avatar-photographer4.svg' WHERE avatar LIKE 'https://api.dicebear.com/%photographer4%';
UPDATE photographers SET avatar = '/static/img/avatar-photographer5.svg' WHERE avatar LIKE 'https://api.dicebear.com/%photographer5%';

-- ---------- 用户头像（登录时生成的 DiceBear 一律换成本地默认头像） ----------
UPDATE users SET avatar = '/static/img/avatar-user.svg' WHERE avatar LIKE 'https://api.dicebear.com/%';
UPDATE users SET avatar = '/static/img/avatar-coser1b.svg' WHERE avatar LIKE 'https://api.dicebear.com/%coser1%';
UPDATE users SET avatar = '/static/img/avatar-coser2.svg'  WHERE avatar LIKE 'https://api.dicebear.com/%coser2%';
UPDATE users SET avatar = '/static/img/avatar-test1.svg'   WHERE avatar LIKE 'https://api.dicebear.com/%test1%';

-- ---------- 作品图（TEXT[] 逐元素替换） ----------
UPDATE works SET images = ARRAY(
  SELECT CASE u
    WHEN 'https://picsum.photos/seed/coswork1/600/450' THEN '/static/img/work-1.jpg'
    WHEN 'https://picsum.photos/seed/coswork2/600/450' THEN '/static/img/work-2.jpg'
    WHEN 'https://picsum.photos/seed/coswork3/600/450' THEN '/static/img/work-3.jpg'
    WHEN 'https://picsum.photos/seed/coswork4/600/450' THEN '/static/img/work-4.jpg'
    WHEN 'https://picsum.photos/seed/coswork5/600/450' THEN '/static/img/work-5.jpg'
    ELSE u
  END FROM unnest(images) AS u
) WHERE EXISTS (SELECT 1 FROM unnest(images) x WHERE x LIKE 'https://picsum.photos/%');

-- ---------- 评论头像 ----------
UPDATE reviews SET user_avatar = '/static/img/avatar-coser1.svg' WHERE user_avatar LIKE 'https://api.dicebear.com/%coser1%';
UPDATE reviews SET user_avatar = '/static/img/avatar-coser2.svg' WHERE user_avatar LIKE 'https://api.dicebear.com/%coser2%';

-- ---------- 轮播图 ----------
UPDATE banners SET image_url = '/static/img/cover-chinajoy.jpg' WHERE image_url = 'https://cms3.chinajoy.net/157/upload/resources/image/103312.jpg';
UPDATE banners SET image_url = '/static/img/cover-ccg.jpg'      WHERE image_url LIKE 'https://english.shanghai.gov.cn/%';
UPDATE banners SET image_url = '/static/img/banner-1.jpg'       WHERE image_url LIKE 'https://picsum.photos/%comic1%';
UPDATE banners SET image_url = '/static/img/banner-2.jpg'       WHERE image_url LIKE 'https://picsum.photos/%comic2%';
UPDATE banners SET image_url = '/static/img/banner-3.jpg'       WHERE image_url LIKE 'https://picsum.photos/%comic3%';

-- ---------- 漫展封面（2-7 为内联 SVG data URI，无需处理） ----------
UPDATE comic_events SET cover_url = '/static/img/cover-chinajoy.jpg' WHERE cover_url = 'https://cms3.chinajoy.net/157/upload/resources/image/103312.jpg';
UPDATE comic_events SET cover_url = '/static/img/cover-ccg.jpg'      WHERE cover_url LIKE 'https://english.shanghai.gov.cn/%';

-- ---------- 认证样片 ----------
UPDATE photographer_cert_applications SET evidence_images = ARRAY(
  SELECT CASE u
    WHEN 'https://picsum.photos/seed/cert1/600/450'  THEN '/static/img/cert-1.jpg'
    WHEN 'https://picsum.photos/seed/cert2/600/450'  THEN '/static/img/cert-2.jpg'
    WHEN 'https://picsum.photos/seed/cert2b/600/450' THEN '/static/img/cert-2b.jpg'
    WHEN 'https://picsum.photos/seed/cert3/600/450'  THEN '/static/img/cert-3.jpg'
    WHEN 'https://picsum.photos/seed/cert5/600/450'  THEN '/static/img/cert-5.jpg'
    ELSE u
  END FROM unnest(evidence_images) AS u
) WHERE EXISTS (SELECT 1 FROM unnest(evidence_images) x WHERE x LIKE 'https://picsum.photos/%');
