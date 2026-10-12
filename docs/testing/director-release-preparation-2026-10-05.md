# 导演台候选版本发布准备

日期：2026-10-05（Asia/Shanghai）。候选 ID：`director-beta-candidate-20261005-01`。

## 结果与范围

已构建当前工作区的 Linux 后端和完整前端，准备迁移核验 SQL、源码/产物清单、备份及回滚操作顺序。当前分支为 `codex/commercial-studio-redesign`，HEAD `0cafa8d612f4ad8dcebcff58d7d25d674ebd5945`，包含未提交的导演台修改。

本轮没有改生产源码，没有提交、推送、部署、迁移 UAT、重启服务或付费生成；107 仅执行读取。本候选具备进入受控发布流程的构建和本地回归条件，但部署当时的备份恢复检查、写入冻结及发布后验收仍是必需门槛。不能直接据此邀请创作者验收完整成片闭环。

使用发布流程设计技能补足了版本配套、迁移门槛和分阶段恢复条件；没有引入新 CI/CD 平台或第二套部署工具。

## 当前发布差异

| 组件 | 107 实际基线 | 当前候选 / 待发布内容 | 依赖与验证 |
| --- | --- | --- | --- |
| 后端 | 运行及磁盘 SHA 均为 `88b9a3cc…744a34`，对应 9 月 29 日候选 | 导演接口、镜头板权限和 revision、任务创建同事务关联；此前执行配置原子更新、旧 quote 隔离等修复一并进入完整二进制 | 先迁移 022；完整默认 Go 测试已重跑，真实 DB 证据见前轮记录 |
| 前端 | HTTP/磁盘 index SHA 均为 `79379ed5…15f99a` | 官网/客户端/管理端完整 dist；导演台、免费“保存到镜头”、会话与草稿隔离、余额刷新和此前体验修复 | 新导演页面必须使用配套后端；317 项 Mock/视觉回归通过 |
| 数据库 | 完整 001–021，21 个 checksum 与候选 SQL 原始字节逐项一致；无导演两列 | 仅新增 `022_project_director_board.sql` | 为 projects 增加 JSONB 镜头板和 BIGINT revision；不改财务记录 |
| 启动及 Worker | `SUB2VIDEO_SKIP_BOOTSTRAP=true`；Worker 与 API 同进程 | 保持原配置，独立 `--migrate-only` 后再启服务 | skip 只禁迁移/seed，不禁 Worker；不能用正常启动代替单独迁移 |
| 发布脚本 | 继续使用现有两个 update 脚本 | SHA 校验、备份、文件锁、保留两代 hashed assets；默认不自动回滚旧后端 | 12 项本地故障测试通过；不调用 `finish-107-backend.sh` |

与已部署产物所对应的旧 manifest 比较：43 个已记录输入文件内容变化，30 个当前输入未被旧清单记录。后者不全等于新增文件：旧生成器曾排除 `frontend/src/data`，本轮恢复完整源码清单。逐文件差异写入候选 manifest；不能只用 Git HEAD 或只发布导演页 chunk。

107 只读事实：API PID `4066`、active、启动于 `2026-10-02 23:47:43 CST`；unit `/etc/systemd/system/sub2video-backend.service`，drop-in `/etc/systemd/system/sub2video-backend.service.d/90-skip-bootstrap.conf`，工作目录 `/opt/sub2video/backend`。

Nginx 站点 `/etc/nginx/conf.d/sub2video-uat.conf`，监听 12000，根目录 `/opt/sub2video/frontend/dist`，配置 SHA `c6cd9dc6d3b7061f969d2181020b63d5666dabfc69ecec8055734bb409a1a81d`。PostgreSQL/Redis 容器运行中；名为 `sub2video-minio` 的容器不存在，不能由此断言对象存储不可用，因为系统支持独立 COS/其他存储配置，本轮不读取凭据或重做连通性测试。部署分区约 71.7 GiB 可用。

## 候选身份与内容

保留目录：`E:\apps\github.com\sbu2video\.tmp_release_20261005_01\candidate`。它是本轮待审阅发布交付物，不是可提前清理的测试垃圾；所有产物均被 Git 忽略。

| 内容 | SHA-256 |
| --- | --- |
| `sub2video-server` | `b8ad0f489bc5b2007898ac6e479ea58cd4088430bb38d3a4e9e8f79b29a8f5d6` |
| `sub2video-frontend-dist.tgz` | `bbde58f6be4eaf2e5e60dd3023a8be80436d7ce1c9c82fef74de0791bc3910ba` |
| 新前端 `index.html` | `88cdf672df2f85dd5243f3093a471418bf4d7ea1bb6f88ef0de5060d9bd8d4fe` |
| 380 个实际源码/测试/部署输入清单 | `a1c062e48e6554b886ad4ad84030232396bcf836725bd002a49c13c6d738583b` |
| 022 SQL | `7a6254b511e36bc3a15c9350ffab23f763ccdece614e1799a6430857e7878026` |

