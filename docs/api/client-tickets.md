# 客户端 · 帮助与工单

## 页面职责

提交售后、生成异常、售前、充值类工单；查看历史工单状态。可关联任务 ID。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/tickets` |
| 页面组件 | `frontend/src/pages/client/TicketsPage.vue` |
| 侧栏角标 | 未解决工单数 `openClientTicketCount` |
| 分页 | `pageSize = 10` |

## 业务流程

```text
新建工单（展开表单）
  → 选分类、填标题、可选 taskId、填详情
  → POST tickets
  → 列表置顶，状态「待处理」

点击列表项
  → 后续：工单详情 / 会话线程（原型为 toast）
```

## 工单对象 `ClientTicket`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `TKT-*` |
| `title` | `string` | 标题 |
| `category` | `string` | 售后支持 / 生成异常 / 售前咨询 / 充值问题 |
| `status` | `TicketStatus` | 待处理 / 处理中 / 已解决 |
| `updated` | `string` | 最后更新时间展示 |
| `task` | `string` | 关联任务 ID，无则 `-` |
| `detail` | `string` | 详情（列表页不展示，详情页用） |

```ts
type TicketStatus = '待处理' | '处理中' | '已解决'
```

侧栏角标：`status !== '已解决'` 的数量。

## 创建表单

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `category` | 是 | 四选一 segmented |
| `title` | 是 | 工单标题 |
| `taskId` | 否 | 如 `tsk_88840` |
| `detail` | 否 | 问题描述（原型提交时未传 store，后端应保存） |

## 推荐接口

### 列表

```
GET /api/v1/client/tickets?page=1&pageSize=10
```

```json
{
  "items": [
    {
      "id": "TKT-1028",
      "title": "任务生成失败，如何申请退款？",
      "category": "售后支持",
      "status": "处理中",
      "task": "tsk_88840",
      "updated": "2026-09-01T18:12:00+08:00"
    }
  ],
  "total": 2,
  "openCount": 1
}
```

### 创建

```
POST /api/v1/client/tickets
Content-Type: application/json

{
  "category": "售后支持",
  "title": "任务生成失败，如何申请退款？",
  "taskId": "tsk_88840",
  "detail": "任务失败后余额未退回……"
}
```

```json
{
  "id": "TKT-1030",
  "status": "待处理",
  "createdAt": "2026-09-01T20:00:00+08:00"
}
```

创建后同步管理端工单中心（`adminTickets`）。

### 详情（后续）

```
GET /api/v1/client/tickets/{id}
```

含回复线程、关联订单/任务摘要、退款进度。

### 摘要（导航角标）

```
GET /api/v1/client/tickets/summary
→ { "openCount": 1 }
```

## 业务规则

1. 生成异常 / 售后类工单可关联 `taskId`，便于客服定位与退款。
2. 管理端「核准退款」会更新客户端工单为已解决并入账（见 `approveTicketRefund`）。
3. 列表无「我的工单」分组头；直接展示分页列表。
4. `detail` 字段原型未写入 mock，接口设计需包含。

## 与当前 mock 对应

| 操作 | store |
| --- | --- |
| 列表 | `tickets` |
| 创建 | `createTicket(title, category, taskId)` |
| 未读数 | `openClientTicketCount` |

管理端对称文档待写：`docs/api/admin-tickets.md`。
