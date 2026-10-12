# 导演台真实数据库验收记录

日期：2026-10-05。分支 `codex/commercial-studio-redesign`，HEAD `0cafa8d`。测试对象包含当前工作区尚未提交的导演台实现，不能只用 HEAD 标识本次测试版本。

本记录接续 [导演台首版本地实施与验收](director-workspace-2026-10-05.md)。上一轮因缺少数据库配置及 SSH 连接关闭而跳过，本轮 SSH 恢复后使用独立 PostgreSQL/Redis 实际执行；旧报告保留为当时证据。

## 结论与边界

已在真实 PostgreSQL 16.15 / Redis 7.4.11 上通过 30 个 store 集成测试、1 个导演接口测试及 1 个升级迁移测试，共 32 个独立顶层测试，没有 SKIP。导演台的两个专用 store 测试先单独通过，再纳入这 30 个回归测试，不另重复计数。

已验证镜头持久化、权限、版本冲突、候选状态、任务与镜头原子关联、并发幂等单扣和财务对账。迁移验证包含空库首次/重复执行、合成 019 数据升级到 022、再次执行和 checksum 被改动时拒绝继续。

这是当前源码的数据库与接口验收，不是当前版本真实供应商交付或 107 页面验收。没有运行供应商 Worker、没有付费生成，也没有从 UAT 复制用户、余额、凭据或业务数据。

## 隔离环境

| 项目 | 本次实际使用 | 隔离边界 |
| --- | --- | --- |
| 主机 | `192.168.0.107` | 只复用 Docker 主机和本机已有镜像，不部署应用 |
| Docker 网络 | `s2v-director-db-20261005-01` | `internal=true`，无公网出站 |
| PostgreSQL | `s2v-director-pg-20261005-01`，16.15 | 无端口映射，专用合成测试库，数据目录 tmpfs |
| Redis | `s2v-director-redis-20261005-01`，7.4.11 | 无端口映射，无持久化，临时数据目录 |
| 每个测试 | 随机 schema / 自有 Redis probe key | 不将 public 加入 search_path，不执行 FLUSHDB |
| 测试进程 | Linux 静态 Go test binary，`docker run --rm` | 只读根目录/挂载，限 CPU/内存，仅 `/tmp` 可写 |

两容器和网络均带 `com.sub2video.test-run=director-db-20261005-01` 标签。PostgreSQL 的无密码 trust 仅限这张隔离网络，未使用或输出 UAT 数据库连接串。

镜像内容 ID：

- PostgreSQL：`sha256:cf78e76683b9ca8c5733cbbdce6c9262b45b6767934dd0a95e671f9a0fc20685`
- Redis：`sha256:ff02b58f971e7d7d156a1267e283fcbbeee91773b6aa36c49dac28ecfe28eadf`

UAT 原服务核对为 `MainPID=4066`、`active`、启动时间 `2026-10-02 23:47:43 CST`；原 PostgreSQL/Redis 容器 ID 与启动时间前后逐字比较相同。这证明本轮未重启这些进程，不等同于业务全面健康检查。

## 本轮变更

| 文件 | 变更原因 |
| --- | --- |
| `backend/internal/store/integration_test_helpers_test.go` | 去除 public 回退；校验实际 schema/数据库；清理失败必须使测试失败；清理使用限时 context；Redis 写入前即注册清理 |
| `backend/internal/store/director_integration_test.go` | 按整数分核对钱包、订单、任务和扣费流水；拒绝他人/回收项目上下文；补齐他人素材、已删素材和交付未就绪断言；验证注入的 P0001 确实来自镜头写入 |
| `backend/internal/handler/director_integration_test.go` | 补真实 Router、鉴权和数据库的导演接口合同验证 |
| 本文与上一轮验收记录 | 保存新证据并区分历史环境 skip |

本轮没有修改生产业务逻辑、前端样式、022 SQL 或部署脚本；保留上一轮实现及其他已有修改。

## 通过场景

