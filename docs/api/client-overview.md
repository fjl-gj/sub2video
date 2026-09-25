# 客户端 · 首页概览

## 页面职责

创作者登录后的默认落地页。汇总账户、计费、存储与任务状态，提供快捷入口进入**任务**与**创作**，不承载配置或编辑操作。

对应产品定义见 `docs/产品方案总览.md`：查看余额、进行中任务、最近结果；广场灵感入口可在后续迭代补充。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/overview` |
| 页面组件 | `frontend/src/pages/client/OverviewPage.vue` |
| 当前数据来源 | `GET /api/v1/client/overview`；Pinia 仅缓存余额、容量和工程状态 |

## 页面区块与数据映射

```text
┌─ PageHeader ─────────────────────────────────────────────┐
│ title: 欢迎回来，{workspace.displayName}                  │
│ actions: → /app/works | → /app/create                    │
└──────────────────────────────────────────────────────────┘
┌─ metrics[5] ─────────────────────────────────────────────┐
│ 累计支出 | 可用余额 | 每秒视频均价 | 空间资源 | 进行中任务 │
└──────────────────────────────────────────────────────────┘
┌─ activeTaskPanel ────────────────────────────────────────┐
│ 进行中的任务（首条 active，无则空态）                      │
└──────────────────────────────────────────────────────────┘
┌─ recentCompletedTasks[0..2] ────────────────────────────┐
│ 最近完成任务卡片列表                                       │
└──────────────────────────────────────────────────────────┘
```

### 1. 页头 `header`

| UI 字段 | 类型 | 说明 | 当前实现 |
| --- | --- | --- | --- |
| `workspace.displayName` | `string` | 工作区/创作者显示名 | 后端工作区名称 |
| — | — | 「查看任务队列」跳转 `/app/works` | 纯导航 |
| — | — | 「新建生成任务」跳转 `/app/create` | 纯导航 |

### 2. 指标卡 `metrics[]`

固定 5 项，顺序与前端一致。

| `label`（展示） | 响应字段 | 类型 | 说明 |
| --- | --- | --- | --- |
| 累计支出 | `billing.totalSpent` | `number` | 历史任务消费总额（不含充值） |
| （副文案） | `billing.monthSpent` | `number` | 自然月内消费额 |
| 可用余额 | `wallet.balance` | `number` | 可下单余额 |
| 每秒视频均价 | `pricing.available` | `boolean` | 当前是否存在可报价且可执行的创作能力；为 `false` 时 UI 显示不可用 |
| | `pricing.referencePricePerSecond` | `number` | `available = true` 时的最低有效参考价；实际扣费以任务提交时快照为准 |
| （副文案） | `pricing.referenceNote` | `string` | 如「768P / 1080P 按秒计」 |
| 空间资源占用 | `storage.usedGb` | `number` | 已用容量 |
| | `storage.quotaGb` | `number` | 套餐配额 |
| （副文案） | `storage.fileCount` | `integer` | 已保存文件数（跨作品） |
| 进行中的任务 | `tasks.activeCount` | `integer` | `status ∈ {排队中, 生成中}` 的数量 |
| （副文案） | — | — | 「查看队列 →」跳转 `/app/works` |

**交互：** `wallet.balance` 卡片副文案为「立即充值 →」，跳转 `/app/wallet`。

### 3. 进行中任务 `activeTask`

取活跃任务列表的第一条（建议后端按 `updatedAt` 降序）。无活跃任务时返回 `null`，前端展示空态。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 任务 ID，如 `tsk_88921` |
| `title` | `string` | 作品/任务标题 |
| `mode` | `string` | 创作模式展示名，如 `首尾帧生成视频` |
| `status` | `TaskStatus` | 见枚举 |
| `progress` | `integer` | 0–100 |
| `duration` | `string` | 展示用时长，如 `15 秒` |
| `resolution` | `string` | 如 `768P` |
| `cost` | `number` | 已预扣金额 |
| `thumbUrl` | `string` | 缩略图 URL（可选，概览行未展示但任务域共有） |

前端组件：`ActiveTaskRow`（`frontend/src/features/overview/ActiveTaskRow.vue`）。

### 4. 最近完成任务 `recentCompletedTasks[]`

最多 2 条，`status = 已完成`，按完成时间降序。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 任务 ID |
| `title` | `string` | 标题 |
| `status` | `TaskStatus` | 固定为 `已完成`；UI 展示为「生成成功」 |
| `resolution` | `string` | 分辨率 |
| `duration` | `string` | 时长 |
| `cost` | `number` | 实际扣费 |
| `thumbUrl` | `string` | 缩略图 |
| `completedAt` | `string` | ISO 8601（列表副信息可扩展） |

卡片操作「下载」：后续对接 `GET /api/v1/client/tasks/{id}/download` 或签名 URL。

## 枚举

```ts
type TaskStatus = '排队中' | '生成中' | '已完成' | '失败'
```

活跃任务：`status ∈ {'排队中', '生成中'}`。

## 推荐接口

### 方案 A：聚合接口（推荐）

首屏一次请求，减少瀑布流。

```
GET /api/v1/client/overview
Authorization: Bearer <token>
```

**Response `200`**

```json
{
  "workspace": {
    "displayName": "李维工作室"
  },
  "billing": {
    "totalSpent": 148.0,
    "monthSpent": 36.0
  },
  "wallet": {
    "balance": 256.0
  },
  "pricing": {
    "available": true,
    "referencePricePerSecond": 0.2,
    "referenceNote": "768P / 1080P 按秒计"
  },
  "storage": {
    "usedGb": 1.9,
    "quotaGb": 5.0,
    "fileCount": 6
  },
  "tasks": {
    "activeCount": 1,
    "activeTask": {
      "id": "tsk_88921",
      "title": "雨夜街道霓虹漫步",
      "mode": "首尾帧生成视频",
      "status": "生成中",
      "progress": 68,
      "duration": "15 秒",
      "resolution": "768P",
      "cost": 4.5,
      "thumbUrl": "https://..."
    },
    "recentCompleted": [
      {
        "id": "tsk_88918",
        "title": "产品桌面柔光展示",
        "status": "已完成",
        "resolution": "768P",
        "duration": "15 秒",
        "cost": 4.5,
        "thumbUrl": "https://...",
        "completedAt": "2026-08-31T13:42:00+08:00"
      }
    ]
  }
}
```

| 字段 | 可空 | 说明 |
| --- | --- | --- |
| `tasks.activeTask` | 是 | 无活跃任务时为 `null` |
| `tasks.recentCompleted` | 否 | 无数据时 `[]` |

## 业务规则

1. **余额与预扣：** 进行中任务展示的 `cost` 为提交时已预扣金额；失败或终止后按钱包规则退款（见钱包域）。
2. **参考单价：** 概览仅展示当前可报价、可执行组合的最低参考价；没有可用组合时 `pricing.available = false`，页面不得回退为固定价格。实际创作先请求无扣费的 `tasks/estimate`，提交时服务端重新路由并固化售价、成本和执行配置快照，与概览解耦。
3. **活跃任务数：** 与侧栏「任务」角标、`tasks.activeCount` 一致，均统计 `排队中` + `生成中`。
4. **存储统计：** `usedGb` / `fileCount` 跨所有未删除作品；删除作品进入回收站后是否计入配额由存储域定义（建议：回收站占用配额直至清空）。
5. **刷新策略：** 进行中任务建议 WebSocket / SSE 推送 `progress`，或前端轮询（间隔 ≥ 5s）；指标卡可在进入页面时拉取一次，从任务页返回时可 `invalidate`。

## 当前前端状态映射

| 接口字段 | 当前前端状态 |
| --- | --- |
| `wallet.balance` | 概览响应写回 `app.balance` 与认证余额 |
| `storage.*` | 概览响应写回 `app.storageUsed`、`app.storageQuota` |
| `tasks.*` | 直接使用概览响应；任务页另行分页拉取 |
| `billing.*` | 后端按钱包流水汇总 |
| `pricing.*` | 后端当前可执行组合的参考价与可用性 |
| `workspace.displayName` | 后端工作区名称 |

类型定义参考：`frontend/src/entities/types.ts` → `Task`、`TaskStatus`。

## 关联导航数据（非本页主体）

进入概览时，顶栏与侧栏还依赖：

| 位置 | 数据 | 建议来源 |
| --- | --- | --- |
| 顶栏余额 pill | `wallet.balance` | 与 `metrics` 共用，可全局 store |
| 侧栏任务角标 | `tasks.activeCount` | 与 `metrics` 一致 |
| 侧栏工单角标 | `tickets.openCount` | `GET /api/v1/client/tickets/summary` |
| 侧栏回收站角标 | `trash.projectCount` | 资产域 |

可在 `overview` 响应中增加 `navBadges` 字段，或由独立轻量接口提供。

## 待产品确认

- [ ] 是否在概览增加**广场**入口（产品文案含「灵感入口」，当前 Vue 未实现）。
- [ ] 「累计支出」是否包含已退款任务（建议：展示净消费 = 消费 − 退款）。
- [ ] 最近完成列表是否改为「最近保存到作品的视频」而非「最近完成任务」。
