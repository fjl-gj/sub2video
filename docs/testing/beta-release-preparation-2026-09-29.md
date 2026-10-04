# 封闭 Beta 候选版本发布准备（2026-09-29）

## 结论与权限边界

本记录对应 `beta-candidate-20260929-01`，分支 `codex/commercial-studio-redesign`，HEAD `a479c330d90b468fa17f2b0b3ce79a3f415b0dae` **加现有未提交修改**。不能只凭 HEAD 认定本地和 107 是同一版本。

本轮准备了 Linux 后端、包含官网/客户端/管理端的完整前端、校验清单、迁移入口和发布脚本故障测试，并于 2026-09-29 在维护窗口内完成 107 的备份、迁移、后端重启、前端切换和无扣费验收。没有提交或推送；没有创建付费任务、修改 UAT 用户余额/资产/供给/价格。集成测试使用专用测试库。

**候选已完成受控升级并通过本轮无扣费验收；仍不能据此宣布已经适合 5～10 位创作者独立试用。** 真实新任务、专用账号上传保存和实体 Safari/弱网仍未验收。当前候选没有新增本轮 UI 改动或视觉基线，包含此前尚未发布的体验修复。

## 发布差异

| 组件 | 本地候选 | 107 当前证据（本轮只读） | 待发布内容 / 依赖 | 兼容风险与验证 |
| --- | --- | --- | --- | --- |
| 后端合同 | `handler/client.go` 以 `durationSec` 接收估价/提交，旧 quote 路由返回 410 | 候选 PID 3435073；运行 SHA 与候选一致 | 已发布当前 handler/store/Worker 全量二进制 | 发布后浏览器 5 秒估价返回200；不能重引 quote 或改前端迎合旧包 |
| 前端（官网、客户端、管理端同一 dist） | `index-C66Dxnp2.js`，126 个文件 | HTTP 首页 SHA 与候选一致 | 已发布创作准备/估价/幂等恢复，任务详情、钱包与帮助、管理编辑/分页、320px 余额修复 | 不是三个独立部署；旧 hashed chunks 已保留，浏览器验收完成 |
| 任务状态与退款 | `canCancel`、`reviewRequired`、退款金额/时间、用户化失败原因 | 候选已提供字段与用户化失败状态 | `store/tasks.go`、`handler/client.go`；真实退款取账本 | 已用已有失败任务完成发布后对账 |
| 保存与下载 | 持久 `savedDestination`、归属校验后的 attachment 新签名 | 候选已提供字段与附件签名 | task/store、delivery handler、blob/MinIO/COS 签名及客户端下载 | 已验证已有成片真实下载和播放；保存写入仍待专用账号 |
| 迁移 | 001–021 + migration checksum 校验 | 仅 001–019；`schema_migrations` 没有 checksum 列，任务没有租约字段 | **必须补 checksum 元数据、020、021** | 不因上一轮没有新增 SQL 就跳迁移；历史 checksum 首次补录不证明历史 SQL 一致 |
| Worker | 提交/轮询租约、待核对、超时/交付恢复、事件记录 | 老 Worker 同 API 进程；当前 queued/running 为 0 | 021 后才能启动；保留 `SUB2VIDEO_SKIP_BOOTSTRAP=true` | 新进程启动即启动 Worker，并非 HTTP 健康后才启动；切换前必须重新确认无在途任务 |
| 启动/发布脚本 | 新增 `--migrate-only`；产物 SHA 必填、可验证回滚 | 原进程 `SUB2VIDEO_SKIP_BOOTSTRAP=true`，原二进制也包含此选项及跳过日志 | 单独迁移，正常服务继续跳过 seed/bootstrap | 直接换包不会补迁移；取消 skip 又会运行 seed/管理员提升/Worker。禁止以完整服务启动代替迁移 |

107 本轮只读环境还确认：Linux x86_64，部署分区约 71GB 可用；PostgreSQL 16、Redis 7.4；`pg_dump`、`tar`、`sha256sum`、`flock`、`timeout` 可用。数据库 `sub2video` 由 `sub2video-postgres` 容器承载，用户 `sub2video`。Nginx 生效站点为 `/etc/nginx/conf.d/sub2video-uat.conf`，监听 12000，API 反代到 127.0.0.1:12080。

