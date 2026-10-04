# Sub2Video 产品体验验收与修复记录（2026-09-29）

## 结论与版本边界

本轮完成真实页面审计、针对性代码修复和分层回归。当前工作区可以作为下一次 UAT 的候选版本；当前 107 还不适合直接邀请 5～10 位创作者独立使用。最明确的阻塞是：107 创作页发送合法的 `durationSec: 5`，估价接口仍返回 `400 UNAVAILABLE_COMBINATION / 视频时长不合法`。此外，107 的下载仍打开视频页，失败详情仍可见上游错误文字，尚未使用本轮下载与状态修复。

这不是供应商连通性验收，也不是商业上线声明。没有创建付费生成任务，没有修改 UAT 资产、供给或价格，没有提交、推送、部署或重启服务。

| 项目 | 本轮版本 / 环境 |
| --- | --- |
| 工作区 | `E:\apps\github.com\sbu2video` |
| 分支 / HEAD | `codex/commercial-studio-redesign` / `a479c330d90b468fa17f2b0b3ce79a3f415b0dae` 加当前未提交修改 |
| UAT | `http://192.168.0.107:12000/`，真实已有账号会话 |
| UAT 前端版本证据 | 浏览器加载 `/assets/index-C72E6G3D.js`，与上一轮响应式发布记录一致 |
| UAT 后端版本证据 | 运行二进制启动及修改时间为 2026-09-26 18:08:21；构建标识为同一 HEAD 但 `vcs.modified=true`，不能用 commit 号认定它与当前工作区相同 |
| 本轮前端构建 | `pnpm build` 成功，入口 `index-C66Dxnp2.js`，未上传 |
| 本地浏览器 | Windows Chromium；内置浏览器检查 UAT 与本地登录入口；Playwright 检查当前代码 |
| 响应式范围 | 320、390、768、1024、1440、1920px；不是实体 iOS/Safari 验收 |
| 数据库测试 | 107 专用 `sub2video_test` 数据库中的随机 schema，Redis DB 15；通过临时本地 SSH 转发访问 |

工作区在开始前已有大量修改。下文仅列本轮触及的文件和行为，不把相对 HEAD 的全部 diff 认作本轮成果。

## 问题与修复表

“UAT”表示真实部署页面观察；“本地复现”表示当前代码或隔离测试，不能当作线上事故证据。没有证据表明本轮观察中发生过重复扣费或数据丢失。