| 范围 | 实际断言 | 证据边界 |
| --- | --- | --- |
| 导演数据 | 空板读取、保存后重读、顺序保留、同 revision 并发单成功、手工附加候选与整板保存竞争不丢候选 | PostgreSQL 并发事务 |
| 归属 | 跨用户读取/保存、他人任务、他人上传/作品素材、回收项目均拒绝；两个用户财务状态不变 | 合成双用户，非真实用户资产 |
| 素材失效 | 旧引用删除后仍能读取板；保存必须先移除/更换无效引用 | 读取可恢复不代表源文件仍存在 |
| 选片 | running 不可选；completed 无存储结果不可选；completed 且有交付引用可选 | 交付 URL 为 fixture，不证明视频可播放 |
| 提交事务 | 人工触发板写入失败，钱包/流水/订单/任务整体回滚；相同 key 并发只创建一次 | 本地数据库事务，不调用供应商 |
| 幂等与对账 | 原单/原任务重放；镜头删除和供给关闭后仍重放；每笔订单恰好一条匹配扣费流水，金额按分一致 | 合成账户，不改真实余额 |
| 共享创作路径 | 能力/供给过滤、指定模型不切换、支持适配器的最低成本排序、模式售价、估价后重新计算、不写 quote、保存作品幂等 | 30 个 store 测试的一部分 |
| 失败恢复 | 取消/失败退款幂等、租约 fencing、提交不确定核对、已受理 ID 保护、轮询/存储恢复 | 数据库状态与故障注入，不是进程 kill 或外部服务验收 |
| 迁移 | 合成 019→022 保留钱包/订单/账本/任务/成本/售价/作品；旧作品获得空镜头板；重复执行不变；checksum 篡改报错 | 只在临时库执行，未迁移 UAT |
| 导演 HTTP 合同 | 三个接口匿名 401 / 他人 404、保存重读、409 不覆盖、候选重试不递增 revision、未交付不可选、就绪选片持久化、未知内部字段拒绝、回收作品 404 | 真实 Gin Router/JWT/数据库，使用 httptest 内存请求，不是浏览器或公网监听 |
| 响应隐私 | 遍历 JSON 字段及已知私有 fixture 值；不包含采购成本、执行配置、凭据、供应商或私有地址；prompt 内普通 token 单词合法 | 导演接口专用断言，不用关键词误报认证和用户文案 |

## 命令与版本证据

构建工具：`go1.25.1 windows/amd64`。测试二进制目标：`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`。

```powershell
# cwd: backend；输出仅放本轮临时目录
$env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go test -c -o ../.tmp_director_db_20261005_01/store.test ./internal/store
go test -c -o ../.tmp_director_db_20261005_01/migrations.test ./migrations
go test -c -o ../.tmp_director_db_20261005_01/handler.test ./internal/handler
```

| 测试产物 | SHA-256 |
| --- | --- |
| `store.test`（最终补强断言后） | `d3c59c5691786cb82e695b5b220670e6f589a8cf410ad3492643fa2f2ce17716` |
| `migrations.test` | `202e8b25a4a7c64cdd3ef69d06d24c22361b487b3830fcfcfa6c1b05e9ed755e` |
| `handler.test` | `61dd23d8226621cd1543c00a25419bbe4c6631e91381791ba46e193a71b9a64a` |

最终后端源码摘要：`bc708df7797d879cbcfe6118806bc7da44d28e1abcd9740170e65a3ea54b5997`。范围为 `backend/cmd`、`backend/internal`、`backend/migrations` 的全部 `.go/.sql` 文件及 `backend/go.mod/go.sum`，共 110 个文件。每行以相对 backend 的 `/` 路径、一个空格、小写文件 SHA-256 组成，按区分大小写路径排序，LF 拼接（末尾 LF）、UTF-8 编码后再取 SHA-256；文件本身按原始字节计算，包含未跟踪源码，不包含业务数据或产物。

最初两项导演 store 及首次 30 项回归使用的 store binary 摘要为 `e9a4a0c6f407244fc3dbadc1d5a477bba2cb870643c4b17139e2352c81c2eb11`；强化错误来源与 Redis 清理后重新编译，并重新执行全部 30 项，以最终产物结果为准。

上传后 `sha256sum --check` 匹配。实际测试启用 `SUB2VIDEO_RUN_INTEGRATION=1`，两个 TEST URL 仅指向该隔离网络中的测试服务。以下命令为临时 runner 内的筛选与参数，正常仓库环境也可使用同样的 `go test -count=1 -v ... -run` 筛选重跑。