任务现状：已完成 2、失败 1，queued/running 为 0，均为历史任务；这不是当前候选真实供应商生成证据。020 的执行配置/分辨率/音频重复成本组为 0，无未关联执行配置成本；发布当天必须重查，不能沿用此时快照。

## 本轮必要修复与修改文件

| 文件 | 本轮变更 |
| --- | --- |
| `backend/cmd/server/main.go` | 增加显式 `--migrate-only`，只连接 PostgreSQL 并执行现有迁移器；之后退出，先于上传目录、Redis、seed、管理员提升、Worker、对象存储和 HTTP |
| `backend/cmd/server/main_integration_test.go` | 新增真实命令子进程测试：首次/重复迁移、无 seed/上传目录、不可用 Redis/HTTP 不影响、非法参数在连接前拒绝 |
| `backend/migrations/upgrade_integration_test.go` | 新增无 checksum 的 019 数据库升级到 021 的演练；保留人工钱包、订单、流水、在途任务和管理员价格，二次执行一致，checksum 篡改拒绝 |
| `backend/README.md` | 补迁移命令、SkipBootstrap 与旧版本回滚边界；保留既有文档修改 |
| `deploy/update-107-api.sh` | SHA 校验、文件锁、同目录原子替换、备份/候选留存；不兼容回滚默认停机，兼容恢复也检查结果，失败再次停机 |
| `deploy/update-107-frontend.sh` | SHA 校验、锁、实际入口 hash 核对；发布保留旧 chunks，回滚保留新 chunks，不删除失败产物 |
| `deploy/tests/update-107.test.sh`、`deploy/tests/mock-command.sh` | 11 个纯本地文件切换/故障路径测试，服务和网络全部 mock |
| 本记录 | 候选身份、分层证据、迁移与回滚、发布后验收 |

既有 020、021 SQL 未在本轮修改；但属于本次候选相对 107 的必需发布范围。大量此前修改保留，不将相对 HEAD 的整体 diff 冒充本轮新增。

## 候选产物与源码身份

本地候选目录：`E:\apps\github.com\sbu2video\.tmp_release_20260929_01\candidate`。

| 产物 / 证据 | SHA-256 |
| --- | --- |
| 候选 Linux `sub2video-server`（23,051,843 字节） | `88b9a3cc479e9d99d55ec32e38ce526f2bbc97f0bfc23159659920478a744a34` |
| 完整前端 `sub2video-frontend-dist.tgz`（3,375,987 字节） | `5d1afdd5a19087c113d6e98be4f557a841ee66df01895913e325ccb358e27a05` |
| 候选 `index.html` | `79379ed562514024752d345ac7c7e112ba6e2a3c6f8c014cc31e8f9d6e15f99a` |
| 候选 `update-107-api.sh` | `a52fe65f0331805ddff85f2fca538c075b6dd3b249b8dff128adf3b51f92fdc4` |
| 候选 `update-107-frontend.sh` | `0293916c10cd8b6abe1afb5cc654e2519975b6b063f2fd779b2c1baba4a1368a` |
| 353 个构建/测试/部署输入文件的清单摘要 | `4e07c0dd65093c759360a67719851e4b7247498fc7f3ca37e9917617ad5fbfd0` |
| 107 旧后端（运行与磁盘相同） | `6155515fe3615a6daedf79631de0bf5f1c216c53fd3214934c892ad893f0d0f2` |
| 107 当前 `index.html` | `c1ea7441bbb3340bcf6737932ea23d46002f57f0c1f36a4498502c1ea0851f2b` |
| 020 SQL | `6c65cb4fc2d5cbd9ca0c980e11b7475b1ba09e70f293640d9ab39a9f2fcc2e65` |
| 021 SQL | `2167aca49c2c3856c9a6a8d406ada7b42d94b855d1212772c321ee53d1ce8683` |