| 页面 / 优先级 | 复现步骤与实际行为 | 用户影响 | 涉及文件 / 数据 | 修复与验收 |
| --- | --- | --- | --- | --- |
| 创作 / P1 / UAT | 文生视频，5 秒、768P、16:9、无音频；估价请求包含 `durationSec: 5`，返回“视频时长不合法”；页面仍允许提交 | 有合法参数却无法了解价格，错误文案还引导继续消费 | `CreatePage.vue`、`api/client.ts`、后端 `handler/client.go` | 本地在估价成功前禁止普通提交；保留正式 `durationSec` 合同；107 前后端版本差异须在发布时解决，不能宣称已线上修复 |
| 创作 / P1 / UAT | 首尾帧 0/2 张时显示“暂时无法预估费用，提交时将重新计算” | 用户无法区分缺素材与服务故障 | `CreatePage.vue`、`CreatePromptSubmit.vue` | 素材未就绪时不请求估价，显示补齐素材提示；存储、组合不可用、余额不足分别提示；桌面/手机合同测试通过 |
| 创作 / P1 / 本地复现 | 服务端可能已接单但响应丢失，刷新或重新点击会丢失客户端原请求身份 | 可能产生第二次购买，用户不知道如何恢复 | 新增 `pending-creation.ts`，`CreatePage.vue`；复用已有订单幂等约束 | 发送前按账号持久化原参数和请求 ID；失败后“确认原提交结果”重放原请求；不可持久化则不发请求；响应损坏、刷新、编辑草稿回归通过 |
| 创作 / P1 / 本地复现 | 本地 File 未上传，刷新后草稿残留 uploaded 标记 | 误认为图片已经准备好 | `create-draft.ts`、`CreatePage.vue` | 只有持久素材引用可跨刷新保持就绪；本地文件丢失后要求重新选择 |
| 任务 / P1 / 本地复现 | 列表按显示状态推断取消，无法区分已进入提交边界、结果待核对 | 用户可能收到无法兑现的取消/退款预期 | `store/tasks.go`、`handler/client.go`、`WorksPage.vue`、`TaskDelivery.vue`；已有 tasks/ledger | 服务端统一计算 `canCancel` / `reviewRequired`；客户端缺字段不推断可取消；取消竞争失败保留提示；真实 DB 边界及退款幂等回归通过 |
| 任务详情 / P1 / UAT | 已有失败任务显示上游 AutoDL 原始错误，详情没有对应真实退款金额/时间 | 暴露内部供应商，用户无法确认钱是否退回 | `handler/client.go`、失败合同测试、交付组件；wallet_ledger | 本地错误映射为用户可理解的原因，仅有退款证据才声明已退；107 尚未更新 |
| 下载 / P1 / UAT | 已成功任务可播放，点击下载打开新视频标签，30 秒内未捕获下载事件 | “下载”与实际行为不一致 | 新增 `features/delivery/download.ts`；`handler/delivery.go`，blob/MinIO/COS | 通过任务归属校验后生成独立 attachment 签名；签名失败明确返回错误，不回退到预览；真实 DB 鉴权、SDK 签名及跨域浏览器下载 fixture 通过；107 实际附件下载未验收 |
| 作品定位 / P1 / 本地复现 | 保存后位置只存在页面状态，刷新难以回到成片目录 | 成片保存与资产管理路径断裂 | `store/tasks.go`、`handler/client.go`、`api/client.ts`、`TaskDelivery.vue`；已有 projects.folders_json/sourceTask | 返回持久 `savedDestination`，回收站/删除后不返回失效路径；真实 DB 保存和重新读取回归通过 |
| 钱包 / P2 / UAT | 提示充值未开放，但仍有充值档位与操作入口；加载异常易被误读为空 | 影响付费信任 | `WalletPage.vue`、`TermsPage.vue` | 移除未开放充值交互；保留真实余额、流水、兑换；明确读取失败与重试、取消边界；移动端先显示流水 |
| 工单 / P2 / 本地复现 | 查看工单原为通知式占位，失败任务上下文不足 | 用户找不到已提交问题和处理状态 | `TicketsPage.vue`、`TaskDelivery.vue`、`handler/client.go` | 使用现有弹窗显示原问题/状态，不编造客服回复；任务详情跳转保留 taskId；Mock 合同测试通过，UAT 未新建工单 |
| 供应商成本 / P1 / 本地复现 | 编辑多分辨率执行配置只用一组成本覆盖多个维度；加载失败可能按空值保存 | 配置价格与管理员意图不一致 | `SupplierModelsPage.vue`；已有 supplier_offerings/supplier_cost_rules | 按分辨率与音频维度分别编辑，仅提交变更；空值不是 0；读取失败禁止保存，部分失败保留并重试未保存项；Mock 请求断言通过，未操作 UAT 成本 |
| 管理列表 / P2 / 本地复现 | 供应商选项只取首批、嵌套执行缺分页、任务搜索只过滤当前页 | 找不到已有记录、以为数据丢失 | `SupplierModelsPage.vue`、`TasksPage.vue`、`api/admin.ts`、`store/commerce.go` | 执行与任务服务端分页；任务关键词送后端；忽略过时响应；真实 DB 跨页搜索和 Mock 分页回归通过 |
| 平台价格 / P1 / 本地复现 | 编辑可改规则维度，空金额会被转为零；后端新建模式价的 NULL 表达式类型不正确 | 修改错规则或无法新增价格 | `PricingPage.vue`、`store/creation_catalog.go`；retail_price_rules | 编辑锁定维度，旧模型绑定规则只读，空值拒绝；修复 `NULL::bigint`；真实 DB 新增/更新模式价回归通过 |
| 图片素材 / P1 / 本地复现 | 用户自有音频 uploadId 放入图片槽，旧归属校验不拒绝 MIME | 错素材进入收费生成链路 | `store/tasks.go`、新增集成测试；uploads | 图片槽仅接受已支持图片 MIME，在扣费前拒绝音频；首帧/尾帧/多图均验证余额和任务/订单/流水不变；未扩展参考音频功能 |
| 320px 顶栏 / P2 / UAT | 余额货币符号与数字换成两行 | 小屏层次混乱 | `shared/ui/studio.css`、responsive 测试 | 最小宽度收敛字标和装饰图标，金额不换行，保持 44px 触达；新增 320px 截图并人工审图 |

