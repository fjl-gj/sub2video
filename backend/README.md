# Sub2Video Backend

Gin + PostgreSQL 客户端 API。从 Sub2API 提取认证/钱包模式，不含 API 网关。

客户端创作采用直链路合同：`/api/v1/client/tasks/estimate` 只计算不扣费；`/api/v1/client/tasks` 携带完整参数和 `Idempotency-Key`，服务端在事务内重新路由、扣费并创建订单与任务。旧 `/api/v1/client/tasks/quote` 仅保留路由占位并返回 `410 LEGACY_QUOTE_DISABLED`；旧表和内部方法只用于已部署历史数据，不属于新客户端流程。

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
POST /api/v1/client/tasks/quote           # 410，旧报价合同已停用
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

当前封闭 Beta 的真实执行路径为已验证的 AutoDL `workflow + autodl_h3_v1`；数据库支持 `api` 执行类型，但 AutoDL 独立 API 的 submit/poll 协议尚未验证，因此该组合不会进入客户端目录、估价或提交路由。接入 API 前必须先实现独立适配器并完成契约测试和真实 UAT。

管理端接口位于 `/api/v1/admin/...`，包括创作模式、承运商/模型、供应商供给、采购成本、对客售价、任务、工单、设置与操作审计。旧 `routes`、`workflows`、`pricing` 接口已不再挂载，也不参与启动 seed、报价、任务创建或 Worker 路由；其旧表仅为已部署数据库的迁移兼容保留。

## 执行与交付

- 报价只会选择已启用、已配置凭据、健康可用且由当前 Worker 支持的供应商供给；未部署适配器的供给不能保存为可报价配置。
- 新任务的执行配置、供给、成本与售价都会写入不可变快照；Worker 不会重新按当前最低成本换路由。提交与轮询分别使用有期限的租约，并用认领令牌阻止旧 Worker 覆盖新状态。提交前等待配置可以恢复；请求可能已发出但受理结果不确定时进入 `SUBMIT_UNKNOWN`，不会自动重新提交或退款。
- 缺失锁定供应商凭据或任务对象存储时，任务保持排队并标记等待配置，不会模拟成功；上游已完成但对象存储不可用时，任务保持等待持久化交付并自动重试。估价会先确认供应商凭据和对象存储已配置，提交时再确认对象存储桶可用。
- 上游失败会在同一退款语义下写入一次退款流水；连续三次失败会将该供给标为不可用。
- 客户端只可下载 `result_oss_url` 的持久化交付结果，无法下载或保存上游临时 URL。
- 历史结果地址可能含有已过期的签名。任务详情、列表缩略地址与资产下载会在完成归属检查后，从当前存储的可信 bucket 地址提取对象 key，生成有效期 1 小时的新地址；不写回任务或资产，不依赖旧签名仍有效。任务详情与资产下载签名失败返回 `503 DELIVERY_UNAVAILABLE`，不回退到旧地址或未签名私有对象地址。更换存储端点/桶、删除对象或浏览器无法连接存储仍需单独处理。
- 图片工作流会把对象存储的签名素材 URL 交给云端供应商。后端能连接内网 MinIO，不代表云端也能读取该 URL；真实图生验收还需要供应商可访问的素材域名和有效签名。

### 提交结果不确定的恢复

管理端任务详情提供“核对上游任务”操作，对应 `POST /api/v1/admin/tasks/:id/reconcile`，要求管理员身份、明确确认和 10–1000 字核对依据：