`manifest.json` 包含分支/HEAD、未提交文件状态、相对 HEAD 的 numstat、输入文件路径/大小/hash、完整前端文件清单、工具与构建命令。`source-files.sha256` 是排序后逐文件摘要，摘要覆盖实际文件字节，不只 Git 状态；它不包含凭据、backend/data、server.linux 或旧临时目录。该清单不包含这份发布说明等非构建文档，不能用其代替整个工作区的档案备份。

`SHA256SUMS` 同时校验二进制、前端包、脚本、manifest 和源码清单。前端归档仅从 `frontend/dist` 生成；重新解包并逐个核对 126 个文件均相同。没有把仓库、`.env`、业务数据、日志或旧临时目录打包。Go 二进制内嵌迁移，无须另行上传 SQL 来执行升级。

工具：Node v22.16.0，pnpm 10.32.0，Vite 6.4.3，Go 1.25.1 windows/amd64，交叉目标 linux/amd64，CGO_ENABLED=0，`-trimpath`。已检查 ELF 为 x86-64。构建保留 `vcs.modified=true`；未以虚构版本标签伪装干净提交。

构建命令（工作目录分别为 frontend/backend；PowerShell 使用进程环境设置 GOOS/GOARCH/CGO 后恢复）：

```text
pnpm build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o ../.tmp_release_20260929_01/candidate/sub2video-server ./cmd/server
```

## 验证记录与证据边界

| 层次 | 本轮命令 / 结果 | 能证明与不能证明 |
| --- | --- | --- |
| 静态 | `pnpm lint:check`、`pnpm typecheck` 通过 | 当前 Vue/TS 与规则；不证明 API/供应商 |
| 构建 | `pnpm build` 通过（6.93 秒）；Linux Go 交叉构建通过 | 当前候选可构建，产物已校验；未在 107 启动 |
| 前端 Mock 交互 + 视觉 | `pnpm test:visual --reporter=list --output=../.tmp_release_20260929_01/frontend/visual`：265 通过、15 skip、0 失败（1.9 分钟） | 创作、恢复、下载 fixture、管理编辑/分页和响应式回归；不证明真实钱包/供应商/交付；本轮未更新截图 |
| 默认 Go | `go test -count=1 ./...` 通过，退出0 | 集成开关关闭时跳过的环境用例不算真实集成通过 |
| 真实 PG 升级 | `TestUpgradeFrom019PreservesExistingDataIntegration` 通过（11.62 秒） | 人工旧 schema/数据升级、重复执行与 checksum 拒绝；不是 UAT 数据副本恢复演练 |
| 真实 PG 命令 | `TestMigrateOnlyCLIHasNoServiceSideEffectsIntegration` 通过（20.66 秒） | 真子进程在 SkipBootstrap=true 下执行迁移，未调用服务初始化 |
| 真实 PG/Redis 业务 | Store 12项、HTTP handler 2项、Worker 1项顶层测试通过；加上述迁移/CLI共17项顶层、24子场景，0skip/0fail | 专用测试库随机 schema；Store 325.841秒、Handler 37.973秒、Worker 37.266秒；供应商/对象 HTTP 为明确 fixture，不是实际视频生成 |
| 发布脚本 | Bash `-n` + `bash deploy/tests/update-107.test.sh`：11/11 通过 | 文件切换、错误 SHA、回滚判断与 chunks 留存；systemctl/nginx/curl/flock/sleep 为 mock，不证明真实服务操作 |
| 包验证 | `sha256sum -c SHA256SUMS` 全通过；前端 126 个解包文件 hash 全通过 | 待上传产物与审阅产物一致 |
| 工作区 | `git -c core.autocrlf=false diff --check` 通过 | 没有空白错误；不意味着干净工作区 |
| 真实 UAT | 版本、进程、schema、在途任务数、基础设施只读核对 | 本轮未部署、未重新进行浏览器业务验收，历史成功记录没有计入候选通过 |