本轮不新增迁移、数据表或独立设计系统。使用原有订单、任务、账本、目录模型和签名接口抽象。模型能力标签等既有功能未借此扩展新模型或新供应商。

## 真实页面观察

- 官网说明三种单镜头创作入口；参考影片明确标注“实拍影像参考 / 非生成成片”，未将其当作生成案例。教程和准备步骤可读。
- 已有登录会话下，顶栏、创作页、钱包均显示 ¥255.96；没有复现本轮线上余额不一致。新账号登录和注册未执行。
- 文生视频无扣费估价失败已经捕获真实请求和响应；首尾帧缺素材状态也可复现。未绕过界面直接制造“估价成功”。
- 成功任务 `tsk_…8200953b69ab` 的真实视频在浏览器解码到 readyState 4，864×480，时长约 1.625 秒；这证明该历史成片当前可预览，不证明当前工作区新生成成功。
- 点击上述任务下载未得到浏览器文件下载事件；只打开视频标签，因此下载结果记为失败，而不是用 HTTP 200 记通过。
- 失败任务 `tsk_…a04440f06041` 的钱包有 -¥0.36 消费与 +¥0.36 退款，记录时间为 09-10 19:24。这是已有账本证据，不是本轮新失败退款实验。详情仍未清楚呈现这笔退款。
- 真实资产页可读取 default 作品与 9 个文件；没有上传、保存、移动、删除或修改真实资产。
- 管理端读取了供应商、执行配置、任务与价格页面，打开并关闭编辑弹窗；未保存配置、激活、禁用或核对任务。
- UAT 创作页六档宽度均无文档级横向溢出；768/1024px 为单栏，1440/1920px 为两栏。320px 的余额换行问题已在本地修复。

## 本地实现与视觉验收

采用现有 UI/UX、Vue 与 Vue 测试技能辅助检查，设计系统仍以 `docs/ui/design-system.md` 为唯一依据。没有改品牌、调换技术栈或重做全站。

截图更新是可审阅的预期变化：钱包流水优先且移除假充值入口、创作显示准备与恢复状态、管理列表和说明准确、小屏余额单行。没有提高截图容差。人工查看了创作、钱包、工单、任务、供应商、价格页面的桌面/移动截图，以及待核对提交和 320px 新基线。

可审阅的代表截图（内容是确定性测试数据）：

- [钱包桌面](../../frontend/tests/e2e/surfaces.visual.spec.ts-snapshots/client-wallet-desktop-win32.png)
- [钱包手机](../../frontend/tests/e2e/surfaces.visual.spec.ts-snapshots/client-wallet-mobile-win32.png)
- [320px 创作](../../frontend/tests/e2e/responsive-layout.contract.spec.ts-snapshots/responsive-create-320-desktop-win32.png)
- [提交恢复手机](../../frontend/tests/e2e/creation.contract.spec.ts-snapshots/pending-creation-mobile-win32.png)
- [管理端供应商](../../frontend/tests/e2e/surfaces.visual.spec.ts-snapshots/admin-supplier-models-desktop-win32.png)

## 验证结果