- `action=resume`：填写在锁定供应商核实过的 `upstreamTaskId`；服务端查询确认该 ID 存在后接续轮询。若已记录受理 ID，不得替换为另一个 ID；同一供应商任务不可关联给两个本地任务。
- `action=refund`：仅在已核实供应商未受理、且系统没有已知受理 ID 时允许退款。不得把超时、4xx、查不到一次或网络故障当作未受理证据。
- 操作与任务、订单、退款流水、执行事件、管理员审计在同一事务。相同核对操作可幂等重放；冲突操作返回 409。没有“盲目重新提交”操作。
- 已开始提交的任务禁止普通取消退款。等待核对时应保留证据并联系供应商，不得直接修改数据库解锁或退款。

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
go test -count=1 -timeout=20m ./...
```

测试库名称必须包含 `test`，避免误连开发或生产数据库；测试在启动时会连接并 Ping Redis。

下载续签的 HTTP/归属集成测试只需要真实 PostgreSQL，不启动 Worker、不调用供应商、不消耗余额。它在专用测试库内建立随机 schema，完成后清理该 schema；MinIO 的 bucket-location 查询使用本地测试服务，实际对象交付还需通过 UAT 验证：

```bash
SUB2VIDEO_RUN_INTEGRATION=1 \
SUB2VIDEO_TEST_DATABASE_URL='postgres://.../sub2video_test?sslmode=disable' \
go test ./internal/handler -run '^TestDeliveryURLHandlerIntegration$' -count=1 -v
```

真实浏览器集成测试默认跳过，使用独立 Playwright 配置，不注入业务 Mock。`live-creation.spec.ts` 在桌面和移动尺寸登录、选择素材并检查三模式估价，同时核对钱包和任务未改变；脚本在发送前拦截任何任务提交，不存在付费开关。`live-upload.spec.ts` 单独验证真实图片上传、鉴权回读和字节一致性，上传文件保留在测试环境目录中，不能把本地素材上传称为上游交付：

```bash
cd frontend
SUB2VIDEO_LIVE=1 \
SUB2VIDEO_LIVE_BASE_URL='http://127.0.0.1:4185' \
SUB2VIDEO_LIVE_EMAIL='...' \
SUB2VIDEO_LIVE_PASSWORD='...' \
pnpm test:integration tests/integration/live-creation.spec.ts tests/integration/live-upload.spec.ts
```

前端应代理到真实后端 `127.0.0.1:12080`，或把 `SUB2VIDEO_LIVE_BASE_URL` 指向已经部署前端的 UAT 地址。测试库、供应商凭据和对象存储未配置时，测试会明确失败或停留在不可用状态，不会伪造通过。

真实 MinIO 测试另需注入 `SUB2VIDEO_TEST_MINIO_ENDPOINT`、`SUB2VIDEO_TEST_MINIO_ACCESS_KEY`、`SUB2VIDEO_TEST_MINIO_SECRET_KEY`、`SUB2VIDEO_TEST_MINIO_BUCKET`（名称必须含 `test`）和 `SUB2VIDEO_TEST_MEDIA_FILE`。测试仅上传到随机独立 key，核对签名下载字节与私有访问权限，清理自己创建的对象；既有媒体是测试夹具，不代表供应商生成的视频。

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

已初始化环境、且本次发布没有表结构变更时，可设置 `SUB2VIDEO_SKIP_BOOTSTRAP=true`。此选项跳过启动迁移、种子数据、演示用户和自动管理员提升；保留正常数据库/Redis 连接、Worker 和 HTTP 服务。它不是迁移执行器，也不会自动检查数据库版本；首次部署或后续新增迁移时，必须先制定并执行迁移方案，不能直接沿用此选项后假设 schema 已更新。

候选二进制支持显式迁移命令：

```bash
/path/to/candidate/sub2video-server --migrate-only
```

在受控发布窗口中，先备份并核对数据库，再向该进程安全注入目标 `SUB2VIDEO_DATABASE_URL`。此命令即使服务配置了 `SUB2VIDEO_SKIP_BOOTSTRAP=true` 也会执行现有迁移器，完成后退出；不会运行 seed、演示账号、管理员提升、Redis、上传目录初始化、对象存储、Worker 或 HTTP。未知参数在数据库连接前报错。不要为了执行迁移临时启动使用 UAT 数据库的完整候选服务。

2026-09-29 只读核对的 107 数据库为 001–019，缺少 checksum 列；候选还需要 020（执行配置成本唯一约束和旧 offering 关联可空）、021（任务租约字段和执行事件表）。不能直接替换二进制后沿用 `SKIP_BOOTSTRAP=true` 启动。020 前需排查执行配置/分辨率/音频重复成本；历史 checksum 首次补录不代表旧 SQL 已被独立验证。升级后可保持运行服务的 `SKIP_BOOTSTRAP=true`，避免旧版本回滚时重放历史种子和迁移；程序回滚仍需检查新版在途任务，不能直接恢复旧数据库覆盖新增订单和流水。

`TestUpgradeFrom019PreservesExistingDataIntegration` 用隔离随机 schema 与人工数据验证十九版升级、重复执行和 checksum 冲突；`TestMigrateOnlyCLIHasNoServiceSideEffectsIntegration` 启动候选命令子进程，使用不可连接 Redis/HTTP 配置，验证仅迁移且不运行 seed 或创建上传目录。这是代码和迁移演练证据，不等于既有 UAT 业务数据库已升级。

当前工作区的迁移执行器按 `schema_migrations` 跳过已执行版本，并校验 SQL checksum；首次迁移按事务执行。历史无 checksum 记录会补记当前摘要，这不能证明旧库当时执行的 SQL 与当前文件相同，升级前仍需核对版本并备份。种子只填充缺失的成本/售价默认项，不覆盖管理员已配置价格。无 schema 变更的 UAT 热修复可继续使用上述跳过初始化选项；旧版二进制可能有重放迁移副作用，回滚前也需核对。真实隔离测试不能代替既有 UAT 数据库升级演练。