15 个 Playwright skip 是项目去重：12 个自定义宽度仅 desktop 项目运行（实际覆盖 320/390px），3 个手机专属用例仅 mobile 项目运行。不是缺基础设施却标通过。

真实数据库测试由进程内安全注入 `SUB2VIDEO_RUN_INTEGRATION=1`、`SUB2VIDEO_TEST_DATABASE_URL`（专用 `sub2video_test` 的随机schema）、`SUB2VIDEO_TEST_REDIS_URL`（DB15）。连接仅通过本轮SSH隧道本地25432/26379，未复制UAT数据。准确选择如下，在 backend 目录运行：

```powershell
go test ./migrations ./cmd/server -run 'Test(UpgradeFrom019PreservesExistingData|MigrateOnlyCLIHasNoServiceSideEffects)Integration$' -count=1 -v -timeout=5m
$selection = '^(' + (@(
  'TestDirectCatalogMigrationAndSeedIsolationIntegration',
  'TestDirectLowestCostSupportedExecutionAndCapabilityFilteringIntegration',
  'TestDirectSubmissionBillingRefundAndSnapshotIntegration',
  'TestDirectSubmissionRollbackAndConcurrentIdempotencyIntegration',
  'TestSaveCompletedTaskToProjectIsIdempotentIntegration',
  'TestDirectSubmissionLeaseCrashRecoveryAndFencingIntegration',
  'TestDirectPollLeaseAndStorageRecoveryIntegration',
  'TestProviderReconciliationRefundAtomicAndIdempotentIntegration',
  'TestCancelTaskIsIdempotentAfterResponseRetry',
  'TestClientTaskActionsMatchCancellationIntegration',
  'TestClientTaskSavedDestinationSurvivesReloadIntegration',
  'TestImageSlotsRejectAudioUploadsBeforeChargeIntegration',
  'TestEstimateDurationSecHTTPContractHasNoFinancialSideEffectsIntegration',
  'TestDeliveryURLHandlerIntegration',
  'TestWorkerSubmissionUncertaintyIntegration'
) -join '|') + ')$'
go test -p 1 ./internal/store ./internal/handler ./internal/worker -run $selection -count=1 -v -timeout=20m
```

其中与客户端相同的 `durationSec:5` 请求返回200、估价¥0.15，钱包不变，任务/订单/quote/流水为0，供应商请求数0。提交受理后本地持久化失败、HTTP408不确定、租约过期旧Worker写入、存储恢复/等待超时均有回归。对象下载使用本地HTTP fixture核对归属与签名，不宣称真实MinIO交付。

真实供应商新生成、专用账号上传/保存、本次候选删除操作和实体 Safari/弱网验收 **未执行**。同版本已有成片的下载、浏览器播放、退款展示和三模式无扣费估价已完成；新付费任务仍须另列参数和预算。

## 实际发布与发布后证据

2026-09-29 已执行以下操作：

- 远端 release 目录为 `/opt/sub2video/releases/beta-candidate-20260929-01/`，权限为 0700；候选上传后重新校验 SHA。
- 数据库备份 `backup/database.dump` 的 `pg_restore --list` 得到 323 个条目；旧后端、旧前端归档、Nginx 配置和备份 SHA 清单均保留。
- 使用候选 `--migrate-only` 完成 checksum、020、021；迁移后验证 21 个版本、21 个 checksum、10 个租约字段、事件表、成本索引和 `offering_id` 可空，业务数据快照未改变。
- 迁移脚本通过 PowerShell 管道传输时在成功输出后多出一个末尾 CR，远端 shell 返回非零；迁移日志明确记录成功，独立 schema/数据复核和随后候选启动均通过，未把这个传输包装问题误报为迁移失败。
- 后端运行 SHA 为 `88b9a3cc479e9d99d55ec32e38ce526f2bbc97f0bfc23159659920478a744a34`；systemd active，health 返回 `status=ok`，日志确认 SkipBootstrap 和对象存储初始化正常。
- 前端磁盘和 HTTP 首页 SHA 均为 `79379ed562514024752d345ac7c7e112ba6e2a3c6f8c014cc31e8f9d6e15f99a`；Nginx 配置测试通过。
- 发布后任务/订单/账本/quote 数量为 `3/3/4/0`，状态仍为 2 个 completed、1 个 failed；没有因估价产生新任务或扣费。
- 浏览器三种模式无扣费估价均为 5 秒、768P、16:9、无音频、¥0.15；估价响应仅包含 `available`、`billing` 和客户端可见 `selection`，没有供应商、成本、workflowCode、apiModelCode、executionType、Token 或上游地址。
- 已有成功任务下载产生真实 `wen-sheng-verify.mp4`（1,657,822 字节）；文件头为 MP4 `ftyp`，浏览器视频 `readyState=4`、864×480、时长约 1.625 秒。
- 已有失败任务详情显示费用已退回 `¥0.36` 及退款时间，原始上游错误未显示。
- 六档视口 320、390、768、1024、1440、1920px 的创作页 `scrollWidth == clientWidth`，未发现横向溢出。

