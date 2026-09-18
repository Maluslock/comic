# 城市弹窗修复 + 事件标题省略 + 定时抓取 + 城市标签 设计

**日期:** 2026-08-05
**状态:** 已批准 (用户确认 "好的")

## 背景

用户测试发现 2 个 UI bug + 2 个数据功能需求：

1. **城市选择弹窗不能下滑** — 城市列表超出屏幕时无法滚动
2. **近期漫展标题右侧被吞字** — `<text>` inline 元素上 `.ellipsis` 不生效
3. **数据集太小** — 需要定期自动抓取更多漫展（图片/文字详情）
4. **城市标签关联** — 抓取的数据需要城市↔tag 关联

## 设计

### A. 城市弹窗滚动修复

`src/pages/index/index.vue`：
- `.city-popup` 加 `max-height: 70vh; display: flex; flex-direction: column;`
- `.city-list` 加 `overflow-y: auto; flex: 1;`
- 效果：弹窗最高 70% 屏高，城市多时列表内部滚动，头部固定

### B. 事件标题省略号修复

`src/pages/index/index.vue` `.event-name`：
- 加 `display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;`
- 根因：`.ellipsis` 的 overflow 规则对 inline `<text>` 不生效
- 顺带检查 event/detail 等页面是否有同样问题（photographer 卡片 name 已有 flex:1 保护）

### C. 定时抓取程序

扩展 `server/cmd/ingest/main.go`：
- 新增 `-schedule` flag（cron 表达式，如 `0 3 * * *` 每天凌晨3点）
- 用 `github.com/robfig/cron/v3`（需添加依赖）或简单 time.Ticker 每日循环
- 数据源：
  - **nyato.com**（现有）— 每日 1 次，多页抓取
  - **B站漫展**（新增）— `https://www.bilibili.com/` 漫展专题/活动页，每周 1 次
  - 抓取内容：名称、城市、日期、地址、封面图 URL、详情描述
- 复用现有 `UpsertEventFromIngest`（幂等，allcpp_id = FNV(name+city)）
- 新增字段：`description`（详情描述，DB 已有？需确认 comic_events 是否有 description 列）

### D. 城市↔标签关联

- 抓取时事件自带 `location`（城市）
- 城市选择器数据源：从 `GET /api/v1/events` 返回的城市列表实时生成（去重），不再硬编码
- 前端：`src/pages/index/index.vue` 的 `cityList` 从 store 事件动态计算
- 可选：`tags` 数组自动 append 城市名（如 "上海" 展 → tags 含 "上海"）

## 文件清单

| 文件 | 改动 |
|------|------|
| `src/pages/index/index.vue` | A+B+D：弹窗 CSS、事件名 CSS、城市列表动态化 |
| `server/cmd/ingest/main.go` | C：-schedule flag + 每日循环 + B站源 |
| `server/internal/ingest/` | C：新增 bilibili.go 抓取器（可选） |
| `src/api/index.ts` 或 store | D：城市列表 API |
| `server/migrations/` | C：description 列（如缺） |

## 验收标准

1. 城市弹窗 8+ 城市时能滚动
2. 长标题显示 `...` 省略号
3. `go run ./cmd/ingest -schedule="0 3 * * *"` 能启动定时任务（或用 -once 手动触发验证）
4. 城市选择器选项来自 DB 事件（新增城市自动出现）
