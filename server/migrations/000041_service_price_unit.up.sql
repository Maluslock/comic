-- 000041 套餐按张计价（price_unit / photo_count）
--
-- 背景：套餐此前只有「整套总价」一个维度（services.price），无法表达「¥10/张」这类
-- 按张计价的套餐，预约时的总价也只能等于套餐价。本迁移给 services 加计价单位与张数，
-- 并给 bookings 加同名字段作为下单快照（套餐价格/单位改动后，历史订单的算法不被篡改）。
--
-- 幂等性：本项目迁移靠手工 psql 应用，没有 migrate CLI，也没有 schema_migrations 表，
-- 因此全部使用 IF NOT EXISTS / 先 DROP 再 ADD 的写法，可重复执行。
--
-- price_unit 语义：
--   fixed     = 整套价（历史行为，全部旧数据都是这个）
--   per_photo = 按张计价，price 是单价，photo_count 是包含/起拍张数（NULL = 按需议定）
--   per_hour  = 按小时计价
--   per_day   = 按天计价
-- photo_count 只对 per_photo 有意义；其它单位下应为 NULL（不强制，避免历史写坏数据回滚失败）。

ALTER TABLE services ADD COLUMN IF NOT EXISTS price_unit VARCHAR(10) NOT NULL DEFAULT 'fixed';
ALTER TABLE services ADD COLUMN IF NOT EXISTS photo_count INTEGER;

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS photo_count INTEGER;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_unit VARCHAR(10) NOT NULL DEFAULT 'fixed';

-- 计价单位枚举：只允许四种，非法值在写入时就炸，而不是等到前端渲染出乱码。
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_price_unit_check;
ALTER TABLE services ADD CONSTRAINT services_price_unit_check
    CHECK (price_unit IN ('fixed', 'per_photo', 'per_hour', 'per_day'));

-- 张数必须为正整数（NULL 表示议定）。
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_photo_count_check;
ALTER TABLE services ADD CONSTRAINT services_photo_count_check
    CHECK (photo_count IS NULL OR photo_count > 0);

-- 按张计价时单价必须 > 0：¥0/张 没有业务含义（互勉走 price = 0 + price_unit = 'fixed'）。
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_per_photo_price_check;
ALTER TABLE services ADD CONSTRAINT services_per_photo_price_check
    CHECK (price_unit <> 'per_photo' OR (price IS NOT NULL AND price > 0));

ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_price_unit_check;
ALTER TABLE bookings ADD CONSTRAINT bookings_price_unit_check
    CHECK (price_unit IN ('fixed', 'per_photo', 'per_hour', 'per_day'));

ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_photo_count_check;
ALTER TABLE bookings ADD CONSTRAINT bookings_photo_count_check
    CHECK (photo_count IS NULL OR photo_count > 0);