当前浏览器会话不是隔离写入账号，因此没有点击“保存到作品”，避免修改真实用户资产。

## 发布步骤记录与回滚边界

以下是下一轮授权后的操作单，不是已经发生的动作。任何输入源文件变化，都应重新构建/测试受影响部分并更新清单；不要继续沿用本记录的旧 hash。

1. **固定产物并核对目标。** 将候选 6 个文件和 SHA256SUMS 上传到独立 `/opt/sub2video/releases/beta-candidate-20260929-01/`，不要覆盖 `/tmp` 中来源不明的文件。远端 `sha256sum -c SHA256SUMS`；记录旧后端/index hash、现有 unit/drop-in 与 Nginx 配置的路径和 hash，不在报告输出服务环境/密码/token。正常服务继续使用原环境、原工作目录与原 `SUB2VIDEO_SKIP_BOOTSTRAP=true`，不覆盖 service 文件。
2. **建立维护窗口与写入冻结。** 在已确认的 `/etc/nginx/conf.d/sub2video-uat.conf` 中临时限制该站点及 `/api/` 仅发布操作者/127.0.0.1 可访问，先明确操作者 IP，再审阅最小配置 diff、`nginx -t` 后 reload；不得替换其他站点。从未允许地址验证拒绝。若无法建立可靠访问门禁，则暂停升级。授权范围包含此临时配置及恢复原配置；本轮没有修改。旧页面与手机也应被门禁拦住，不能仅靠口头通知无人提交。
3. **再次只读盘点。** 重新查询迁移、重复成本组、成本关联、queued/running、上游已受理/待核对任务；当前零在途只是一时快照。若已有提交中/未确定任务，暂停替换，先通过现有管理核对流程处理或明确接续方案；禁止直接改库解除。可允许已知受理任务自然结束，但不能自动退款/重新提交。
4. **停止旧服务前保留环境到内存。** 在无 shell tracing 的 root Bash 会话中读取当前进程环境到数组，只用于随后 migrate-only 子进程。不得 `echo` 环境、生成含凭据的 `.env` 或把原始 journal 登录响应存入项目。随后停止 `sub2video-backend`，确认进程退出、没有其他 Worker/连接在写；再次核对在途任务。Nginx 留在维护状态。
5. **备份并验证可恢复性。** 在远端 root-only（umask 077）的该 release `backup/` 下保留旧后端、完整旧 dist、仅目标 Nginx 站点配置和 PostgreSQL custom dump。`pg_dump` 使用容器内已核实的本地认证，不在命令拼接密码。记录 SHA，并 `pg_restore --list` 检查；严格恢复演练应恢复到新建隔离数据库（绝不能覆盖 sub2video/sub2video_test），核对迁移/任务/钱包/账本/订单行数与汇总后再继续。备份包含业务/加密配置，仅留受限远端，不下载进仓库。若备份或恢复演练失败，停在此步。
6. **先独立迁移。** 运行已校验候选 `--migrate-only`，外层 `timeout 300s`；仅连接 PG。完成后必须核对 21 个迁移与64位 checksum、021 的10个任务字段和 provider_task_events、020 unique index 与 offering_id 可空，且既有任务/钱包/订单/流水/售价/成本汇总未改变。任一异常停止，不启动 Worker。不要只用退出码代替结构/数据核验。
7. **启动候选后端。** 使用该候选内的 `update-107-api.sh <后端SHA>`，显式指定候选路径。默认不启用自动旧版回滚。只有确认没有新版在途/订单语义、旧二进制支持 skip 且当前020/021 schema允许它运行后，才可设置 `SUB2VIDEO_ROLLBACK_COMPATIBLE=true`。脚本不做迁移，也不改变服务环境。检查实际运行 hash、schema、只读业务接口与脱敏错误；health 仅代表存活。
8. **发布完整前端。** 使用 `update-107-frontend.sh <前端包SHA>`，显式指定 archive。脚本保留两代 hash chunks 并检查实际 index hash。官网、客户端、管理端共同更新。保留旧页面标签，再打开新标签，分别测试懒加载/登录回跳；旧页提示或要求刷新，但不得清除待确认提交记录。
9. **门禁内完成无扣费验收。** 执行下方清单；出现 P0/P1 保持关闭，按回滚条件处理。脚本通过不表示发布成功。单独批准的受控写入完成后才能面向5～10位用户解除维护限制。记录恢复的 Nginx 配置 hash、实际页面入口与 API hash。

