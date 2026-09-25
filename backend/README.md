# Sub2Video Backend

Gin + PostgreSQL 客户端 API。从 Sub2API 提取认证/钱包模式，不含 API 网关。

客户端创作采用直链路合同：`/api/v1/client/tasks/estimate` 只计算不扣费；`/api/v1/client/tasks` 携带完整参数和 `Idempotency-Key`，服务端在事务内重新路由、扣费并创建订单与任务。旧 quote/workflow 接口仅保留历史兼容，不属于新客户端流程。

## 栈

| 项 | 选型 |
| --- | --- |
| HTTP | Gin |
| DB | PostgreSQL 16 |
| Auth | JWT access + refresh（bcrypt） |
| 端口 | 默认 `127.0.0.1:12080`（避免与前端 12000 / 常见 8080 冲突） |

## 本地启动

```bash
cd backend
docker compose up -d
go mod tidy
go run ./cmd/server
```

环境变量见 `internal/config/config.go`。默认：

- DB: `postgres://sub2video:sub2video@127.0.0.1:15432/sub2video?sslmode=disable`
- API: `127.0.0.1:12080`
- 演示账号: `demo@sub2video.local` / `Demo1234!`（余额 256）

## 认证接口

```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/me          # Bearer access token
```

## 客户端核心接口

```
GET  /api/v1/client/overview
GET  /api/v1/client/create/catalog
GET  /api/v1/client/create/options
POST /api/v1/client/tasks/estimate        # 只预估，不创建报价、订单或任务
POST /api/v1/client/tasks                 # 完整创作参数 + Idempotency-Key/clientRequestId
GET  /api/v1/client/tasks
GET  /api/v1/client/tasks/:id             # 不返回供应商、成本或上游 URL
POST /api/v1/client/tasks/:id/cancel
POST /api/v1/client/tasks/:id/save-to-project
POST /api/v1/client/uploads
GET/POST/DELETE /api/v1/client/projects/...
GET  /api/v1/client/wallet
GET  /api/v1/client/wallet/transactions
POST /api/v1/client/wallet/redeem
GET/POST /api/v1/client/tickets
```

预估和提交都使用完整的“固定创作模式 → 可选承运商/模型 → 时长/分辨率/画幅/自动音频”参数。预估只返回当前最低成本执行方式对应的对客价格，不生成 `quoteId`，也不锁定执行配置；提交时服务端重新路由，并在同一事务内完成扣费、消费订单和任务创建。客户端不能提交或接收 `execution_type`、供应商、成本、Token、API/workflow 编码或上游地址。

管理端接口位于 `/api/v1/admin/...`，包括创作模式、承运商/模型、供应商供给、采购成本、对客售价、任务、工单、设置与操作审计。旧 `routes`、`workflows`、`pricing` 接口已不再挂载，也不参与启动 seed、报价、任务创建或 Worker 路由；其旧表仅为已部署数据库的迁移兼容保留。

## 执行与交付

- 报价只会选择已启用、已配置凭据、健康可用且由当前 Worker 支持的供应商供给；未部署适配器的供给不能保存为可报价配置。
- 新任务的执行配置、供给、成本与售价都会写入不可变快照；Worker 不会重新按当前最低成本换路由。上游提交前以数据库条件更新原子认领任务，避免多实例重复提交；未取得上游任务 ID 的配置等待状态会释放认领，已提交任务只会继续轮询。
- 缺失锁定供应商凭据或图片任务对象存储时，任务保持排队并标记等待配置，不会模拟成功；上游已完成但对象存储不可用时，任务保持等待持久化交付并自动重试。
- 上游失败会在同一退款语义下写入一次退款流水；连续三次失败会将该供给标为不可用。
- 客户端只可下载 `result_oss_url` 的持久化交付结果，无法下载或保存上游临时 URL。

## 验证

```bash
cd backend
go test ./...
```

真实 PostgreSQL 集成测试默认跳过。仅在专用测试库上执行：

```bash
SUB2VIDEO_RUN_INTEGRATION=1 \
SUB2VIDEO_TEST_DATABASE_URL='postgres://.../sub2video_test?sslmode=disable' \
SUB2VIDEO_TEST_REDIS_URL='redis://127.0.0.1:16379/15' \
go test ./internal/store -run Integration
```

测试库名称必须包含 `test`，避免误连开发或生产数据库；测试在启动时会连接并 Ping Redis。

真实浏览器集成测试默认跳过，使用独立 Playwright 配置，不注入页面 Mock。它会通过登录页访问真实客户端目录、真实参数能力和无扣费预估；只有显式设置 `SUB2VIDEO_LIVE_CREATE_TASKS=1` 才会提交真实任务并产生余额消费：

```bash
cd frontend
SUB2VIDEO_LIVE=1 \
SUB2VIDEO_LIVE_BASE_URL='http://127.0.0.1:4185' \
SUB2VIDEO_LIVE_EMAIL='...' \
SUB2VIDEO_LIVE_PASSWORD='...' \
pnpm test:integration
```

前端应代理到真实后端 `127.0.0.1:12080`，或把 `SUB2VIDEO_LIVE_BASE_URL` 指向已经部署前端的 UAT 地址。测试库、供应商凭据和对象存储未配置时，测试会明确失败或停留在不可用状态，不会伪造通过。

部署后的 UAT 预检会登录测试账号，检查三种固定模式（文生、首尾帧、多图）的目录、能力与无扣费预估合同，但不创建任务或预扣余额：

```bash
SUB2VIDEO_UAT_BASE=http://host:12000/api/v1 python3 backend/scripts/verify_uat_create.py
```

完整任务验收需要显式开启，并只创建一个文生视频测试任务，预扣 UAT 钱包余额，等待 Worker 和对象存储完成持久化交付，且可能调用已配置的上游供应商。仅在确认测试账号余额、供应商额度和对象存储均可消耗后执行：

```bash
SUB2VIDEO_UAT_BASE=http://host:12000/api/v1 python3 backend/scripts/verify_uat_create.py --create-tasks
```

## 107 部署

1. `docker compose up -d` 在 107 上起 PG/Redis（端口仅绑 127.0.0.1）
2. `GOOS=linux GOARCH=amd64 go build -o sub2video-server ./cmd/server`
3. 监听 `127.0.0.1:12080`，Nginx `/api` 反代
4. systemd 单元见 `deploy/sub2video-backend.service`