包中还包含 `manifest.json`、`SHA256SUMS`、`source-files.sha256`、两个既有 update 脚本、`preflight.sql`、`verify-022.sql`、`data-fingerprint.sql` 和本文副本 `release-runbook.md`。完整前端共 133 个文件，归档根直接是 index.html；解包后逐文件比较摘要。SQL 核验文件只有只读查询，真正迁移来自 Go 二进制内嵌的 SQL。

源码清单包括未跟踪文件，按区分大小写路径排序，文件字节 SHA-256 两个空格路径、LF 行尾，无 BOM。包含 backend cmd/internal/migrations、Go依赖、frontend src/public/tests及配置、deploy；不包含文档、环境文件、业务数据、旧临时目录或源码压缩包。后端内置默认开发值属于既有源码，候选未读取或注入任何实际运行凭据；运行时必须继续使用 107 原有安全配置。

本轮固定使用 `StringComparer.Ordinal` 排序，避免操作系统文化排序差异。前轮数据库记录的110文件摘要 `bc708df7…5997` 来自 `Sort-Object -CaseSensitive`（文化排序），本轮用原算法已完整复现；同一批文件改用ordinal排序为 `b3443818…b70c`。这次差异是排序规则而非已观察到的源码漂移，前轮真实数据库证据的后端输入仍一致。

构建工具：Go 1.25.1、Node v22.16.0、pnpm 10.32.0、Vite 6.4.3。后端验证为 ELF64 x86-64，`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 -trimpath`，VCS 元数据明确 `vcs.modified=true`。

只要任一输入发生变化，必须重新构建/验证受影响部分并重做清单，不能继续沿用旧 SHA。最终 `SHA256SUMS` 校验包内所有载荷和清单；它本身不是签名，不替代本记录中的已知产物 SHA。

## 验证结果与边界