| 层次 / 命令 | 结果 | 证据范围 |
| --- | --- | --- |
| `frontend: pnpm lint:check` | 通过，退出 0 | 静态规则 |
| `frontend: pnpm typecheck` | 通过，退出 0 | TypeScript/Vue 类型 |
| `frontend: pnpm build` | 通过，Vite 22.21 秒 | 当前工作区可构建；未发布 |
| `frontend: pnpm test:visual --reporter=list --output=../.tmp_experience_20260929/final-visual` | 265 通过、15 跳过、0 失败，1.9 分钟 | 当前 Vue 实现、Mock 交互和视觉；不证明真实供应商/钱包 |
| `backend: go test ./...` | 通过，退出 0 | 默认测试；需环境的集成用例在此命令中按开关跳过，另见下面实际执行 |
| 真实 Store 集成测试首轮 | 12 项顶层测试通过，289.983 秒 | 真实 PG/Redis；任务/账本/订单、取消、核对恢复、租约、保存位置、管理分页和价格写入 |
| `TestDeliveryURLHandlerIntegration` | 通过，16.865 秒，8 个子场景 | 真实 PG 归属与 SDK 签名；对象 HTTP 服务为本地 fixture，不证明真实 MinIO/COS 交付 |
| 图片 MIME 限定回归 | 2 项顶层、5 个子场景通过，49.213 秒 | 真实 PG：错误图片引用不扣费；合法图片和音频原路径不受影响 |
| `TestEstimateDurationSecHTTPContractHasNoFinancialSideEffectsIntegration` | 通过，17.229 秒 | 与浏览器相同的 `durationSec: 5` HTTP 请求在当前 handler 返回估价 0.15；余额 100 不变，任务/订单/quote/流水全 0，供应商请求为 0；存储可用性使用本地 GET/HEAD fixture |
| `git diff --check` | 通过，退出 0 | 有既有 LF/CRLF 提示，无空白错误 |

15 项 Playwright skip 为避免项目重复：12 个自定义宽度只在 desktop 项目运行（实际包含 320/390px），3 个最小手机测试只在 mobile 项目运行。没有把基础设施缺失包装成这些用例通过。

真实集成环境变量仅安全注入进程，不在报告或脚本保存凭据：

```text
SUB2VIDEO_RUN_INTEGRATION=1
SUB2VIDEO_TEST_DATABASE_URL=postgres://<user>:<secret>@127.0.0.1:25432/sub2video_test?sslmode=disable
SUB2VIDEO_TEST_REDIS_URL=redis://127.0.0.1:26379/15
```

在 backend 目录执行的测试选择：

```powershell
go test ./internal/store -count=1 -timeout 15m -v -run '^Test(ClientTaskActionsMatchCancellationIntegration|ClientTaskSavedDestinationSurvivesReloadIntegration|RetailPriceInsertAndUpdateRemainModeScopedIntegration|AdminTaskSearchPaginatesAcrossAllMatchingRowsIntegration|CancelTaskIsIdempotentAfterResponseRetry|DirectSubmissionBillingRefundAndSnapshotIntegration|DirectSubmissionRollbackAndConcurrentIdempotencyIntegration|DirectImageSubmissionUsesOneTransactionConnectionIntegration|ProviderReconciliationRefundAtomicAndIdempotentIntegration|ProviderKnownAcceptedIDCannotBeRefundedAsUnacceptedIntegration|ProviderCrashRecoveryAndResumeIntegration|ProviderLeaseAndRefundStateIntegration)$'
go test ./internal/handler -count=1 -timeout 3m -v -run '^TestDeliveryURLHandlerIntegration$'
go test ./internal/store -count=1 -timeout 3m -v -run '^Test(ImageSlotsRejectAudioUploadsBeforeChargeIntegration|DirectImageSubmissionUsesOneTransactionConnectionIntegration)$'
go test ./internal/handler -count=1 -timeout 3m -v -run '^TestEstimateDurationSecHTTPContractHasNoFinancialSideEffectsIntegration$'
```

主要新增/扩展前端合同包括：响应丢失刷新恢复、响应损坏、浏览器存储失败、估价失败、素材未就绪、余额不足、真实保存位置的界面使用、下载失败、跨域附件 fixture、待核对任务、工单上下文、逐维度成本更新、部分失败重试与服务端分页。

## 本轮文件范围