示例发布命令只供下一轮使用，当前没有执行：

```bash
# root Bash；不要 set -x。先确认 PID 非零且是目标进程。
release_dir=/opt/sub2video/releases/beta-candidate-20260929-01
api_pid=$(systemctl show -p MainPID --value sub2video-backend)
test "$api_pid" -gt 0
mapfile -d '' -t release_service_env < "/proc/$api_pid/environ"
# 在维护、在途核对、停止与备份步骤完成后：
timeout 300s env -i "${release_service_env[@]}" "$release_dir/sub2video-server" --migrate-only
unset release_service_env
# 核对实际 schema 和账务数据后才启动：
SUB2VIDEO_API_CANDIDATE="$release_dir/sub2video-server" \
  bash "$release_dir/update-107-api.sh" 88b9a3cc479e9d99d55ec32e38ce526f2bbc97f0bfc23159659920478a744a34
SUB2VIDEO_FRONTEND_ARCHIVE="$release_dir/sub2video-frontend-dist.tgz" \
  bash "$release_dir/update-107-frontend.sh" 5d1afdd5a19087c113d6e98be4f557a841ee66df01895913e325ccb358e27a05
```

远端数据库备份示例，必须在停止写入后执行且目录已以0700创建；本轮未产生 UAT dump：

```bash
umask 077
docker exec sub2video-postgres pg_dump -U sub2video -d sub2video -Fc > "$release_dir/backup/database.dump"
test -s "$release_dir/backup/database.dump"
docker exec -i sub2video-postgres pg_restore --list < "$release_dir/backup/database.dump" > /dev/null
sha256sum "$release_dir/backup/database.dump"
```

**禁止使用 `deploy/finish-107-backend.sh` 完成本次发布。** 它有重置演示余额、输出登录响应的历史行为，不属于这份发布操作单。也不调用会重启服务的存储配置脚本。

## 备份、回滚和 Worker 边界

