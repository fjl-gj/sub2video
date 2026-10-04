# 封闭 Beta 候选验收记录 · 2026-09-28

本记录只对应当前工作区，不沿用历史 UAT 生成成功作为证据。当前结论：最终真实数据库全量测试、真实存储与免费浏览器预检已通过；真实供应商三模式交付尚未执行，不能宣布达到封闭 Beta 门槛。

## 版本、环境与边界

- 分支：`codex/commercial-studio-redesign`；基础提交：`a479c33`，包含既有未提交协作修改及本轮修复。未提交、未推送、未部署。
- 当前 257 个后端/前端源文件、测试和依赖配置的内容清单 SHA-256：`91383294c408e37465d11af1e2de69d3ad51f3d51d9f8bf9afe071676db1f4a4`。清单按相对路径排序，每行是路径加文件 SHA-256；不包含文档、截图、临时输出、数据和可执行文件，不把脏工作区误标为单一已提交版本。
- 本地新版 API：`127.0.0.1:22181`；Vite：`127.0.0.1:4187`。
- 107 独立 PostgreSQL 16：端口 `25432`，数据库 `sub2video_beta_test`；独立 Redis 7.4：端口 `26379`。数据库测试另建随机 schema，完成后删除各自 schema；不执行共享 Redis flush。
- 107 独立 MinIO：端口 `29000`，独立数据卷；真实存储测试使用 `sub2video-beta-test` 桶内随机 key，完成后删除该 key。
- 验证工具：Windows/amd64 Go 1.25.1，pnpm 10.32.0，项目锁定的 Playwright/Chromium。
- 专用浏览器账号：`beta-20260928@sub2video.test`，隔离测试余额 ¥10.00；不是充值支付收入。密码与供应商凭据不写入本记录。
- 上传、测试产物放在本轮 `.tmp_beta_20260928` 目录。未触碰 `backend/data/`、`backend/server.linux`。
- 107 原 UAT API/前端、数据库、Redis、对象和 systemd 服务没有重启或覆盖；检查时原服务 MainPID 为 `3222986`。

## 审计差距与处理

| 当前审计发现 | 目标 / 本轮处理 | 文件与数据表 | 新迁移 | 风险与验证 |
| --- | --- | --- | --- | --- |
| 旧 quote 仍有历史结构，部分说明已过期 | 正式路径只走 estimate/tasks；旧接口 410，历史表保留；更新说明 | handler/client.go、server/router.go、README；task_quotes | 否 | 410 合同测试；估价前后 task/order/quote/ledger 数量不变 |
| 部分集成预期曾允许未实现 API 获选 | 不支持的 adapter/API 必须过滤，已验证 workflow 按成本、priority、稳定 ID 排序 | store/creation_catalog_integration_test.go；supplier_executions、supplier_cost_rules | 否 | 低成本未支持 API 与有效 workflow 混合候选真实测试 |
| 目录、选项、推荐对禁用状态/额度/有效价格的过滤不一致 | 修复共用状态和有效成本/售价配对，环境凭据 fallback 一致 | creation_catalog.go、direct_routing.go；模式/策略/承运商/模型/供给/价格表 | 否 | 19 项状态矩阵、能力和有效价格组合真实测试 |
| 重复 seed 重置供应商成本、启停和有效期 | 只补缺失默认值，保留管理员配置 | creation_catalog.go；supplier_cost_rules | 否 | 改价/禁用后重复 seed，值必须保留 |
| 首尾帧图数未约束，混合素材槽位可能绕过估价 | 固定模式形状检查；首尾同图仍占两个槽位；估价只校验形状，提交查归属 | tasks.go、creation_catalog.go、direct_routing.go、handler/client.go | 否 | 槽位单测和真实路由能力子场景 |
| 部分路由查询原先不在提交事务内；事务中再借连接可能耗尽池 | 路由与必要配置/归属读复用当前事务，幂等查询在最前 | store/user.go、direct_routing.go 及配置/素材查询；钱包/订单/任务表 | 否 | 并发幂等、回滚和单连接池提交回归 |
| HTTP 错误或受理后落库失败可能被当成普通失败 | 保守记录 SUBMIT_UNKNOWN；不自动退款、不重新提交，保留已知上游 ID | provider/autodl/client.go、worker/provider.go、store/provider_tasks.go | 否 | HTTP 408、受理后 DB 异常、崩溃租约恢复，均只发送一次 |
| 普通取消可能给未知提交退款 | 提交边界后禁止普通取消；用管理员核对恢复 | tasks.go、admin_ops.go、provider_reconciliation.go、管理端任务弹窗；审计/事件/订单/钱包 | 否 | 用户 403、冲突 409、核实恢复 200、重复幂等、审计故障事务回滚 |
| 真实浏览器测试可能用开关直接扣费 | 三模式预检无付费分支；独立上传测试无任务提交 | live-creation.spec.ts、live-upload.spec.ts、Playwright integration config | 否 | 真实登录、参数/素材/估价匹配、余额和任务不变、下载字节哈希 |
| 内网 MinIO 可读写，但云端供应商无法直接访问内网素材 URL | 需要供应商可访问的对象存储地址，完成后再做付费验收 | 对象存储运行配置、Worker 素材 URL；本轮未修改 UAT 配置 | 否 | 阻塞：现有 UAT 与隔离 MinIO 都是 192.168 地址，尚无公网访问证据 |

