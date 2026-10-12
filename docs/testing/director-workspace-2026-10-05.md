# 导演台首版本地实施与验收记录

日期：2026-10-05。分支：`codex/commercial-studio-redesign`。开发前基线为 `0cafa8d`，开始时工作区干净；因此未创建空提交。本轮新功能尚未提交、推送或部署。

后续进展：真实 PostgreSQL/Redis 验证已在下一轮执行，见 [导演台真实数据库验收记录](director-database-acceptance-2026-10-05.md)。下文的数据库 SKIP 和 SSH 阻塞保留为首轮历史事实，不再代表最新验证状态。

## 结果与范围

实现作品内的镜头导演台：镜头编排、保存、候选任务关联、选片、清单导出，以及通过现有创作页生成镜头的上下文。采用 Toonflow 的工作场景，在现有 Vue / Go 栈实现，不嵌入上游引擎或复制上游代码。没有新增依赖、账户体系、供应商或付费入口。

入口：资产 → 进入作品 → 导演台；路由 `/app/projects/:projectId/director`。

本地实现完成不代表真实数据库或 107 已验收。本轮没有付费生成、没有真实任务/订单 ID，也没有当前版本真实视频交付证据。

## 改动清单

| 范围 | 文件 | 目的 |
| --- | --- | --- |
| 数据 | `backend/migrations/022_project_director_board.sql` | 复用 projects，添加镜头 JSON 和 revision |
| 后端 | `backend/internal/store/director.go`、`backend/internal/handler/director.go`、`backend/internal/server/router.go` | 读取、并发保存、归属校验、关联候选 |
| 创建任务 | `backend/internal/store/direct_routing.go`、`backend/internal/handler/client.go` | 幂等查询后校验导演上下文，原事务内绑定候选 |
| Go 回归 | store 的 `director_test.go` / `director_integration_test.go`、handler 的 `director_test.go`、`backend/migrations/upgrade_integration_test.go` | 局部验证、原子性与并发场景、迁移升级与重复执行 |
| 前端合同 | `frontend/src/api/director.ts`、`frontend/src/api/client.ts` | 镜头 DTO、API 与可选 directorContext |
| 导演台 | `frontend/src/features/director/`、`frontend/src/pages/client/DirectorPage.vue` | 镜头列表、编辑、候选、任务选择与创作上下文 |
| 现有页面串接 | `AssetLibrary.vue`、`CreatePage.vue`、`TaskDelivery.vue`、`AssetPickDialog.vue`、`CreateSettingsPills.vue` | 作品入口、明确应用、素材和任务回链；参考音频不进入导演流程 |
| 会话和导航 | `frontend/src/shared/lib/create-draft.ts`、`frontend/src/router/index.ts`、`frontend/src/widgets/SidebarNav.vue` | 镜头草稿单独存储、路由与资产导航归属 |
| 浏览器回归 | `frontend/tests/e2e/director.contract.spec.ts` 与 snapshots、`journey.visual.spec.ts` 与作品页 snapshots | 交互合同、六种宽度、桌面/手机视觉 |
| 文档 | `docs/ui/director-workspace-2026-10-05.md`、`docs/engineering/toonflow-integration-2026-10-05.md`、`docs/产品方案总览.md`、`docs/api/client-create.md`、本文 | 范围、来源、正式合同与证据边界 |

## 关键可靠性设计

- 请求幂等查询先于镜头校验；已有订单重放不受后续镜头删除、修改或供给下线影响。
- 首次提交锁作品/镜头，任务、扣费、消费订单、账本和候选关联在同一事务完成。
- 全板保存使用 revision，409 保留编辑；后台新增候选会递增 revision，不能被旧页面静默覆盖。
- 只接受当前用户拥有的素材/任务引用，回收站项目不可操作；已交付才允许选片。
- 普通创作草稿与每个镜头草稿分离，显式应用和恢复，不自动生成。
- 网络结果不确定时冻结并重放原请求；业务明确拒绝才解除待确认状态。
- 注销/会话失效立即清空私有镜头数据，不被未保存提示拦截。

## 修复与审阅证据

独立后端审查未发现确定的 P0/P1；发现大量素材引用逐次扫描的问题，已改为批量上传查询和一次作品素材扫描。独立前端审查发现并修复未保存提示拦截注销、导演任务收到作品 404 后永久待确认两处问题，均补充浏览器回归。

人工审图后，手机镜头列表改为最高 224px 内滚动，选中项保持可见；待交付候选顶标为“交付准备中”。查看了桌面导演台、手机顶部/编辑/候选区域，以及桌面和手机作品入口。基线使用明确标注的 API fixture，没有伪造真实生成媒体。

新增作品入口使手机旧视觉基线失败，确认是预期按钮与换行变化后更新；桌面对应入口基线一并更新。该作品页像素容差由 2% 收紧到 0，未提高任何容差掩盖差异。

## 执行命令与结果