| 失败位置 | 安全处理 | 不允许的处理 |
| --- | --- | --- |
| SHA/备份/迁移前检查失败 | 保持原产物；未开始迁移时可恢复原访问，说明发布未发生 | 忽略 hash、跳备份、直接运行种子初始化 |
| 迁移失败 | 服务保持停止，检查已完成版本；迁移按每份 SQL 事务执行，不是020+021整体事务；必要时继续修复迁移或在明确无新写入前提下走独立恢复决策 | 假设全部回滚、删checksum记录重跑历史001–019 |
| 候选服务启动失败 | 默认脚本停止候选并保留旧binary路径；检查 schema/在途兼容后才恢复旧binary，验证运行hash及业务读取 | 盲目启动旧Worker、只用health宣称恢复 |
| 前端切换/入口验失败 | 脚本恢复旧index与资源，保留新chunks和失败目录；验证原index hash；保留受控门禁 | 删除新版chunks让已打开页面崩溃；把双rename说成零停机 |
| 业务验收失败但未允许新写入 | 保持门禁；兼容成立时回滚后端与前端产物，通常保留020/021增量schema；核对账本/任务不变 | 默认回退数据库或降低合约校验 |
| 已产生新订单/提交/交付后失败 | 立即冻结新增写入，保存事件与受理ID；优先修复新版本或只回退安全前端。管理端核对 `SUBMIT_UNKNOWN`：证实受理后接续，证实未受理且无受理ID后退款，均带审计 | 恢复旧dump覆盖新订单、未知受理重提/退款、旧Worker接管未知任务 |

旧二进制会在正常启动运行 Worker；`SkipBootstrap=true` 只禁迁移和 seed，**不禁 Worker**。服务停止/崩溃不代表供应商撤销；新版租约过期后的不确定提交应进入核对，而非二次提交。当前在途为0，仍需维护窗口内再次检查，不能用进程停止替代账务核对。

020/021主要是放宽旧关联、新索引、新字段/事件表，没有本轮自动 down migration。二进制回滚与数据库恢复是两件事。即便回到旧产物，原107存在的时长合同问题也会回归；“恢复原基线”不等于“可以开放创作”。保持关闭直到有通过业务验收的同版本组合。

本轮已完成数据库 dump、真实 systemd 切换和前端 Nginx reload；尚未把 dump 恢复到隔离数据库做完整恢复演练。原 unit/drop-in 未改，没有把其内联秘密复制到本地发布包。

## 发布后的体验验收单

以下清单记录同版本发布后的验收；优先使用已有账号会话、已有成功/失败任务与素材。新生成和专用账号写入仍需单独授权。

| 场景 | 操作与通过条件 | 本轮证据边界 |
| --- | --- | --- |
| 登录与入口 | 官网→教程→创作→登录回跳；原参数/教程上下文保留，管理端权限正确 | 当前 Mock 已覆盖；新版本真实会话待验证 |
| 三模式准备 | 文生/首尾帧/多图的素材数量、类型、时长、分辨率/画幅提前明确；指定模型不换模 | 真实PG能力/合约与Mock分层验证；真实组合页面待发布 |
| 估价 | 合法5秒等参数免费估价；任务/订单/账本不增加；缺素材、无供给、存储异常不混淆 | 错误供给/存储不得人为破坏UAT配置；不能自然复现时保留隔离测试证据并记UAT未验 |
| 余额 | 同一账户顶栏、创作、钱包一致，估价后金额不变 | 取同一时点/刷新结果，不比较不同账户或过期标签 |
| 任务与退款 | 读已有成功/失败/待核对任务；金额/时间与真实账本一致，canCancel不误导，内部错误不外泄 | 不为造状态新增付费/失败任务；缺状态可标未验 |
| 下载 | 点击后有实际下载文件，核对大小、MIME、可播放/媒体时长；刷新后可重新签名下载 | 不能用预览新标签或200替代下载；有过期签名时不能退回旧URL |
| 保存位置 | 已保存任务详情能定位到真实作品目录，刷新仍存在，回收站/删除不指向失效路径 | 只读已有记录；新保存属于下方受控写入 |
| 帮助和再创作 | 帮助保留taskId；按原参数再创作只回填，不能自动提交/扣费 | 不清除原待确认请求，不能诱导再次购买 |
| 信息边界 | 检查页面及业务响应不含供应商、采购成本、内部执行类型、凭据、上游编码/地址 | 正常登录令牌、用户消费金额、自有存储签名不是同类泄露；勿关键词误判 |
| 六档宽度 | 320、390、768、1024、1440、1920px；创作表单/预览、顶栏余额、任务/钱包、管理列表/弹窗，无横向溢出、遮挡、按钮不可达 | 本轮自动化宽度回归通过；新版本真实API浏览器/实体Safari尚未验 |