## 测试证据分层

### 真实数据库与 Redis

正式命令为 `go test -json -count=1 -timeout 20m -p 1 ./...`，注入：

```text
SUB2VIDEO_RUN_INTEGRATION=1
SUB2VIDEO_TEST_DATABASE_URL
SUB2VIDEO_TEST_REDIS_URL
```

最终全量结果：**退出码 0，54 个顶层测试通过，包含子测试共 160 个通过，0 fail、0 test skip**。9 个有测试的包通过；12 个包无测试文件，不计入测试通过或环境 skip。运行时间为 22:45:01–22:55:40（UTC+8），约 639 秒。`-p 1` 只串行不同 Go 包以避免迁移锁与隧道延迟叠加；幂等测试内部的并发请求没有关闭。

其中 Store 完整包 515.196 秒通过；Handler 42.512 秒通过；真实 MinIO 约 9.60 秒通过；Worker 也在本次完整运行中通过。覆盖：重复迁移与种子配置保护、能力过滤、估价后改价重算、并发与顺序幂等、供给下线后原订单重放、余额不足无残留、中途故障整体回滚、取消/失败退款幂等、实际 Worker 按原快照发送 HTTP、提交与轮询租约、崩溃恢复、执行/存储等待超时、存储等待恢复、保存作品幂等，以及任务/资产下载鉴权。

供应商 HTTP 故障用本地 `httptest` 服务注入。它只证明失败处理与账务语义，不证明 AutoDL 真实生成成功。测试 schema 中的任务和订单是测试记录，不能计入真实用户或收入。

最终 Worker 数据库账务证据（测试余额，非真实供应商收费）：

| 场景 | 脱敏任务 / 订单 | 扣费 | 余额变化 | 退款流水 | 订单终态 |
| --- | --- | --- | --- | --- | --- |
| 执行超时，重复 Tick | tsk_…06c2 / ord_…adbb | ¥0.15 | 100.00 → 99.85 → 100.00 | 1 | refunded |
| 存储交付等待超时，重复 Tick | tsk_…7249 / ord_…fcb1 | ¥0.15 | 100.00 → 99.85 → 100.00 | 1 | refunded |
| 快照提交后上游 FAILED，重复轮询 | tsk_…4bdc / ord_…b7c7 | ¥0.15 | 100.00 → 99.85 → 100.00 | 1 | refunded |

受理后本地保存失败、HTTP 408 的测试另断言：只提交一次，保持 `SUBMIT_UNKNOWN`，没有退款；管理员恢复的权限、审计、冲突及幂等也已通过。

### 真实对象存储

`TestRealObjectStorageMediaRoundTripIntegration`：通过。既有许可参考视频夹具上传到独立 MinIO，然后签名下载，得到完整 **2,206,099 字节**，SHA-256：

```text
538fb3999c7426fd32e49aec4329cf88cd8fb1a36c81484bbc8297bedce61e4b
```

去掉签名返回 403；测试精确清理自身 key。该结果证明真实存储和签名下载，视频来源是既有夹具，**没有证明本轮供应商生成、成片可播放或整条交付链路**。

所需额外变量见 backend/README.md：测试 MinIO 的 endpoint/access key/secret/bucket，以及既有媒体路径。未配置会明确 skip。

### 真实浏览器 API

命令：

```text
pnpm exec playwright test -c playwright.integration.config.ts tests/integration/live-creation.spec.ts tests/integration/live-upload.spec.ts
```

使用真实登录、API、隔离数据库；没有响应 Mock，关闭携带认证信息的 trace。三模式预检覆盖 desktop/mobile，精确匹配最终 mode/model/时长/分辨率/画幅/音频/素材参数的响应，并比较测试前后余额和任务列表。请求安全拦截仅阻止未授权写请求，不制造成功响应。

