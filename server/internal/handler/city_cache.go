package handler

import "time"

// cityCacheTTL 是城市清单的缓存时长。
//
// 数据源一天只更新两次（cron.yaml 的 events-sync-morning 03:00 / events-sync-afternoon 15:00），
// 10 分钟足够让新增场次在用户无感的时间内出现，也把重聚合查询压成每 10 分钟一次。
const cityCacheTTL = 10 * time.Minute