| 分类 | 命令与结果 | 边界 |
| --- | --- | --- |
| 静态 | `pnpm lint:check` PASS；`pnpm build` 内 `pnpm typecheck` PASS | 本轮无 UI 源码变化 |
| 前端构建 | `pnpm build --outDir ../.tmp_release_20261005_01/work/frontend-dist` PASS，2472 modules | 首次因 esbuild `spawn EPERM` 失败，按权限提升后重跑通过；不是源码修复 |
| Go 回归 | `SUB2VIDEO_RUN_INTEGRATION=0 go test -count=1 ./...` PASS | 真实基础设施用例有意跳过，不算集成通过 |
| 后端构建 | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o ../.tmp_release_20261005_01/candidate/sub2video-server ./cmd/server` PASS | 未在 107 启动该二进制 |
| 前端 Mock/视觉 | `pnpm test:visual --reporter=dot --output=../.tmp_release_20261005_01/work/visual`：317 PASS / 15 SKIP / 0 FAIL，约 2.8 分钟 | 测试 proxy 指向不可用 loopback；15 项为按视口去重。本轮无快照更新 |
| 发布脚本故障测试 | `bash deploy/tests/update-107.test.sh <本轮目录>/deploy`：12 PASS | systemctl/nginx/curl 等全部 Mock；包含旧危险脚本拒绝执行 |
| 107 只读 | 版本/迁移/租约/成本重复组、preflight 和 data-fingerprint SQL 成功 | 首次探针错误使用 `error_code`，实际列为 `upstream_status`，修正后通过；所有事务 READ ONLY，未写数据 |
| 包与补丁 | 全部源码/产物摘要一致、133 文件解包逐项匹配、`git diff --check` PASS | 不等于正式业务验收 |
| 前轮真实数据库 | [32 项 PostgreSQL/Redis 验收](director-database-acceptance-2026-10-05.md) | 前轮证据，不计作本轮重跑；本轮没有建立新测试库 |
| 前轮真实浏览器 | [4 项真实 API 流程](director-browser-acceptance-2026-10-05.md) | 已覆盖免费准备、上传、保存恢复、冲突；前轮来源服务并非本轮生产 dist |
| 供应商交付 / UAT 导演页 / 备份恢复 | 未执行 | 不用历史成片、Mock或/health替代 |

## 现场数据门槛

本次只读观测：1 用户、1 作品、1 钱包，余额合计 255.96；3 任务（2 completed、1 failed）、3 订单、4 账本条目、0 quote、0 upload；12 成本规则、8 售价规则。所有在途、待核对、WAITING_STORAGE、有效提交租约和重复成本组均为 0。

这些只是一时快照，发布前必须在阻止新提交后重新检查。`SUBMIT_UNKNOWN` 的真实表示包括 `status=running + upstream_status=SUBMIT_UNKNOWN + submit_uncertain=true`，不能只找名为 submit_unknown 的 status。

`data-fingerprint.sql` 已在 107 只读运行，比较用户、钱包、流水、订单、任务、作品（排除新增导演列）、成本及售价的行级聚合摘要，不输出记录明细。MD5 仅用于前后数据相等性检测，产物校验仍用 SHA-256。服务停止、无其他写入者时采集迁移前后结果；运行中的当前摘要不能替代维护窗口基准。

## 正式发布顺序（本轮未执行）

1. **固定包和窗口。** 下轮明确授权后，创建 `/opt/sub2video/releases/director-beta-candidate-20261005-01/`，`umask 077`、目录0700。上传本候选，不覆盖旧 release；重新核对已知后端/前端 SHA 及 `sha256sum -c SHA256SUMS`。核对旧运行 SHA、当前配置和实际路径没有变化；有变化先重做差异审阅。
2. **阻止新写入，排空任务。** 仅修改 Sub2Video 对应站点的受控维护配置，禁止新建任务，确保已打开标签也被服务器阻断；等待在途、核对任务及存储等待清空。然后冻结全部业务写入。配置检查通过后才 reload，不影响其他站点。不能排空、不能建立门禁或有未知提交时停止发布。
3. **持有环境并停服务。** 在 root 的受限发布进程内读取当前 PID 环境，禁止 shell tracing、打印、写 `.env` 或把秘密放进外部命令参数。保持原工作目录和 `SKIP_BOOTSTRAP=true`，只在子进程内存环境中传给迁移命令。停止 API、确认旧 PID 退出，再查无在途及其他业务写入者。当前 Worker 没有等待执行完毕的 shutdown barrier，不能依靠 stop 代替排空。
4. **备份并验证。** 先保存旧后端、完整旧 dist、Sub2Video Nginx配置、systemd unit/drop-in和上传目录的位置/持久卷信息；含敏感配置的备份只留远端root目录，不下载到仓库。PostgreSQL 使用容器现有本地认证执行 custom dump，不在脚本或参数中放密码。记录全部备份 SHA，`pg_restore --list`检查；再恢复到明确隔离的临时 PostgreSQL数据库，使用 `--no-owner --no-privileges`并比较迁移/业务摘要，不能覆盖 UAT。恢复失败即停止。对象存储文件不因本次迁移搬移或删除。
5. **仅执行迁移。** 已校验候选以原环境、原工作目录运行 `timeout 300s .../sub2video-server --migrate-only`。它只连接PG并迁移，不启动Redis、HTTP、Seed、Worker或存储。核对完整001–022和所有checksum、两列类型/NOT NULL/default、两个已validated CHECK；首次022后旧作品全为空板/revision0。重新执行数据摘要，必须与维护窗口基准一致。失败保持停机，不清checksum或重跑历史SQL。
6. **先启后端。** 使用候选 `update-107-api.sh`，保持默认 `SUB2VIDEO_ROLLBACK_COMPATIBLE=false`。核对新运行SHA、active、/health及鉴权业务接口；Worker随API启动，需观察任务事件。不能以health通过替代业务验收。
7. **发布完整前端。** 使用候选 `update-107-frontend.sh`，保留两代hashed assets。实际HTTP index必须等于新SHA，官网/客户端/管理端和懒加载均能打开。脚本双rename有短暂间隔，在维护窗口内执行。
8. **受控验收后恢复访问。** 先执行下节无扣费清单；有专用账号且写入范围获得授权才做新作品/镜头/上传/保存。任何P0/P1保持门禁，按下节恢复；通过后恢复原Nginx配置并核对SHA。不自行创建付费生成。

发布命令仅在完成上述前置步骤之后执行：

```bash
release_dir=/opt/sub2video/releases/director-beta-candidate-20261005-01
SUB2VIDEO_API_CANDIDATE="$release_dir/sub2video-server" \
  bash "$release_dir/update-107-api.sh" b8ad0f489bc5b2007898ac6e479ea58cd4088430bb38d3a4e9e8f79b29a8f5d6
SUB2VIDEO_FRONTEND_ARCHIVE="$release_dir/sub2video-frontend-dist.tgz" \
  bash "$release_dir/update-107-frontend.sh" bbde58f6be4eaf2e5e60dd3023a8be80436d7ce1c9c82fef74de0791bc3910ba