所有前端浏览器测试使用 `VITE_API_PROXY=http://127.0.0.1:1`、`PLAYWRIGHT_REUSE_EXISTING_SERVER=0`，通过本地 API Mock 运行，不请求 107 或真实供应商。

| 分类 | 命令 | 结果 |
| --- | --- | --- |
| 静态 | frontend: `pnpm lint:check` | 通过 |
| 类型/构建 | frontend: `pnpm build --outDir ../.tmp_director_20261005_01/frontend-final`（先执行 `pnpm typecheck`） | 通过；Vite 2472 modules；临时产物无发布 |
| Go 单元/包测试 | backend: `go test -count=1 ./...` | 通过；需要基础设施的 integration 用例未因此获得实际 DB 证据 |
| 导演台及关联页面 | `pnpm test:visual tests/e2e/director.contract.spec.ts tests/e2e/journey.visual.spec.ts --grep 'director workspace\|journey-project-files' --reporter=dot --output=../.tmp_director_20261005_01/director-final --update-snapshots` | 32 通过：30 个导演台桌面/手机用例 + 2 个作品入口用例 |
| 整套浏览器回归 | `pnpm test:visual --reporter=dot --output=../.tmp_director_20261005_01/final-playwright` | 328 项中 313 通过、15 跳过、0 失败，用时 2.1 分钟 |
| 补丁检查 | `git diff --check` | 通过 |
| 真实数据库 | `SUB2VIDEO_RUN_INTEGRATION=1` 后运行 `go test -count=1 -v ./internal/store ./migrations -run 'TestDirector.*Integration\|TestUpgradeFrom019PreservesExistingDataIntegration'` | 3 个测试入口全部 SKIP，见下文；进程 exit 0 不代表 DB 通过 |

导演台浏览器覆盖：作品入口、新增/排序/保存刷新、409 保留与显式丢弃、只选已交付候选、移除镜头不取消任务、普通草稿保护、导演上下文提交、JSON 清单白名单、未知提交刷新后原 key/payload 重放、修订冲突重读、模式下线不静默切换、作品删除明确拒绝、注销私有内容清空、已有任务关联、首尾帧/多图上传引用鉴权预览，以及 320/390/768/1024/1440/1920px。

15 个浏览器 skip 是现有测试按设备分工避免重复执行：`auth-forms.contract`、`journey.contract`、`studio.contract` 的最小宽度场景仅运行 mobile，`responsive-layout.contract` 的自定义宽度矩阵仅运行 desktop。它们不是由于后端或供应商不可用而跳过；不要与真实数据库的 3 个环境 skip 混合计算。

## 未验证项和准确阻塞

1. 本地 PATH/标准安装位置未发现 Docker、Podman、PostgreSQL 或 Redis，WSL 无发行版。专用 `SUB2VIDEO_TEST_DATABASE_URL` 与 `SUB2VIDEO_TEST_REDIS_URL` 未配置。实际输出为 `SUB2VIDEO_TEST_DATABASE_URL is not configured; skipping PostgreSQL/Redis integration tests`。
2. 只读 SSH 连通性检查 `ssh -o BatchMode=yes -o ConnectTimeout=8 root@192.168.0.107 "true"` 返回 `Connection closed by 192.168.0.107 port 22`。没有取得 107 环境状态或数据库访问，不是服务健康结论，没有继续远端操作。
3. 数据库集成测试代码已补齐：两用户隔离、回收项目、素材/任务归属、选片状态、同 revision 并发、手工附加与整板保存竞争、同 key 并发单扣、不同 key 同 revision 单成功、写入失败全事务回滚、镜头删除/供给下线后原单重放，以及 019 升到 022/重复迁移。以上仍需真实隔离 PostgreSQL/Redis 执行。
4. 未执行新版本真实浏览器 API、实际订单/余额对账、真实候选文件下载或新付费视频生成；Mock 中 ready 任务只能验证界面条件。新的导演台因此尚不能认定已达封闭 Beta 发布门槛。

## 数据、发布与清理

未修改 `backend/data/`、`backend/server.linux`，未修改 UAT 数据、价格、供给或用户余额，未新增付费任务。未创建/切换分支，未执行 reset/checkout/clean；没有提交、推送、部署或重启。

发布需要应用 022 迁移并同步新后端与前端。旧数据默认为空镜头板；迁移实际运行、备份和回滚检查尚未执行，不能仅上传前端后宣布可用。

本轮临时根：`E:\apps\github.com\sbu2video\.tmp_director_20261005_01`，包含本地构建、Playwright 输出与失败追踪；不含凭据或业务数据。全部测试进程退出后核对精确路径与无 reparse point，使用 PowerShell `Remove-Item -LiteralPath` 清理，`Test-Path` 返回 `False`。删除的是可重新生成的产物；旧临时目录不动。审图后的 PNG 基线属于源测试资产，保留。

一个下一步：恢复专用 PostgreSQL/Redis（或可访问的 107 隔离测试环境），先实际运行导演台事务/权限/迁移测试，再决定是否同步发布 022 与前后端。
