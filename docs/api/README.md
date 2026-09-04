# Sub2Video API 文档

API 前缀为 `/api/v1`。服务端使用 Gin + PostgreSQL；前端页面已通过 `frontend/src/api/` 调用 API。创作供给、报价和任务合同以 [创作模式与供应商路由设计](../创作模式与供应商路由设计.md) 为准。

## 客户端

| 文档 | 前端路由 | 核心接口 |
| --- | --- | --- |
| [首页概览](./client-overview.md) | `/app/overview` | `GET /client/overview` |
| [创作](./client-create.md) | `/app/create` | `GET /client/create/catalog`、`GET /client/create/options`、`POST /client/tasks/quote`、`POST /client/tasks` |
| [广场](./client-plaza.md) | `/app/garden` | 场景与预设提示词 |
| [任务](./client-tasks.md) | `/app/works` | `GET /client/tasks` |
| [资产](./client-assets.md) | `/app/assets` | `GET /client/projects` |
| [回收站](./client-trash.md) | `/app/trash` | `GET /client/projects?trashed=true` |
| [钱包](./client-wallet.md) | `/app/wallet` | `GET /client/wallet` |
| [帮助与工单](./client-tickets.md) | `/app/tickets` | `GET /client/tickets` |

## 鉴权与通用约定

- 客户端和管理端接口都需要 JWT（登录、注册等认证接口除外）。
- 金额使用 CNY 元的 `number`；时间使用 ISO 8601。
- 客户端不得收到供应商名称、成本、密钥、连接地址、上游工作流 ID、供应商供给 ID 或路由理由。
- 管理端接口位于 `/api/v1/admin/...`，可管理创作模式、承运商/模型、供应商供给/成本和平台售价。
- 列表响应含 `items`；分页接口含 `total`、`page`、`pageSize`。

## 创作主链路

```text
GET catalog → 选择固定模式 / 模型 → GET options
  → POST quote（服务器选择锁定供给）
  → POST tasks { quoteId, prompt, sourceMeta }
  → 事务预扣、任务快照、异步执行
```

报价有效期 5 分钟且只能使用一次。客户端提交任务时不能带价格快照或路线信息。