素材形状修复后，预检与上传合计 **3 passed / 1 skipped，55 秒**；skip 是 mobile 项目不重复执行相同的无界面 API 上传。事务连接修复后于 22:45:45 重启隔离新版 API，再跑最终 desktop/mobile 预检，结果 **2 passed / 48.1 秒**，没有重复上传。已检查三模式共 12 张首屏/估价区截图：桌面左输入右预览；移动端没有横向溢出，提交按钮未被底栏遮挡。

保留的最终估价页面证据：[桌面](./closed-beta-2026-09-28-desktop.png)、[移动](./closed-beta-2026-09-28-mobile.png)。画面对应真实免费估价，未显示模拟成片。

真实上传 ID 脱敏为 `upl_…3a1e`，鉴权回读返回 200，**116,782 字节**，SHA-256 为 `7890371b5b85f2d92c86728680f2f472e95fc9f2c55643c31851e2dd94e18b5d`；匿名访问 401。上传写入隔离 UploadDir，不是云端供应商素材读取证明。

直接读取隔离数据库核对：余额 ¥10.00、任务 0、消费订单 0、task_quotes 0、钱包流水 0、素材上传 1。免费估价和上传未触发扣费。

### UI 合同和视觉 Mock

`pnpm lint:check`、`pnpm typecheck`：通过。完整 `pnpm test:visual`：**205 passed，3 skipped**；跳过项仅适用 desktop。管理端核对恢复专用合同：桌面/移动 12 passed，新弹窗快照已检查；没有为掩盖漂移更新既有快照。

这些结果只证明页面与交互合同，不作为数据库、供应商、钱包真实扣退或交付证据。

### 验收状态总表

| 层级 | 状态 | 证据边界 |
| --- | --- | --- |
| 静态检查 / Go 编译与测试 | 通过 | lint、typecheck、完整 Go 测试、diff --check |
| 真实 PostgreSQL / Redis | 通过 | 隔离数据库事务、账务、归属、迁移；Redis 实际连接和测试 namespace，不代表生产容量测试 |
| 真实 MinIO | 通过 | 测试视频完整读写、签名权限；没有把已有夹具称为新生成视频 |
| 真实 API 桌面 / 移动免费估价 | 通过 | 三模式参数与素材匹配、余额/任务不变 |
| 真实图片上传与鉴权回读 | 通过 | 字节哈希一致，匿名 401；上传目录在隔离路径 |
| 管理端恢复交互 / 视觉 | 通过 | Mock 合同及视觉，仅作为界面证据；对应后端恢复另有真数据库测试 |
| 当前未解决的自动化失败 | 无 | 早期 seed、能力、取消与连接池问题修复后，最终完整运行通过 |
| 三模式真实供应商生成 | 未验证 / 阻塞 | 公网素材条件未满足，且未获本次预算批准 |
| 本轮供应商成片播放、下载、保存作品后刷新 | 未验证 | 不拿历史 UAT 或已有视频夹具充当完整验收 |
| 部署升级与回滚 | 未执行 | 本轮未授权改变既有 UAT |

## 真实供应商验收与费用门槛

本轮尚未创建任何真实供应商任务，没有真实任务 ID、消费订单 ID、供应商扣费或供应商成片下载可报告。测试账号的完整付费链路仍为未验证：提交 → Worker → 供应商 → 对象存储 → 任务详情 → 下载播放 → 保存作品 → 刷新持久化。

待批准计划：同一测试账号、`car_minimax / mdl_h3`，三条各提交一次；不开启自动重新提交。当前真实估价每条 **¥0.15**，合计 **¥0.45**；拟申请客户端测试余额预算上限 **¥0.60**。这些是平台对客消费金额，供应商采购额度仍会实际消耗，执行时应同时核对其预算。提交会重新估价，若任一参数或费用超出批准范围则停止。

| 模式 | 内部限定工作流（仅测试管理员可见） | 拟用参数 | 素材 |
| --- | --- | --- | --- |
| 文生视频 | minimax_h3_lightx2v_no_pic | 5 秒、768P、16:9、不自动音频 | 无图，海边公路平稳推进提示词 |
| 首尾帧 | minimax_h3_lightx2v | 5 秒、768P、16:9、不自动音频 | 已有许可海岸参考图，分别作为首尾两个槽位 |
| 多图 | 本次最低成本结果为 minimax_h3_zm_u08 | 5 秒、768P、16:9、不自动音频 | 同许可参考图作为一张参考图 |

客户端不能选择或看到上述内部编码。正式执行前需列出实时费用与预算上限，并取得用户确认。不得用循环重试消耗额外供应商额度。