```

## 迁移与回滚边界

022仅加列，旧作品/财务数据不应改变；迁移器按每个SQL独立事务，并非所有迁移一个事务。它会执行所有缺失版本，所以必须先确认001–021完整一致。`ADD COLUMN IF NOT EXISTS`不能保证预先存在的同名列约束正确，必须核实列和CHECK定义。

| 失败位置 | 处理方式 |
| --- | --- |
| SHA / 备份 / 门禁失败 | 不替换产物，不迁移；保持或恢复原已核实基线 |
| 迁移失败 | 保持API停止；记录实际已执行版本，查明原因。禁止删checksum、自动重跑或用旧dump覆盖业务数据 |
| 候选API启动失败 | 脚本默认停止候选并保留旧二进制；旧API恢复必须另核实兼容性，在此之前不开放任务提交 |
| 前端发布或入口校验失败 | 脚本恢复旧dist并保留新chunks；优先保留兼容的新后端和022 schema。核实实际旧indexSHA及业务读取 |
| 必须回退后端 | **持续阻断任务创建**，配套恢复旧前端。旧API可能忽略directorContext而扣费生成普通任务，已打开的新标签也有此风险；不能只让操作者刷新就恢复全部写入。修复/验证兼容合同之前保持创建任务门禁 |
| 已有新订单/任务产生后失败 | 冻结新增写入、保留provider受理ID及事件；优先前向修复，不用旧dump覆盖订单/账本。未知提交不得自动重提或退款 |

正常二进制/前端回退保留022列和已有镜头数据，不做down migration。恢复旧dump是独立的数据恢复决策，只有确认没有新写入且明确授权后才能考虑。备份归档可读不等于恢复演练已通过；本轮二者均未执行。

## 发布后验收清单

无扣费读取优先：官网→登录回跳→三模式目录和素材要求→合法免费估价→顶栏/创作/钱包余额一致→已有任务状态与退款→已有成片下载产生真实文件并可播放→已保存位置及帮助上下文→管理端配置读取。比较估价前后任务、订单、账本、quote及余额；不得人为修改UAT供给/价格来制造失败。

导演台专用受控写入：确认实际测试账号和独立作品名称后，新建作品与镜头→修改/排序/保存→刷新→前往创作并显式应用→上传两张测试图片→免费保存到镜头→返回/刷新恢复参数及图片→两页面409冲突且保留编辑。无余额/存储错误用隔离环境证据，不动真实配置；再次创作不能自动提交。候选采用/下载只用该账号已有真实成功任务，没有则标未验证。

六档宽度320/390/768/1024/1440/1920检查横向溢出、主要按钮、弹窗及侧栏/预览比例；旧标签与新标签均检查懒加载。页面和业务响应不得泄露供应商/采购成本/执行类型/上游编码/凭据，正常认证令牌和用户消费金额分开判断。

## 修改文件、数据与清理

本轮可纳入Git的新增文件只有本文。生产源码、022和现有发布脚本未改。未修改 `backend/data/`、`backend/server.linux`，未清理任何旧临时目录。

候选包和本轮生成器保留于 `.tmp_release_20261005_01` 供下一轮审阅发布，不加入Git。已检查以下两个目录的精确绝对路径及所有子项均无reparse point，随后用 `Remove-Item -LiteralPath` 删除，并分别复核 `Test-Path=False`：

- `E:\apps\github.com\sbu2video\.tmp_release_20261005_01\work`
- `E:\apps\github.com\sbu2video\.tmp_release_20261005_01\deploy`

移除约7.95 MB可重建的解包副本、构建文件、只读探针和Mock日志。候选11个文件及根目录生成器仍保留，未删除旧候选或业务数据。没有在远端创建临时目录、容器、数据库或备份，没有更改真实用户数据或余额。审阅时可用 `./.tmp_release_20261005_01/prepare-candidate.ps1 -Phase verify`重新验证当前源码和候选；正式重新构建应另建独立工作目录。

## 发布条件与下一步

没有发现本轮新增的确定生产P0/P1；混合版本回滚是明确P1风险，已作为禁止开放任务写入的门槛写入方案。当前包可以作为受控107发布候选，不能把“包已准备好”说成“可无条件发布”或“适合直接邀请5–10位用户”。

下一步的部署授权范围应为：仅上传本候选；备份并验证恢复；临时维护门禁；执行022；同步更新后端和完整前端；无扣费验收；专用账号范围明确后的导演台写入。付费生成、Git提交/推送、真实用户资产/余额、供给和价格变更均不包含在内。
