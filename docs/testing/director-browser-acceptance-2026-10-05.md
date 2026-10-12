# 导演台真实浏览器验收与免费准备流程

日期：2026-10-05。分支 `codex/commercial-studio-redesign`，HEAD `0cafa8d612f4ad8dcebcff58d7d25d674ebd5945`。测试对象包含未提交的导演台实现；HEAD 不是完整测试版本标识。

接续 [真实数据库验收](director-database-acceptance-2026-10-05.md)。本轮完成当前前端、真实 Gin Router、隔离 PostgreSQL/Redis 的浏览器操作，不部署、不运行 Worker、不产生付费生成。

## 结论

桌面和移动视口共 4 项真实 API 浏览器测试通过，无跳过、无重试。全量前端 Mock/视觉回归为 317 PASS、15 viewport SKIP、0 FAIL。lint、typecheck 和 production build 均通过。

已经证明：真实登录 → 新建作品 → 新建并保存镜头 → 刷新读取 → 显式应用镜头 → 上传首尾帧 → 保存素材与参数 → 返回导演台 → 刷新 → 再进入创作并从服务器恢复图片与参数。两页面同时编辑时旧 revision 返回 409，旧页面本地编辑保留，只有用户确认重新加载才替换。

本轮没有证明：成功估价、提交生成、Worker 消费、供应商成功、对象存储交付、可播放视频下载、真实任务选片或正式 UAT 的导演台功能。不能以此宣布完整封闭 Beta 验收通过。

## 发现的问题与修复

| 问题 | 实际影响 | 本轮修复 | 验收 |
| --- | --- | --- | --- |
| 创作页缺少将准备好的素材和参数保存回镜头的独立操作 | 用户必须提交生成才把完整输入持久化到镜头，供给/估价不可用时准备工作无法独立完成 | 增加“保存到镜头”，复用现有 board PUT，不新增接口或表；上传本地图片后保存引用 | 真实 API 上传、PUT、刷新、恢复图片通过 |
| 免费保存不能被生成可用性绑住 | 余额或估价故障不应阻止整理镜头 | 保存状态独立于余额和估价，允许准备不完整的镜头 | 真环境验证 STORAGE_UNAVAILABLE 时可保存；余额为 0 的路径用 Mock 验证 |
| 保存结果不确定或版本冲突后重复整板提交可能覆盖外部更新 | 用户可能丢失其他镜头/候选或误判保存成功 | 保留应用时快照和 revision；409/404/未知 PUT 结果保留草稿，要求重新读取、明确应用再保存；保存时冻结相关操作 | 真实两页面 board 冲突；创作页重新应用且保留外部镜头用 Mock 验证 |
| 上传完成后撤掉本地 File，预览仍需有效来源 | 刷新前后图片显示必须一致 | 保留上传响应预览 URL；恢复时使用鉴权上传引用 | 真图片 `complete && naturalWidth > 0` |

只替换当前镜头的公开创作输入，保留其名称、候选和其他镜头。普通草稿与按用户/作品/镜头隔离的草稿继续分开；保存和应用均防御会话、路由及卸载后的迟到响应。不新增参考音频流程。

## 修改文件

- `frontend/src/api/director.ts`：公开镜头输入类型。
- `frontend/src/features/director/useDirectorCreation.ts`：独立保存、revision 快照、草稿保留及恢复状态。
- `frontend/src/features/director/DirectorCreationContext.vue`：保存按钮和真实加载/成功/失败提示。
- `frontend/src/pages/client/CreatePage.vue`：整理公开参数与素材引用、复用上传、保存中冻结操作。
- `frontend/tests/integration/live-director.spec.ts`：真实 API 页面闭环、真实冲突、财务无变化及六种宽度检查。
- `frontend/tests/e2e/director.contract.spec.ts`：无余额/估价故障下保存及冲突恢复的 Mock 合同。
- `frontend/tests/e2e/director.contract.spec.ts-snapshots/director-save-inputs-{desktop,mobile}-win32.png`：两张新增基线。
- 本文、[UI 合同](../ui/director-workspace-2026-10-05.md)、上一轮数据库验收记录：能力与证据更新。

本轮没有改生产后端、迁移或部署脚本；保留前几轮已有未提交修改。临时 Go harness 只用于隔离服务启动，不进入正式代码。

## 环境与隔离