## 待核对任务恢复

1. 管理员打开任务详情，查看提交事件、已知受理 ID、上游状态、时间、成本和售价快照。客户端只看到安全的等待/失败描述。
2. 在锁定供应商处核实该请求是否受理及归属，保存可追溯依据。单次 4xx、超时、断网或一次查不到任务不等于未受理。
3. 已受理：使用“核对上游任务”，提交 `action=resume`、核实过的 ID、核对依据和 `confirmed=true`。服务器再次查询上游；已捕获 ID 不能替换；成功后仅接续轮询。
4. 确认未受理且没有已知受理 ID：使用 `action=refund`，填写核对依据并确认。任务、退款、订单和审计一并提交；重复操作不重复退款。
5. 仍不确定或上游无法查询：保留待核对状态，联系供应商；不得直接改数据库或再次提交同一任务。

## 剩余阻塞与门槛

- 尚缺云端供应商可访问的素材域名/存储配置，已向用户询问；仅配置内网可读写 MinIO 不够。
- 三模式真实生成需用户批准具体费用；当前没有该批准，绝不自动提交。
- 真实成片可播放性、真实供应商成功率/耗时、真实生成后的账务对账、浏览器成片保存与刷新仍未验证。
- 新版尚未在既有 UAT 数据库执行升级/回滚演练；本轮没有授权部署或重启现有 UAT。
- 新 seed 保护只能阻止未来启动覆盖价格；如果历史 UAT 曾被旧 seed 覆盖，需要从原审计/备份核对，不能靠本轮代码还原。上述 ¥0.15 是隔离配置的真实返回，不等于已验证的商业价格或上游最终账单。
- API 执行适配器仍未验证，持续过滤；供应商错误码没有权威“明确未受理”分类，远端提交异常保守进入待核对。

因此本轮输出是有真实基础设施证据的候选工作区，不是已验收上线版本。下一步是补齐供应商可访问的素材配置，确认三条任务费用后执行一次受控真实交付验收。

## 本轮修改范围

以下只归纳本轮涉及部分；工作区中其它已有页面、迁移、文档和未跟踪文件仍属于此前协作修改，未重置或清理。

- 后端运行：`backend/cmd/server/main.go` 隐去种子密码日志，澄清环境 fallback 与数据库凭据的区别。
- 合同与事务：`backend/internal/handler/client.go`，`store/creation_catalog.go`、`direct_routing.go`、`tasks.go`、`user.go`、`settings.go`、`supply.go`、`uploads.go`、`projects.go`。
- 执行恢复：`provider/autodl/client.go`、`worker/provider.go`、`store/provider_tasks.go`、`store/admin_ops.go`；新增 Store/Handler 的 `provider_reconciliation.go`，由 `server/router.go` 注册受保护路由。
- Go 测试：能力/账务/迁移集成、`direct_beta_integration_test.go`、`direct_transaction_integration_test.go`、`source_shape_test.go`、恢复单测/集成、Worker 故障集成、MinIO 真存储集成、客户端安全错误合同。
- 前端：`frontend/src/api/admin.ts`、`pages/admin/TasksPage.vue`、新增 `features/admin/TaskReconcileDialog.vue` 与桌面/移动合同及快照；真实预检 `tests/integration/live-creation.spec.ts`、新增 `live-upload.spec.ts` 和 integration config。
- 文档：`backend/README.md`、`docs/api/client-tasks.md`、本验收记录及两张真实页面证据截图。

本轮没有新增业务数据库迁移。已运行的迁移目录包含工作区此前已有的 `020_execution_cost_rules.sql`、`021_provider_execution_reliability.sql`；本轮没有把它们应用到既有 UAT 数据库。

## 数据影响与清理

验收结束后已停止本轮本地 API、Vite 和 SSH 隧道，删除带本轮专用标签的三个测试容器及匿名数据卷；隔离测试账号、上传图片、测试任务/订单与遗留 schema 随测试数据库一并销毁。远端临时配置/脚本、本地根目录和 backend 下本轮 `.tmp_beta_20260928`、原始测试日志和重复截图已清理，不保留供应商凭据副本。

这些是一次性测试数据，已删除且没有备份，后续真实验收需重新初始化隔离环境。保留的持久证据为本脱敏验收记录、两张真实浏览器估价截图和可重复执行的源码测试。既有业务数据库、素材、用户文件与协作者改动没有清理或覆盖；原 UAT 服务清理后仍为 `MainPID=3222986 / ActiveState=active`。

未创建分支、未提交、未推送、未发布，也没有重启既有 UAT。最终 `git diff --check` 通过。
