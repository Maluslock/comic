# 过期漫展过滤 (del_flag) + 横幅联动修复 设计

**日期:** 2026-08-05
**状态:** 用户批准 ("可以，你先做")

## 背景

1. 首页近期漫展显示 "0天后" — DB 中多个事件已过期（start_date < now）但 status 仍为 'upcoming'，`upcomingEvents` 只按 status 过滤，不过滤日期
2. 横幅滚动图片与近期漫展脱节 — banners 的 linkId 指向事件，但事件列表按日期排序后可能已过期

## 设计

### A. 过期事件软删除 (del_flag)

- 迁移 `000005_event_del_flag.up.sql`：
  ```sql
  ALTER TABLE comic_events ADD COLUMN IF NOT EXISTS del_flag BOOLEAN DEFAULT false;
  ```
- 后端所有事件查询（GetUpcomingEvents/GetEventById/GetEventByAllcppId/SearchEvents/GetAllEvents）加 `WHERE del_flag = false`（或 AND del_flag = false）
- ingest upsert 后：`UPDATE comic_events SET del_flag = true WHERE start_date < NOW() AND del_flag = false;`（标记过期）
- 前端 `upcomingEvents` computed 加 `e.startDate > Date.now()`（最终防线）

### B. 横幅↔事件联动

- 后端 home service：banners 只保留 linkId 指向**存在且未过期**（del_flag=false AND start_date > now）的事件；否则剔除该 banner
- 前端 `onBannerClick`：找不到事件时不导航（防空，已有 guard，确认即可）

## 文件清单

| 文件 | 改动 |
|------|------|
| `server/migrations/000005_event_del_flag.up.sql` | 新建：加列 |
| `server/internal/repository/events.sql.go` + `db/query/events.sql` | 5 个查询加 del_flag 过滤 |
| `server/internal/service/home_service.go` | banners 联动过滤 |
| `server/cmd/ingest/main.go` | upsert 后标记过期 |
| `src/stores/home.ts` | upcomingEvents 加日期过滤 |

## 验收

1. `SELECT count(*) FROM comic_events WHERE del_flag = true` → 过期事件被标记
2. `GET /api/v1/events` 不含 del_flag=true 事件
3. 首页无 "0天后" 事件
4. banner 只显示未过期事件