| 项目 | 实际环境与边界 |
| --- | --- |
| 前端 | 当前源码 Vite，`http://127.0.0.1:4196`，API proxy 指向本机 12186 |
| 后端 | 临时 harness 引用实际 `server.NewRouter`、鉴权、store 和 migrations；只监听 `127.0.0.1:12186` |
| 数据库 | 107 上新建 PostgreSQL 16 容器 `s2v-director-browser-pg-20261005-01`，专用测试库/角色，tmpfs |
| Redis | 新建 Redis 7.4 容器 `s2v-director-browser-redis-20261005-01`，无持久化 |
| 网络 | `s2v-director-browser-20261005-01`，internal 网络，资源均带本轮标签 |
| 本地访问 | SSH 转发本机 25486/26386 到已核实的测试容器内网地址；未暴露公网 |
| 账号 | `director-browser@example.test`；另有未使用的合成 `director-mobile@example.test`。密码和本地鉴权密钥仅在运行时内存，不记录或复制 UAT 凭据 |
| 素材 | 仓库既有公共海岸图片，复制为首尾帧测试上传；文件仅放本轮本地上传目录 |
| 供应商/存储 | 不启动 Worker；假供给地址为不可用 loopback，仅供目录能力选择；不接对象存储，估价真实返回 503 `STORAGE_UNAVAILABLE` |

最初 Docker internal 网络的主机映射端口不可达，SSH 转发连接被拒绝；随后核实容器内网 IP 并改用该地址作为 SSH 目的端。未改 UAT 网络、服务或配置。此问题属于测试连接路径，未作为产品故障处理。

浏览器使用 Playwright Chromium；mobile 项目是窄屏加触控的 Chromium，不是真实手机或移动 Safari。内置浏览器只检查了本地公开注册页，未在其中注册、登录或写数据，不能把这次检查称作导演台内置浏览器全流程验收。

测试禁止 POST tasks 和范围外写入，网络拦截仅允许/中止真实请求，没有 `route.fulfill` 或伪造响应。只有已创建测试作品允许 PUT。trace/video/自动失败截图关闭，防止认证内容进入产物；主动截图仅限登录后的合成作品页面。

## 验证命令与证据边界

以下命令均在 `frontend` 执行。真实测试的启用标记和隔离账号凭据由临时 runner 在内存传入；不得把它们加入 Git 或命令日志。

```powershell
pnpm lint:check
pnpm typecheck
pnpm build --outDir ../.tmp_director_browser_20261005_01/frontend-dist

# 已启动隔离前后端，显式开启 LIVE / LIVE_DIRECTOR / LIVE_ISOLATED_ACCOUNT
node node_modules/@playwright/test/cli.js test -c playwright.integration.config.ts tests/integration/live-director.spec.ts --reporter=line --retries=0 --output=../.tmp_director_browser_20261005_01/live-run

# Mock 套件不连接任何可用后端
$env:VITE_API_PROXY='http://127.0.0.1:1'
$env:PLAYWRIGHT_REUSE_EXISTING_SERVER='0'
pnpm test:visual tests/e2e/director.contract.spec.ts --grep 'saves shot preparation without balance|save conflict' --reporter=line --output=../.tmp_director_browser_20261005_01/mock-focused --update-snapshots
pnpm test:visual --reporter=dot --output=../.tmp_director_browser_20261005_01/visual-all

# 仓库根目录
git diff --check
```

| 分类 | 本轮结果 | 能证明什么 |
| --- | --- | --- |
| lint / typecheck | PASS | 静态合同，不证明运行中的数据正确性 |
| production build | PASS；Vite 6.4.3，2472 modules | 当前源码可构建；浏览器真 API 验收用的是 Vite 源码服务，不是部署这个 dist |
| 真实 API 浏览器 | 4 PASS，0 SKIP，0 retry，约 1.2 分钟 | 两个场景 × 桌面/移动；真实登录、上传、持久化、刷新和冲突 |
| 新增 Mock 合同与基线 | 4 PASS | 0 余额仍能保存；创作页冲突后显式重新应用，保留其他镜头及候选 |
| 全量 Mock/视觉 | 317 PASS / 15 SKIP / 0 FAIL，共 332 项，约 2.3 分钟 | 当前前端回归；不替代真实 API 或财务证据 |
| 15 项 SKIP | 视口专用用例只在对应项目执行 | auth minimum viewport、journey minimum width、studio mobile、responsive 自定义宽度矩阵的重复项目跳过；不是服务不可用导致 |
| Go / 独立数据库套件 | 本轮未重跑 | 生产后端未变化；前轮 32 项真实数据库 PASS 见独立记录，不并入本轮测试数 |
| 付费生成 / 交付 / UAT 导演页 | 未执行 | 本轮范围排除，不标记 PASS |

UI/UX 技能仅用于检查保存动作的加载/成功/错误反馈和恢复入口；Vue 与 Playwright 技能用于行为及测试组织，保留现有 Vue 3 和设计系统。

## 视觉与响应式复核

真实页面均检查 `320 / 390 / 768 / 1024 / 1440 / 1920 px`：页面无横向溢出，主要动作在视口内、触达高度至少 40px，手机底部导航不遮挡动作。对导演台和创作页分别留过截图。

人工审图包括真实 create 320/768/1440、director 390/1024/1920，以及两张新增 desktop/mobile Mock 保存成功基线。小屏按钮自然换行，宽屏保持既有对齐和表单/预览分区；预览显示实际上传的参考图，没有伪装成已生成视频。

