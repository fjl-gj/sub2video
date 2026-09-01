# Sub2Video API 文档

本目录按**客户端菜单**梳理接口与数据结构，供后台设计与联调。

**页面操作流程**（用户从哪进、点什么、跳到哪）见 [`docs/flows/客户端页面流程.md`](../flows/客户端页面流程.md)。

## 客户端

| 文档 | 页面 | 前端路由 | 核心接口前缀 |
| --- | --- | --- | --- |
| [首页概览](./client-overview.md) | 首页概览 | `/app/overview` | `GET /client/overview` |
| [创作](./client-create.md) | 创作 | `/app/create` | `GET /client/create/form`, `POST /client/tasks` |
| [广场](./client-plaza.md) | 广场 | `/app/garden` | `GET /client/scenes` |
| [任务](./client-tasks.md) | 任务 | `/app/works` | `GET /client/tasks` |
| [资产](./client-assets.md) | 资产 | `/app/assets` | `GET /client/projects` |
| [回收站](./client-trash.md) | 回收站 | `/app/trash` | `GET /client/projects?trashed=true` |
| [钱包](./client-wallet.md) | 钱包 | `/app/wallet` | `GET /client/wallet` |
| [帮助与工单](./client-tickets.md) | 帮助与工单 | `/app/tickets` | `GET /client/tickets` |

个人资料从顶栏进入（`/app/profile`），文档待补。

## 客户端业务主链路

```text
广场 ──draftPrompt──► 创作 ──POST tasks──► 任务 ──save──► 资产（作品工程）
                              │                    │
                              └──── 钱包预扣/退款 ◄┘
资产 ──trash──► 回收站 ──restore/purge──► 资产
任务/钱包异常 ──► 帮助与工单
```

## 全局约定

- 路径前缀建议为 `/api/v1/client/...`（客户端）与 `/api/v1/admin/...`（管理端）。
- 金额单位为元（`number`，保留两位小数）；存储单位为 GB（`number`）。
- 时间字段对外使用 ISO 8601；页面展示文案由前端按 locale 格式化。
- 列表分页默认 `pageSize = 10`，响应含 `total`、`page`、`pageSize`。
- 本文档描述**产品语义**，不等同于已实现的后端；当前前端为 Pinia mock。

## 共享领域对象

| 对象 | 定义位置 | 文档 |
| --- | --- | --- |
| `Task` / `TaskStatus` | `frontend/src/data/mock.ts` | 任务、首页概览、创作 |
| `WorkProject` / `FsItem` | `frontend/src/data/projects.ts` | 资产、回收站、任务保存 |
| `LedgerEntry` | `frontend/src/data/mock.ts` | 钱包、首页概览 |
| `ClientTicket` | `frontend/src/data/mock.ts` | 工单 |

## 状态

| 状态 | 含义 |
| --- | --- |
| 草案 | 已与当前 Vue 页面对齐，待评审 |
| 待定 | 产品规则未最终确认（各文档「待产品确认」节） |