受控写入应另记账号和范围：使用专用 `beta-experience` 测试身份（名称仅建议，**未创建，实际 userId 待授权明确**），独立作品目录 `beta-acceptance-20260929`，仅上传明确提供的测试图片并保存一条该账号已有成功任务到作品，刷新定位，再次创作只回填不提交。账号无既有成功任务时，不能复用他人资产/改归属；将保存验证保留为阻塞，或使用同版本隔离环境夹具。

受控范围：上传→鉴权读取→保存→刷新定位→再次创作；记录创建的 upload/project/task关联，仅清理本次身份及目录中的测试产物，不清理其他资产。当前不授权执行这些写入。若确需新的付费视频，单独列模式/模型/素材/时长/分辨率/音频、账号、估价和预算，明确确认后一次性提交；不能无限重试。

## 剩余阻塞与开放门槛

| 项目 | 是否阻止受控发布 | 是否阻止邀请5～10位创作者 |
| --- | --- | --- |
| 107旧合同/缺迁移/未上新交付字段 | 已完成迁移、后端和前端同步发布 | 已不再阻止封闭试用，仍需专用账号写入验收 |
| 同版本估价、真实附件下载/播放、退款详情 | 已通过发布后验收 | 不代表真实供应商新生成 |
| 专用账号受控上传/保存/刷新未验 | 不阻止只读升级，需单独限定写入授权 | **是，关键路径尚无同版本真实UI证据** |
| 历史001–019 checksum来源不全 | 需要备份/隔离恢复、结构核对后升级；本轮人工升级测试不能替代现场 | 通过升级和账务核对前不能开放 |
| 真实支付、本轮不新增的供应商/影片工程 | 不属于封闭Beta范围；钱包应明确充值未开放 | 不把这些当已交付，也不能作为公众收费上线结论 |
| 实体Safari/弱网设备、真实新付费任务 | 不阻止受控107升级，明确记未验 | 试用前完成代表性设备和批准范围的必要检查；不将历史任务冒充当前生成 |

本轮没有证据表明发生了UAT重复扣费、越权或数据丢失（P0），也没有进行完整新版本运行审计来证明所有P0均不存在。可审阅候选与可开放私测分开判定。

## 数据、临时文件与下一步

本轮没有修改 `backend/data/`、`backend/server.linux`，没有清理任何旧临时目录。

集成首跑出现过一次测试隔离错误：pgx修改RuntimeParams后取ConnString没有携带新search_path，CLI在原先确认空的 **sub2video_test.public** 创建了33张空表，而非随机schema；未使用UAT库、未seed用户、未启动Worker。已改为在子进程URL显式设置search_path，并按数据库名、精确表清单、迁移数量、零用户/连接守卫事务清理了本轮空表。首跑失败日志保留，没有删除失败证据或把首跑记为通过。最终专用库随机schema/用户表/其他连接、Redis15键、本地SSH隧道监听均为0；107仍为原PID及启动时间。

仅上述33张本轮合成空表和测试随机schema被删除；不包含任何既有用户数据，可通过测试重新生成。所有此前临时目录均保留。

候选及证据保留在唯一目录：`E:\apps\github.com\sbu2video\.tmp_release_20260929_01`

其中 `candidate/` 为已发布产物副本，`backend/` 为Go与隔离测试日志，`frontend/` 为前端日志，`deploy/mock-run.722op2/` 为最终11项脚本演练，`archive-review/` 为解包核对副本，`prepare-candidate.ps1` 为本轮临时清单生成器。候选和回滚证据确认前不要清理此根目录；确认留存所需产物后，此目录是本轮唯一可整体清理的临时范围。`frontend/dist` 为现有标准构建输出（Git忽略），未提交；不把它或临时日志加入Git。

下一步是：**为专用隔离账号授权一次上传→保存→刷新定位→再次创作回填验收，并补做隔离数据库恢复演练。** 通过后再决定是否邀请 5～10 位创作者封闭试用。不包含付费生成、用户余额修改、供给/价格编辑、真实用户资产写入或其他站点变更。