新增快照的预期变化只有上下文区的“保存到镜头”及说明/成功反馈，没有改主题、圆角或容差，也没有覆盖旧基线掩盖漂移。几何检查与抽样截图不等于所有真实设备验收。

## 数据与财务证据

测试完成后，直接查询隔离数据库：

| 指标 | 值 |
| --- | --- |
| 合成用户 | 2 |
| 合成钱包余额合计 | 512.00，来自测试初始化，不是真实资金 |
| 作品 | 6：2 个初始化空作品 + 4 个浏览器新作品 |
| 上传 | 4：桌面和移动各 2 张 |
| generation_tasks | 0 |
| consumption_orders | 0 |
| task_quotes | 0 |
| wallet_ledger | 0 |
| 64 位 checksum 迁移记录 | 22 |

每个浏览器用例还读取真实钱包、任务列表和流水，断言前后完全相同。

桌面素材作品 `prj_…e1eecadc` 和移动素材作品 `prj_…9218c0a3` 均保存为 revision 2、1 个 `mode_first_last` 镜头；各含两个真实 `upl_…` 引用，服务器恢复后图片成功解码。两个冲突作品最终均为 revision 2，第二个页面的旧保存被拒绝。没有任务或订单 ID，因为没有创建任务或订单。

## 可复核版本摘要

以下均为文件原始字节 SHA-256，包含本轮未提交源码：

| 文件 | SHA-256 |
| --- | --- |
| `frontend/src/api/director.ts` | `63c52aff3803528e4c26e335db7ed6ddf99091d9dcf5ecb652b082389d1d074c` |
| `frontend/src/features/director/useDirectorCreation.ts` | `308869c2c6391c659f1b9caabb81f1861f8974489b4802e862433ae0962eb8e5` |
| `frontend/src/features/director/DirectorCreationContext.vue` | `f20d08e06db697d51f25993f505a72dbb0e4bccf79514da287923acbb8ef9067` |
| `frontend/src/pages/client/CreatePage.vue` | `18974bbd76e321a2178ad46e0d8d7cc8c82a1f1371e21b6d5db402d56ef07469` |
| `frontend/tests/e2e/director.contract.spec.ts` | `fa1c241843b46940d3141d0afd5440a256bd36933ac83e239fb48fb0ee590ef9` |
| `frontend/tests/integration/live-director.spec.ts` | `1eae450d5b31be775ac2bb2de817ac9684af14134ce9067a4b4d9f2eb0d50cc8` |
| 本轮 build `index.html` | `88cdf672df2f85dd5243f3093a471418bf4d7ea1bb6f88ef0de5060d9bd8d4fe` |
| 临时真实 Router harness EXE | `7afeeae808ae6c9bf4dc99592cddf0bff95bbf5be69dd1af61d890b377d025d1` |

后两者仅标识本轮测试产物，不是可直接发布的候选包；临时产物按工程规则清理。

## 数据影响、清理与剩余工作

无提交、推送、部署、UAT 迁移或重启；未修改 `backend/data/` 和 `backend/server.linux`，未清理以前的临时目录。UAT `MainPID=4066`、`active`、启动时间 `2026-10-02 23:47:43 CST`；原 PostgreSQL/Redis 容器 ID 和启动时间在测试前、测试后、清理后逐字比对相同。这只证明这些进程未重启，不是 UAT 业务全面健康证明。

本轮创建的临时资源：

- 本地 `E:\apps\github.com\sbu2video\.tmp_director_browser_20261005_01`
- 本地 `E:\apps\github.com\sbu2video\backend\.tmp_director_browser_20261005_01`
- 远端 `/opt/sub2video/.tmp_director_browser_20261005_01`
- 上文两个测试容器、一张 internal 网络和仅限本机的 SSH 转发。

本地测试前后端、SSH 转发和本轮内置浏览器标签已关闭，4196/12186/25486/26386 无监听。远端校验了精确路径、无符号链接、本轮标签及恰好两个网络成员，随后删除两测试容器、网络和临时目录，并验证不存在。合成账户、余额、作品、上传记录随 tmpfs 测试数据库移除，可按测试重建，不保留业务备份。

本地先确认两个精确路径位于仓库、根及子项均无 reparse point，再以 `Remove-Item -LiteralPath` 删除；两个 `Test-Path` 均返回 `False`。删除约 29.37 MB 的本轮脚本、临时 harness、构建、测试上传和运行截图/日志，不保留这些可重建临时产物；正式测试源码、两张经过审图的视觉基线和本文证据保留。未删除业务数据或以前的临时目录。

本轮未发现确定的新增 P0/P1。上传尚未完成就离开/刷新时，本地 File 可能需要重新选择；此时不会显示保存成功或扣费。后续仍需准备 022 + 前后端同版本候选、备份与回滚，再获授权同步发布并执行 UAT 无扣费验收。已交付真实任务的候选采用/下载和新版本供应商交付依然不能用 Mock 或这次准备流程替代。