- 创作：`frontend/src/pages/client/CreatePage.vue`、`frontend/src/features/create/workbench/CreatePromptSubmit.vue`、`frontend/src/shared/lib/create-draft.ts`、新增 `frontend/src/shared/lib/pending-creation.ts`。
- 交付与账户：`frontend/src/api/client.ts`、`frontend/src/features/delivery/TaskDelivery.vue`、新增 `frontend/src/features/delivery/download.ts`、`frontend/src/pages/client/WorksPage.vue`、`WalletPage.vue`、`TicketsPage.vue`、`frontend/src/pages/auth/TermsPage.vue`。
- 管理：`frontend/src/api/admin.ts`、`frontend/src/pages/admin/SupplierModelsPage.vue`、`PricingPage.vue`、`TasksPage.vue`。
- 小屏共享样式：`frontend/src/shared/ui/studio.css`。
- 后端：`backend/internal/store/tasks.go`、`commerce.go`、`creation_catalog.go`；`backend/internal/handler/client.go`、`admin.go`、`delivery.go`；`backend/internal/storage/blob/store.go`、`minio/client.go`、`cos/client.go`。
- 后端测试：`backend/internal/store/client_task_state_integration_test.go`；`backend/internal/handler/client_contract_test.go`、`client_failure_contract_test.go`、`delivery_test.go`、`delivery_integration_test.go`、新增 `estimate_integration_test.go`。
- 前端测试：新增 `frontend/tests/e2e/admin.experience.contract.spec.ts`、`experience-delivery.contract.spec.ts`；扩展 `creation.contract.spec.ts`、`studio.contract.spec.ts`、`journey.contract.spec.ts`、`responsive-layout.contract.spec.ts` 及关联截图。
- 文档：`docs/api/client-tasks.md`、本记录。

## 尚未验证及恢复步骤

1. **当前候选版本尚未部署到 107。** 不能用历史成片、旧部署或本地 Mock 来填补这个缺口。授权发布后，先确认前后端来自同一候选版本，再重做无扣费估价、已有任务详情与附件下载。
2. **真实新账号登录、素材上传、新任务提交、保存和删除本轮未在 UAT 执行。** 原因是未授权写入真实用户资产，也未取得本轮专用 UI 测试账号/隔离服务。最小条件是同版本专用测试账号与隔离数据范围；付费生成仍须另列参数及预算确认。
3. **新下载功能尚无真实对象存储浏览器文件落盘证据。** 已完成签名、鉴权和跨域 fixture；必须在同版本 UAT 对已有成片重新点击下载，验证文件可播放，不只检查状态码。
4. **新退款/保存定位界面尚无同版本 UAT 证据。** 数据库测试和前端合同分别通过，需发布后用已有任务只读复核组合行为。
5. **提交恢复记录仅约束同一账号、浏览器持久记录内的原请求重放。** 不声称不同设备、手动清除存储、另建新请求也代表同一次购买。恢复记录损坏时先查任务或联系支持，不引导清空记录直接再买。
6. **成本多维更新仍由既有多次 API 写入完成。** 界面准确显示部分成功并仅重试未保存项；没有宣称整组设置是单事务。
7. **MIME 修复依据已保存上传记录。** 本轮未引入媒体内容解码安全检测，也没有扩展参考音频能力。
8. 手机为 Chromium 视口模拟，实体 Safari、弱网设备及真实触摸键盘仍未验收。

## 数据、进程和清理

没有触碰 `backend/data/`、`backend/server.linux`，没有运行迁移到 UAT。集成测试仅使用专用测试库随机 schema；清理后确认测试 schema、非系统表、其他测试连接、Redis DB15 key 均为 0。没有在磁盘保存测试凭据。

本轮 SSH 隧道和本地 4186 预览进程已停止；浏览器视口已恢复，本地登录预览标签已关闭。一个先前连接失败生成的浏览器错误页因 `data:` URL 策略无法通过工具关闭，可由用户手动关闭；与应用运行无关。

本轮可清理的临时路径只有：

- `E:\apps\github.com\sbu2video\.tmp_experience_20260929`
- `E:\apps\github.com\sbu2video\frontend\.tmp_experience_20260929`

本轮测试输出保留供复核；不清理任何以前的 `.tmp_*` 目录。正式测试截图与本记录不是临时文件。`frontend/dist` 是本次验证生成的忽略构建产物，未提交、未上传。

## 私测建议

现在可评审本地体验修复和测试证据，暂不建议把当前 107 直接交给 5～10 位用户独立创作。进入封闭试用的下一步只有一个：经授权，将同一候选版本的前后端同步发布到 107，并复核估价、旧成片附件下载、失败退款详情和保存位置。通过后再使用专用测试账号完成受控写入验收，再发邀请。