| 分类 | 命令/筛选 | 结果 |
| --- | --- | --- |
| 导演 store | `store.test -test.v -test.count=1 -test.timeout=600s -test.run='^TestDirector.*Integration$'` | 2 PASS / 0 FAIL / 0 SKIP |
| 升级迁移 | `migrations.test -test.v -test.count=1 -test.timeout=600s -test.run='^TestUpgradeFrom019PreservesExistingDataIntegration$'` | 1 PASS / 0 FAIL / 0 SKIP |
| 共享 store 回归 | `store.test -test.v -test.count=1 -test.timeout=600s -test.run='Integration$\|^TestDirectTaskSubmissionIsIdempotentAfterSupplyChanges$\|^TestCancelTaskIsIdempotentAfterResponseRetry$'` | 30 PASS / 0 FAIL / 0 SKIP |
| 导演接口 | `handler.test -test.v -test.count=1 -test.timeout=600s -test.run='^TestDirectorBoardHandlerIntegration$'` | 1 PASS / 0 FAIL / 0 SKIP |
| 本地 Go 全包 | backend：`SUB2VIDEO_RUN_INTEGRATION=0 go test -count=1 ./...`（PowerShell 通过 `$env:` 设置） | 通过；基础设施集成在该命令中有意 skip，真实 DB 证据见上列独立运行 |
| 补丁检查 | `git diff --check` | 通过；Git 仅提示既有 LF/CRLF 自动转换 |

本地首次受限环境交叉编译失败于 Go 标准库/缓存读取权限；提升命令权限后编译成功。没有修改源码规避编译错误。

store 的 Redis 使用证据限于连接及隔离 probe 读写清理；任务队列的真实 Redis 消费和运行中的 Worker 崩溃重启仍不在本次范围。并发请求测试验证本次交错结果，不宣称穷尽全部调度顺序。

## 尚未验证与下一阶段

- 本轮不修改前端，不重跑其无变化的 Mock/视觉测试；上一轮 `313 PASS / 15 viewport SKIP` 仅为历史前端证据。
- 未启动新版本真实前后端进行浏览器完整操作；尚未证明真实登录后“作品→导演台→上传/保存→刷新→创作回填”的页面闭环。
- 未用新版本生成或下载真实视频；数据库中 completed fixture 不构成交付证据。
- 未执行 UAT 022 迁移、同步发布或发布回滚演练。不能据此宣布导演台已上线或已达到封闭 Beta 全部门槛。

下一阶段：在同样隔离的测试账号/应用环境中做导演台真实 API 浏览器验收，先覆盖不付费的作品/镜头保存、并发冲突、素材选择、刷新恢复和创作回填；再单独准备 022 与前后端同步发布。

## 数据、提交、部署与清理

无 Git 提交/推送，无部署/服务重启，无付费任务。未修改 `backend/data/`、`backend/server.linux`，未清理以前的临时目录。

本轮只产生可重建的合成测试数据。store/迁移测试结束后测试 schema 数为 0、Redis keys 为 0；UAT 原进程指纹不变。

本轮资源已清理并复核不存在：

- 本地 `E:\apps\github.com\sbu2video\.tmp_director_db_20261005_01`
- 远端 `/opt/sub2video/.tmp_director_db_20261005_01`
- 上表两个测试容器和一张测试网络

删除对象只限本轮资源，日志/二进制/归档不加入 Git；正式证据保留在本文中。

远端清理前确认了精确路径、无符号链接、容器/网络的本轮标签、无映射端口和网络仅有两个成员；删除两容器及网络后重新核对不存在，UAT 原服务/原数据库容器指纹仍相同。随后仅删除远端本轮临时目录。

本地先确认精确路径位于仓库且所有内容均无 reparse point，再用 `Remove-Item -LiteralPath` 删除该目录；`Test-Path` 返回 `False`。移除的是约 55 MB 可重建的测试二进制、脚本与日志，未删除源码或其他临时目录。临时合成数据已随 schema 清理及 tmpfs 容器移除，不保留备份，可按测试重新生成。

本轮未发现确定的新增生产 P0/P1；这不是完整产品审计结论。当前阻止导演台宣称已完成封闭 Beta 验收的缺口是新版本真实前后端浏览器闭环，以及正式环境的 022/前后端同步发布验证。

## 后续浏览器补充

同日下一轮已完成隔离前后端真实浏览器验收，并修复创作页缺少独立“保存到镜头”的体验缺口。桌面/移动共 4 项真实 API 测试通过，覆盖图片上传、参数保存、刷新恢复和版本冲突；全量前端回归为 317 PASS / 15 viewport SKIP。详见 [浏览器验收记录](director-browser-acceptance-2026-10-05.md)。本记录的 32 项数据库测试保持为前轮证据，不宣称在浏览器轮重新执行；正式 UAT 发布及新版本成片交付仍未完成。
