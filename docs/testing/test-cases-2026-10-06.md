# Sub2Video 封闭 Beta 测试用例库

编制日期：2026-10-06。范围：官网、认证、单镜头创作、任务与财务、交付资产、导演台、管理操作及发布恢复。共 112 条用例（TC-001—TC-112）。本文是可执行用例库，不是通过报告；本轮真实执行结果由协调者写入独立验收记录，不复制历史成功结论。

## 使用方法与证据规则

- 优先级：P0 为重复扣费、越权、数据丢失或不可恢复的执行错误；P1 为核心流程阻断、状态误导或重要恢复缺失；P2 为布局、理解和交互一致性问题。
- 「准备状态」只用三种值：**已有自动化**表示存在对应测试定义，**待补自动化**表示仅部分覆盖或无对应测试，**待执行**表示本用例以人工或受控环境验收为主。三者都不表示已通过。
- 自动化列采用「文件别名 · 精确测试名」。别名在下表唯一映射到文件；动态标题保留源码模板并写明参数。标为“部分”的测试只能证明对应子步骤，剩余步骤仍须执行。新增但未运行的测试也只能标为已有自动化。
- 证据分类：**M**=浏览器 Mock 交互；**V**=视觉截图；**U**=纯单元或本地 HTTP 故障服务器；**D**=真实 PostgreSQL 数据/事务；**L**=真实浏览器或 HTTP API 环境；**O**=真实对象存储；**P**=真实供应商；**H**=人工/真机。D 可结合假上游，不能因此升级为 P；L 的本地图片预览不能证明已上传。
- 运行记录必须包含版本（HEAD 与未提交差异摘要）、环境、用例 ID、实际参数、命令、结果、脱敏任务/订单 ID、断言或截图/trace 路径、skip 原因和清理范围。失败和未验证必须保留，不能只统计退出码。
- UI 的尺寸、对齐、容器断点、颜色、焦点、触摸和截图标准全部引用 [ui-layout-acceptance.md](./ui-layout-acceptance.md) 的标准 ID；唯一视觉合同仍是 [design-system.md](../ui/design-system.md)。本文不创建第二套视觉阈值。
- 支付渠道与参考音频新功能不在本轮实现范围；只检查充值未开放时的真实说明，不通过模拟支付制造成功证据。

## 共享角色、环境和前置条件

| 代号 | 角色/前置 |
| --- | --- |
| G | 未登录访客；清除本轮浏览器上下文中的会话，不操作其他用户浏览器资料。 |
| C | 专用创作者 A；有授权测试账号和当前目录。需写入时只用隔离环境及本轮数据前缀。 |
| C2 | 第二个专用创作者 B；与 A 的作品、素材、任务明确分离。 |
| A | 专用管理员；配置变更仅在隔离环境执行；记录变更前值、审计及恢复。 |
| E | 测试工程师；允许启动本地 Mock/故障服务器、检查隔离数据库；不改 UAT 业务库。 |
| R | 发布操作者；只有另行授权发布窗口才操作服务、备份或恢复。本文不是部署授权。 |
| M0 | 本地当前工作区前端，固定 fixture、本地字体；未知业务请求必须失败并被记录，不能落到真实后端。 |
| D0 | `SUB2VIDEO_RUN_INTEGRATION=1`，专用 `SUB2VIDEO_TEST_DATABASE_URL`（库名含 test）；随机 schema 且 search_path 仅该 schema。Store 公共 helper 另需隔离 `SUB2VIDEO_TEST_REDIS_URL`。 |
| L0 | 当前构建前后端、专用账号、批准的读写范围；不预设供给、价格、余额或存储可用。记录无供给等真实状态也是有效证据，但不是成功估价证据。 |
| W0 | D0＋本地假供应商；真实 Runner 执行，用触发器/控制时钟制造故障。不得调用真实供应商。 |
| F0 | 已知金额的合成钱包、任务和账本；同时保存余额、流水、消费订单和任务快照以便对账。只存在于隔离数据。 |
| V0 | 按 UI 标准的六宽度和字体条件；截图中样例任务明确是 fixture，不能充当交付证明。 |

真实 UAT 默认仅允许登录、读取、无扣费估价、已有结果预览/下载。上传、保存、删除等需明确专用账号和数据范围。真实付费生成需先列参数、账号、费用、预算并获得确认；下面任何用例都不能隐式解除该限制。测试 trace 不保存认证表单、Authorization、Cookie 或服务端凭据；证据只保存脱敏摘要。

## 自动化文件索引

所有路径均相对仓库根目录。测试名必须与当前文件核对，不能仅凭目录名判断证据等级。

| 别名 | 精确文件 |
| --- | --- |
| F-PUB | `frontend/tests/e2e/public-home.contract.spec.ts` |
| F-AUTH | `frontend/tests/e2e/auth-forms.contract.spec.ts` |
| F-SESSION | `frontend/tests/e2e/auth-session.contract.spec.ts` |
| F-JOURNEY | `frontend/tests/e2e/journey.contract.spec.ts` |
| F-CREATE | `frontend/tests/e2e/creation.contract.spec.ts` |
| F-STUDIO | `frontend/tests/e2e/studio.contract.spec.ts` |
| F-AVAIL | `frontend/tests/e2e/create-availability.contract.spec.ts` |
| F-DELIVERY | `frontend/tests/e2e/delivery.contract.spec.ts` |
| F-AFTERCARE | `frontend/tests/e2e/experience-delivery.contract.spec.ts` |
| F-DIRECTOR | `frontend/tests/e2e/director.contract.spec.ts` |
| F-ADMIN | `frontend/tests/e2e/admin.experience.contract.spec.ts` |
| F-RECON | `frontend/tests/e2e/admin.task-reconcile.contract.spec.ts` |
| F-CATALOG | `frontend/tests/e2e/admin.catalog.contract.spec.ts` |
| F-SUPPLIER | `frontend/tests/e2e/admin.supplier.tabs.contract.spec.ts` |
| F-LAYOUT | `frontend/tests/e2e/responsive-layout.contract.spec.ts` |
| F-UI | `frontend/tests/e2e/ui-standards.contract.spec.ts` |
| F-AUTH-V | `frontend/tests/e2e/auth.visual.spec.ts` |
| F-JOURNEY-V | `frontend/tests/e2e/journey.visual.spec.ts` |
| F-SURFACE-V | `frontend/tests/e2e/surfaces.visual.spec.ts` |
| L-ENTRANCE | `frontend/tests/integration/live-public-entrance.spec.ts` |
| L-CREATE | `frontend/tests/integration/live-creation.spec.ts` |
| L-UPLOAD | `frontend/tests/integration/live-upload.spec.ts` |
| L-DIRECTOR | `frontend/tests/integration/live-director.spec.ts` |
| L-READONLY | `frontend/tests/integration/uat-readonly.spec.ts` |
| B-CATALOG | `backend/internal/store/creation_catalog_integration_test.go` |
| B-BETA | `backend/internal/store/direct_beta_integration_test.go` |
| B-REPLAY | `backend/internal/store/direct_routing_integration_test.go` |
| B-CANCEL | `backend/internal/store/cancel_idempotency_integration_test.go` |
| B-STATE | `backend/internal/store/client_task_state_integration_test.go` |
| B-TX | `backend/internal/store/direct_transaction_integration_test.go` |
| B-SUPPLIER | `backend/internal/store/supplier_offering_integration_test.go` |
| B-DIRECTOR | `backend/internal/store/director_integration_test.go` |
| B-RECON | `backend/internal/store/provider_reconciliation_integration_test.go` |
| B-WORKER | `backend/internal/worker/provider_integration_test.go` |
| B-ADAPTER | `backend/internal/provider/autodl/client_test.go` |
| B-ESTIMATE-H | `backend/internal/handler/estimate_integration_test.go` |
| B-DELIVERY-H | `backend/internal/handler/delivery_integration_test.go` |
| B-DELIVERY-U | `backend/internal/handler/delivery_test.go` |
| B-DIRECTOR-H | `backend/internal/handler/director_integration_test.go` |
| B-DIRECTOR-U | `backend/internal/handler/director_test.go` |
| B-RECON-H | `backend/internal/handler/provider_reconciliation_integration_test.go` |
| B-CLIENT-U | `backend/internal/handler/client_contract_test.go` |
| B-ERROR-U | `backend/internal/handler/client_failure_contract_test.go` |
| B-SHAPE-U | `backend/internal/store/source_shape_test.go` |
| B-CURRENCY-U | `backend/internal/store/direct_currency_test.go` |
| B-MIGRATE | `backend/cmd/server/main_integration_test.go` |
| B-UPGRADE | `backend/migrations/upgrade_integration_test.go` |
| B-MINIO | `backend/internal/storage/minio/client_integration_test.go` |

## 1. 官网与教程（TC-001—006）

目标：访客能理解单镜头能力、可靠进入创作，公开页面不访问私有资料或诱发生成。共享前置：G＋M0；真实复测用 L0，但不提交任务。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-001 / P1 | G；监听业务请求 | 打开 `/`；滚动到模式、教程和 FAQ；等待网络稳定。 | 主内容无需登录；没有 client/admin 私有读取、估价、创建任务请求。 | F-PUB · `is public and does not read private data, estimate, or create a task`（M）；L-ENTRANCE · `continues a public shot through real login and logout without creating a task`（L，仍需本轮运行）。 | 已有自动化 |
| TC-002 / P1 | G；三模式分别执行 | 点击每个模式入口；完成登录；检查回跳模式及已有草稿。 | 保留选择模式；不自动提交；不擅自覆盖已有描述和素材。 | F-PUB · `${mode.inputType} entry survives login and changes only the selected mode of an existing draft`（text/first_last/multi_image，M）。 | 已有自动化 |
| TC-003 / P0 | G；构造外部 redirect 查询 | 从带外部跳转参数的官网进入登录；分别走登录/注册链接。 | 只跳转可信内部目的地；不向外站携带认证结果。 | F-PUB · `uses a safe creation destination even when the public URL contains an external redirect`；F-JOURNEY · `preserves the tutorial destination through registration and rejects an external login destination`（M）。 | 已有自动化 |
| TC-004 / P1 | C；已有未保存草稿 | 打开教程；返回创作但不应用；随后明确点击应用教程。 | 未确认前原草稿保留；确认后才变更教程参数；教程图片不变成用户已上传素材。 | F-JOURNEY · `keeps an existing draft until the creator explicitly applies tutorial settings`；`${guide.id} requires the creator's images instead of treating public artwork as uploaded material`（M）。 | 已有自动化 |
| TC-005 / P2 | G；按 reduced-motion、手机、save-data 分组 | 分组打开官网；检查媒体请求；主动播放并注入失败；用键盘展开 FAQ。 | 默认媒体策略与 UI A04 一致；失败保留封面、创作和教程入口；FAQ 可键盘开合。 | F-PUB · `does not fetch reference video automatically with reduced motion`、`keeps mobile visitors on the poster even when motion is allowed`、`respects data saving without downloading a video`、`opens the tutorial section and toggles every FAQ with the keyboard`（M）；后台暂停另待人工。 | 已有自动化 |
| TC-006 / P1 | G＋H；当前实际官网/教程 | 逐段对照发布能力检查标题、价格、样片、版权说明和充值承诺；点所有主要 CTA。 | 明示单镜头及外部剪辑边界；参考片有来源而非假客户案例；不承诺未上线支付/整片生成；入口有效。 | 无完整语义自动化；F-PUB 只覆盖部分交互。需记录具体文案与截图，不能以视觉匹配代替内容核验。 | 待执行 |

## 2. 认证与会话（TC-007—016）

目标：真实表单、清晰故障与账户隔离。共享前置：G/C/C2＋M0；真实账号生命周期必须用隔离账号，邮件是否送达单独记录。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-007 / P1 | G；第一次登录返回错误 | 输入邮箱密码；切换密码可见；Enter 提交；修复响应后重试。 | native form 与 autocomplete 合理；错误可见且不清空输入；重试成功按内部目的地跳转。 | F-AUTH · `login uses real form semantics, reveals password and retries without clearing fields`（M）；L-ENTRANCE 覆盖真实登录，不能证明真实注册。 | 已有自动化 |
| TC-008 / P1 | G；公开注册配置依次失败、关闭、开放 | 打开注册；配置失败后重试；切换隔离 fixture 的开放状态并刷新。 | 未取得配置不能提前允许注册；关闭时无可提交表单或假重置入口；开放后正常显示。 | F-AUTH · `registration is gated by settings and can recover from an outage`、`closed registration hides the form and login advertises no fake registration or reset action`（M）。 | 已有自动化 |
| TC-009 / P1 | G；注册开放 | 依次缺验证码、未同意条款、弱密码；请求验证码并在冷却期重复点击；填写合法值后提交。 | 每个门槛单独阻止提交并说明原因；冷却期不重复发请求；仅完整条件允许提交。 | F-AUTH · `verification, cooldown, consent and password requirements are real registration gates`（M）；邮件送达、服务端验证码消费及注册持久化待补 D/L。 | 已有自动化 |
| TC-010 / P1 | G；认证背景视频/封面失败 | 分别让视频失败、视频和海报都失败；操作登录表单及媒体暂停。 | 表单和品牌说明始终可用；不显示虚假播放成功；可播放本地片可显式暂停。 | F-AUTH · `media failure leaves an informative poster and a usable login form`、`a missing poster and video do not hide the login or brand copy`、`local background film really plays and pauses on request`（M，播放的是参考片）。 | 已有自动化 |
| TC-011 / P0 | C/C2；A 有草稿和缓存 | A 编辑后退出，B 登录，再回 A；让 A 的迟到请求在 B 会话中返回。 | B 不能读 A 草稿/项目；A 原草稿仍可恢复；无归属的旧草稿不自动归给 B；迟到响应不污染新会话。 | F-SESSION · `isolates drafts across A to B to A without discarding the original draft`、`never assigns an unowned legacy draft to the current account`、`clears private project cache on account change and ignores a late previous-account response`（M）；服务端身份复用回归待补。 | 已有自动化 |
| TC-012 / P1 | C；过期 access token，可刷新 | 冷启动私有页；让多个读取同时 401；第二标签页也触发刷新。 | 私有数据在续期后加载；单会话并发共享刷新；两标签页不互相作废新 token。 | F-SESSION · `renews an expired token during bootstrap before loading private page data`、`shares a rotating refresh across two same-origin tabs`（M）；真实轮换持久化另需 D/L。 | 已有自动化 |
| TC-013 / P1 | C；刷新分别返回临时 503、确定失效 | 在已编辑创作页触发刷新失败；临时故障显式重试；确定失效后重新登录。 | 临时故障保留会话及草稿；确定失效才回登录并保留内部目的地；重放 401 不形成刷新死循环。 | F-SESSION · `keeps the session after a temporary refresh failure and allows an explicit page retry`、`redirects an expired mounted session and preserves the creation destination and text draft`、`terminates a replayed 401 without a second refresh or stale authenticated state`（M）。 | 已有自动化 |
| TC-014 / P0 | C；两标签页；退出请求返回 503/迟到 | 退出 A；立即登录 B；让原退出/刷新响应到达；查看另一标签页。 | 本地私有状态已清除；迟到结果不能复活 A 或踢掉 B；跨标签退出传播，A 草稿仍按账户隔离。 | F-SESSION · `leaves protected pages and clears local credentials when logout revocation returns 503`、`allows another login while logout is pending and ignores its late response`、`propagates logout to another tab without losing its unfinished text draft`（M）。 | 已有自动化 |
| TC-015 / P0 | C，非管理员 | 直接输入管理 URL；再用 A/B 身份分别读取对方任务、作品和导演台 ID。 | UI 不显示管理页；服务端拒绝跨用户对象访问且不泄露其存在/内容；UI 路由守卫不能替代后端鉴权。 | F-CREATE · `redirects a creator account away from the protected admin routes`（M）；B-DELIVERY-H · `TestDeliveryURLHandlerIntegration`、B-DIRECTOR-H · `TestDirectorBoardHandlerIntegration`（D/Handler）。 | 已有自动化 |
| TC-016 / P0 | E＋D0；专用身份数据，可控制 JWT 密钥/过期 | 建立真实账号、登录/续期/注销；删除测试账号并验证旧凭据；模拟重置后 public ID 冲突，重放旧 JWT。 | 已注销/失效/旧身份凭据不能取得新用户资产；数据库身份与会话生命周期一致；不能仅靠前端清缓存。 | 当前未发现完整 auth/middleware 回归；Handler 集成直接签测试 JWT。需新增服务端身份生命周期测试，重置测试不得在 UAT 执行。 | 待补自动化 |

## 3. 创作模式、能力与草稿（TC-017—024）

目标：用户明确选择模式、承运商、模型和参数，不能被隐式替换。共享前置：C＋M0；数据库能力验证用 D0。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-017 / P1 | C；目录仅开放部分模式 | 打开概览并依次进入开放模式；再改变 fixture 顺序/禁用一模式。 | 只展示已启用模式并按客户端固定顺序；各入口选中正确模式；未满足图片条件不提前估价。 | F-CREATE · `renders only enabled catalog modes in the client-defined order`；F-STUDIO · `opens each overview entry and waits for required images before estimating`（M）。 | 已有自动化 |
| TC-018 / P1 | C；文生模式可用 | 选择文生、承运商/模型、合法参数；填写描述；检查估价与提交体，不点击真实付费提交。 | 使用当前 `durationSec` 合同和所选模型；无 quoteId；文生不被带入旧图像素材。 | F-CREATE · `estimates a selected model and submits the complete creation contract`（M）；B-BETA · `TestDirectCapabilityCombinationsAndModelPreferenceIntegration`（D）。 | 已有自动化 |
| TC-019 / P1 | C；首尾帧模式可用 | 只选首帧观察状态；补尾帧；删除尾帧；再将同一已上传图片用于两个槽位。 | 缺尾帧明确提示素材不足；两个槽位完整才允许估价；槽位数量按位置计算，不因同 ID 去重为一张。 | F-CREATE · `distinguishes missing materials and low balance from service failures`（M）；B-BETA · `TestDirectCapabilityCombinationsAndModelPreferenceIntegration`（D，同图两槽）。 | 已有自动化 |
| TC-020 / P1 | C；多图模式能力明确 | 从 0 张添加至允许范围；删除一张；尝试超上限；观察请求中的 refs 顺序。 | 空素材和超上限被拒绝且不扣费；合法图按用户顺序进入路由能力检查。 | B-BETA · `TestDirectCapabilityCombinationsAndModelPreferenceIntegration`（D，0/2/10 边界）；F-STUDIO · `includes selected visual references in estimate capability routing`（M，非全上限矩阵）。 | 已有自动化 |
| TC-021 / P1 | E＋D0；已知支持矩阵 | 分别传入不支持的时长、分辨率、比例、自动音频和图像数量；再恢复合法组合。 | 不可用组合稳定拒绝；合法组合恢复；不通过替换模型使测试通过。 | B-BETA · `TestDirectCapabilityCombinationsAndModelPreferenceIntegration`；B-SHAPE-U · `TestCreationSourceShapeAndSlotCount`（D/U）。 | 已有自动化 |
| TC-022 / P1 | C；显式选定模型随后无供给 | 打开当前草稿；让选项返回模型组合不可用；点重新加载；监控所有 modelId。 | 明确是组合不可用，不误报网络；保持模型和描述；没有静默切换、估价或任务 POST。 | F-AVAIL · `explains a selected model without usable supply without blaming the network or switching models`（M）；B-BETA · `TestDirectSubmissionRejectionsLeaveNoFinancialRecordsIntegration`（D）。 | 已有自动化 |
| TC-023 / P1 | C；选项请求先网络失败，再服务端异常 | 分别注入断网/500；检查提示；恢复网络后点击重新加载。 | 区分网络、服务异常、无供给；保留草稿；重试只恢复能力和估价，不自动提交。 | F-AVAIL · `recovers a network failure through explicit retry while retaining the draft and model`、`distinguishes a server error from a network or supply failure and hides diagnostics`（M）。 | 已有自动化 |
| TC-024 / P1 | C；教程/导演镜头原模式不可用 | 打开不可用或未知教程；应用已有导演镜头到创作。 | 显示可行动的不可用说明；不替换成另一模式；原镜头/草稿仍存在。 | F-JOURNEY · `does not switch an unavailable tutorial mode or accept an unknown tutorial`；F-DIRECTOR · `does not silently switch a shot whose original mode is unavailable`（M）。 | 已有自动化 |

## 4. 素材准备与上传（TC-025—030）

目标：本地选择、真实上传、素材归属和持久恢复边界清楚。共享前置：C/C2＋M0；真实上传仅 L0 的专用隔离账号。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-025 / P1 | C；仅选择本地文件，尚未上传 | 打开素材选择器，选图片；刷新创作页。 | 刷新后不伪装为已上传素材；提示重新选择；不把 blob URL 当服务器资产 ID。 | F-CREATE · `does not restore a local-only image as uploaded after refresh`（M）。 | 已有自动化 |
| TC-026 / P2 | C；已选图片 | 用 Tab 聚焦移除按钮并按 Enter/Space；观察素材选择器和槽位。 | 图片被移除；选择器不被意外再次打开；估价/可提交状态随素材更新。 | F-STUDIO · `clears a reference image from the keyboard without reopening the picker`（M）；完整读屏语义另见 TC-105。 | 已有自动化 |
| TC-027 / P0 | E＋D0；A/B 各有图片、另有音频 | A 用 B 的图片引用或音频引用填入图像槽；提交/保存导演准备参数。 | 跨用户或错误媒体类型在扣费前拒绝；钱包、订单、任务不变。 | B-STATE · `TestImageSlotsRejectAudioUploadsBeforeChargeIntegration`；B-TX · `TestDirectImageSubmissionUsesOneTransactionConnectionIntegration`；B-DIRECTOR · `TestDirectorBoardPersistenceAndOwnershipIntegration`（D）。 | 已有自动化 |
| TC-028 / P1 | C＋L0；获准上传测试图片 | 记录钱包/任务；上传许可图片；用所属账号下载字节，随后匿名访问；比较 SHA、MIME 和大小。 | 上传返回持久 ID；所属账号字节一致；匿名拒绝；不增加生成任务或扣费。 | L-UPLOAD · `uploads and reads identical source bytes with authentication and without charging`（L；会真实写上传记录，测试本身不自动删除，需记录并清理批准范围）。 | 已有自动化 |
| TC-029 / P1 | C＋L0；导演准备需上传，存储交付不可用 | 把测试图片与参数保存到镜头；关闭页面后重新打开；读取鉴权预览。 | 准备内容真实持久化；交付存储不可用不阻止只准备镜头；不能因此允许付费生成。 | L-DIRECTOR · `persists real shot material and parameters despite unavailable delivery storage, then restores them from the server`（L；需隔离数据且不创建任务）。 | 已有自动化 |
| TC-030 / P1 | C；上传请求可被中断 | 分别测试超大文件、伪扩展名、上传断网和部分多图成功；重试后核对引用去重及状态。 | 明确指出失败文件；不把失败文件标成可用；可保留成功素材，重试不丢描述、不提交任务；服务端按内容限制而非仅扩展名。 | 现有 L-UPLOAD 仅成功路径和匿名拒绝；缺完整错误/大小/MIME/部分成功 UI＋后端矩阵。 | 待补自动化 |

## 5. 估价与路由（TC-031—040）

目标：估价只计算、选择满足能力的最低成本配置，展示平台售价。共享前置：D0＋F0；UI 用 M0；真实预检使用 L0 且禁止任务创建。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-031 / P0 | E；合法 5 秒参数；财务快照 | 调用 `/client/tasks/estimate`；前后比较余额、账本、订单、任务、task_quotes。 | 返回可用估价；各记录和余额不变；没有 quoteId/锁价/过期凭证。 | B-ESTIMATE-H · `TestEstimateDurationSecHTTPContractHasNoFinancialSideEffectsIntegration`；B-BETA · `TestDirectEstimateThenRecalculateWithoutQuoteIntegration`（D；存储/上游用本地 fixture）。 | 已有自动化 |
| TC-032 / P1 | E；构造敏感内部字段 | 读取目录、选项、估价及错误响应；递归检查 JSON 字段并检查页面；包含合法 prompt 单词 token。 | 不输出供应商、采购成本、执行类型、编码、凭据或上游地址；不误判合法 auth token、用户消费金额或描述文本。 | B-ESTIMATE-H · `TestEstimateDurationSecHTTPContractHasNoFinancialSideEffectsIntegration`、B-ERROR-U · `TestCreationSupplyFailuresHideCredentialAndSupplierDetails`（D/U）；L-CREATE · `estimates three explicit mode/model selections without creating or charging a task`（L）；完整全部 DTO 递归规则须持续维护。 | 已有自动化 |
| TC-033 / P1 | C；延迟/失败估价 fixture | 改参数启动估价；保持等待后失败；点重新估价成功。 | 显示计算中/失败/可用三态；未知价格不显示免费；有可用估价前不能提交。 | F-STUDIO · `distinguishes an estimate in progress or failure from a free price, then retries`；F-CREATE · `keeps submit disabled until an estimate is available`（M）。 | 已有自动化 |
| TC-034 / P1 | C；两次估价逆序响应 | 从短时长切到长时长，让旧响应最后到；检查金额与当前参数。 | 新参数对应估价保留，旧响应不能覆盖；提交使用当前完整参数。 | F-STUDIO · `keeps the new duration price after a slower previous estimate arrives`（M）。 | 已有自动化 |
| TC-035 / P0 | E；多条同模型工作流且成本/priority/ID可控 | 先不同成本选最低；改成同成本比较 priority；再同 priority 比稳定 ID；加入未实现 API。 | 支持的配置按成本、priority、稳定 ID 确定排序；未实现 API 过滤，不能伪装成功；用户指定模型不变。 | B-CATALOG · `TestDirectLowestCostSupportedExecutionAndCapabilityFilteringIntegration`；B-ADAPTER · `TestClientRejectsUnverifiedAPIExecutionAndPollsWorkflow`（D/U）。 | 已有自动化 |
| TC-036 / P1 | E；依次禁用各可用性条件 | 分别改变模式策略、承运商/模型、供应商/绑定、执行健康/额度、成本生效时间和适配器；比较目录/选项/推荐/路由。 | 同一条件在所有入口一致；不可用原因不伪装成可提交；恢复后恢复供给。 | B-BETA · `TestDirectAvailabilityGatesStayConsistentIntegration`、`TestDirectOptionsOnlyAdvertiseBillableCapabilitiesIntegration`（D）；真实凭据和 MinIO 健康另验。 | 已有自动化 |
| TC-037 / P1 | E；凭据只有合法环境 fallback | 清除隔离 supplier 的直接凭据配置；开启受支持 fallback；读取目录和提交预检。 | 目录与提交对凭据判断一致；缺失凭据不能目录可用但提交无说明失败；客户端不泄露 fallback 值。 | B-BETA · `TestDirectAvailabilityGatesStayConsistentIntegration` 的 `environment_credential_fallback`（D，合成凭据）；真实解密/上游认证不在该证据内。 | 已有自动化 |
| TC-038 / P0 | E；同模式不同模型/供应商 | 写入/修改平台售价；切换供应商成本和模型；固定模式、分辨率、自动音频估价。 | 平台售价身份只按模式×分辨率×自动音频及版本；成本变化不能隐式改变零售价。 | B-STATE · `TestRetailPriceInsertAndUpdateRemainModeScopedIntegration`；B-CATALOG · `TestDirectSubmissionBillingRefundAndSnapshotIntegration`（D，部分）；跨供应商/模型价格不变完整对照仍需补测。 | 已有自动化 |
| TC-039 / P0 | E；已有估价，随后改变售价与最低成本供给 | 取得旧估价；更新隔离配置；重新提交完整参数，不传 quoteId。 | 提交按当时新价格/路由；不比较旧估价；一次扣费并保存新快照；失败退款可还原余额。 | B-BETA · `TestDirectEstimateThenRecalculateWithoutQuoteIntegration`（D）；此用例替代错误的“quote 过期凭证”测试。 | 已有自动化 |
| TC-040 / P1 | C＋L0；三模式有明确供给与素材形状 | 每种模式选择目录中的模型与合法参数、选择必要本地图片，观察最终估价；前后读取钱包/任务。 | 三模式分别有明确结果；成功时金额对应选项且无任务/扣费；图片本地预览不当上传成功。无供给时记录阻塞，不伪造价格。 | L-CREATE · `estimates three explicit mode/model selections without creating or charging a task`（L，仅预检）；F-AVAIL · `explains storage unavailability at estimate and restores only the estimate on retry`（M，覆盖存储错误）。 | 已有自动化 |

## 6. 提交、幂等与账务（TC-041—050）

目标：请求身份持久、同一事务一次扣款、可核对退款。共享前置：E＋D0＋F0；浏览器提交仅 M0；真实付费步骤需另行明确预算授权。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-041 / P0 | E；余额足够；合法请求 | 首次提交后读取任务、订单、钱包、账本和 execution/cost/retail 快照。 | 创建一任务、一消费订单、一扣费流水；金额一致，余额精确减少；Worker 所需执行信息已快照。 | B-CATALOG · `TestDirectSubmissionBillingRefundAndSnapshotIntegration`（D）。 | 已有自动化 |
| TC-042 / P0 | E；同 user/request ID | 同一请求顺序重放，再并发发送多次；比较所有返回 ID 与财务数量。 | 所有成功响应返回原任务和订单；只扣一次；不会仅靠按钮禁用实现幂等。 | B-CATALOG · `TestDirectSubmissionRollbackAndConcurrentIdempotencyIntegration`；B-REPLAY · `TestDirectTaskSubmissionIsIdempotentAfterSupplyChanges`（D）。 | 已有自动化 |
| TC-043 / P0 | E；一次成功后关闭供给 | 提交成功；禁用其供给/改能力；同 key 重放。 | 事务先命中旧订单；不再次路由；即使当前供给不可用仍返回原任务/订单、原金额。 | B-REPLAY · `TestDirectTaskSubmissionIsIdempotentAfterSupplyChanges`（D）。 | 已有自动化 |
| TC-044 / P0 | C；首次响应丢失 | 点击提交；让服务器已受理但客户端丢响应；刷新后显式重试；比较 payload 和 Idempotency-Key。 | 保留冻结的请求体和同一请求 ID；不生成新 key；成功后恢复原任务。 | F-CREATE · `replays the same frozen request after a lost response and page refresh`；F-SESSION · `replays a task submission with its original payload and Idempotency-Key`（M）；D 的一次扣费由 TC-042证明。 | 已有自动化 |
| TC-045 / P0 | C；无法写入恢复记录/不完整成功响应 | 拒绝 localStorage 持久化后点提交；另组返回缺 task/order ID 的成功响应。 | 前者不发任务请求；后者保留原请求身份，不能当作明确失败自动生成新 key。 | F-CREATE · `does not send a task when its recovery record cannot be persisted`、`keeps the request identity when a success response is incomplete`（M）。 | 已有自动化 |
| TC-046 / P0 | E/C；估价后余额降低 | 后端用不足余额提交；UI 分别观察预先不足和提交时不足；检查所有财务记录。 | 没有新增任务/订单/流水或部分扣款；留在创作页并说明余额不足，保留素材与描述。 | B-BETA · `TestDirectSubmissionRejectionsLeaveNoFinancialRecordsIntegration`；F-CREATE · `keeps the creator on the form when precharge reports insufficient balance`（D/M）。 | 已有自动化 |
| TC-047 / P0 | E；在事务中注入失败 | 用测试触发器让订单/任务相关写入失败；导演场景再让 board 写入失败；读取事务前后快照。 | 钱包、扣费流水、订单、任务及导演绑定整体回滚，没有孤立任务或已扣费无订单。 | B-CATALOG · `TestDirectSubmissionRollbackAndConcurrentIdempotencyIntegration`；B-DIRECTOR · `TestDirectorTaskBindingAtomicAndIdempotentIntegration`（D）。 | 已有自动化 |
| TC-048 / P0 | E；多小数单价与 5 秒时长 | 按当前金额精度提交并失败退款；比较钱包、订单、扣费/退款流水到分。 | 后端统一舍入；流水加总与余额变化吻合；退款等于原实扣而非重新读取新售价。 | B-CURRENCY-U · `TestRoundCurrencyMatchesWalletPrecision`；B-BETA · `TestDirectEstimateThenRecalculateWithoutQuoteIntegration`；B-DIRECTOR · `TestDirectorTaskBindingAtomicAndIdempotentIntegration`（U/D）。 | 已有自动化 |
| TC-049 / P0 | E/C；排队且允许取消 | 取消一次；模拟响应丢失再取消；另组让 Worker 已受理后再取消。 | 可取消任务只退款一次；已受理任务拒绝普通取消；UI 保留拒绝说明、不先宣称退款成功。 | B-CANCEL · `TestCancelTaskIsIdempotentAfterResponseRetry`；B-STATE · `TestClientTaskActionsMatchCancellationIntegration`；F-AFTERCARE · `only offers a server-approved cancellation and retains a raced rejection in the dialog`（D/M）。 | 已有自动化 |
| TC-050 / P1 | C；钱包读取与兑换/退款重叠；充值未开放 | 同时打开顶栏、创作和钱包；让旧钱包读取最后返回；兑换错误后重试；点击充值入口。 | 三处余额保持最新；读取失败不显示空余额；兑换错误保留；充值明确未开放，无假成功或入账。 | F-SESSION · `an earlier wallet read cannot overwrite a confirmed redemption in any balance surface`、`reconciles overlapping refund and redemption: ${scenario}`；F-AFTERCARE · `keeps wallet errors and a rejected redemption visible without pretending the account is empty`（M）；L-READONLY · `UAT-RO-01 real login, creation states and wallet stay consistent without business writes`（L，只读）。充值文案需人工核对。 | 已有自动化 |

## 7. Worker、租约与恢复（TC-051—060）

目标：提交不确定时不盲目重复调用或退款，旧租约不能覆盖新状态。共享前置：E＋W0＋F0。以下假上游不是真实供应商交付验收。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-051 / P0 | E；空凭据/无法编码请求 | 调用 adapter submit；记录本地假服务器请求计数和错误分类。 | 本地明确未发送的失败可判为未创建；请求计数为 0；不能笼统把所有上游 4xx 也判为未创建。 | B-ADAPTER · `TestSubmitLocalFailuresAreDefiniteAndDoNotSend`（U）。 | 已有自动化 |
| TC-052 / P0 | E；逐项注入 400/408/409/502、损坏 200、业务错误 | 对每个响应调用提交；检查 definite/uncertain 分类，特别是响应含 task ID 时。 | 可能已受理的错误保留不确定性；不能自动生成新提交或承诺退款。 | B-ADAPTER · `TestSubmitClassifiesDefiniteAndUncertainFailures`（U，故障服务器）；真实供应商错误语义需另获证据。 | 已有自动化 |
| TC-053 / P0 | E；上游已受理，本地保存 ID 的 SQL 失败 | 启用测试触发器；运行两次 Runner.Tick；检查上游请求次数、状态、provider_request_id、余额。 | 上游仅收到一次；进入 SUBMIT_UNKNOWN/待核对，保留可恢复 ID；不退款、不再次提交。 | B-WORKER · `TestWorkerSubmissionUncertaintyIntegration`（D＋U，真实 Runner）。 | 已有自动化 |
| TC-054 / P0 | E；Worker A 仅 claim 未开始 submit | 令 A 的租约过期；B 重新 claim；用 A token 写开始/失败；用 B token 继续。 | B 可恢复尚未越过提交边界的任务；旧 token 写入拒绝；无重复扣款。 | B-BETA · `TestDirectSubmissionLeaseCrashRecoveryAndFencingIntegration`（D，手动时间戳）；不是进程崩溃测试。 | 已有自动化 |
| TC-055 / P0 | E；已标记 submit_started，结果未知 | 令租约过期；尝试 reclaim、普通取消、退款和再次 Tick。 | 待核对状态不被普通重试抢走；不自动退款或提交；财务保持原实扣并提供管理恢复入口。 | B-BETA · `TestDirectSubmissionLeaseCrashRecoveryAndFencingIntegration`；B-RECON · `TestProviderCrashRecoveryAndResumeIntegration`（D）；UI 核对见 TC-093。 | 已有自动化 |
| TC-056 / P0 | E；已接受上游任务 | A 获取轮询租约；B 在有效期内尝试，再于过期后尝试；A 用旧 token 完成/退款。 | 活跃轮询只允许一持有者；过期可恢复；旧轮询不能覆盖 B 状态或产生退款。 | B-BETA · `TestDirectPollLeaseAndStorageRecoveryIntegration`（D）；真实多进程调度待 TC-060。 | 已有自动化 |
| TC-057 / P0 | E；任务已有执行/成本快照 | 创建任务后修改工作流编码、成本及路由顺序；运行 Runner 并捕获提交体。 | 使用原任务快照中的执行和输入，不重选当前最低价；成本快照不变。 | B-WORKER · `TestWorkerSubmissionUncertaintyIntegration` 的 snapshot/failed-poll 场景（D＋U）；B-CATALOG · `TestDirectSubmissionBillingRefundAndSnapshotIntegration`（D）。 | 已有自动化 |
| TC-058 / P0 | E；上游失败或 deadline 已过 | 分别让轮询明确失败、RUNNING 超时、WAITING_STORAGE 超时；连续 Tick 两次；核对退款。 | 正确终态与失败原因持久化；退款一次；终态不再调用上游；退款取原订单金额。 | B-WORKER · `TestWorkerSubmissionUncertaintyIntegration` 的 failed poll、RUNNING/WAITING_STORAGE timeout 场景（D＋U）。 | 已有自动化 |
| TC-059 / P1 | E；上游完成而结果存储暂不可用 | 记录交付等待；再次轮询/交付并恢复存储；检查结果 URL、状态与晚到失败。 | 暂无自有结果不显示已交付；保留上游结果供恢复；完成后晚到失败不能退款或覆盖。 | B-BETA · `TestDirectPollLeaseAndStorageRecoveryIntegration`（D，合成 OSS URL）；真实复制失败/恢复需 O 环境，不能以本例代替。 | 已有自动化 |
| TC-060 / P0 | E；独立后端进程、故障服务器、隔离 PG | 在 claim 后、发送前、受理后保存前、轮询中分别终止进程；启动新进程；释放迟到响应。 | 各边界与 TC-054—059 一致；submit 未知不二次调用；旧进程/租约结果不覆盖新状态；记录真实 PID、时间及请求计数。 | 现有测试仅手工租约时间和 Runner.Tick，无真实 kill/restart/优雅退出测试。仅允许隔离进程；不得停止 UAT。 | 待补自动化 |

## 8. 任务详情、下载与售后（TC-061—068）

目标：状态、退款和交付真实可理解；下载必须验证文件而非 HTTP 200。共享前置：C＋M0；真实检查优先读取账号自己的已有任务。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-061 / P1 | C；排队、执行、待核对、失败未退款/已退款 fixture | 逐一打开详情；检查状态、失败原因、可取消性、退款金额/时间；注入内部诊断。 | 各状态语义明确；未有退款证据不宣称已退款；待核对不承诺自动重试；内部供应商诊断不展示。 | B-ERROR-U · `TestClientFailureReasonRequiresRefundEvidenceAndHidesProviderErrors`、`TestClientReviewRequiredDoesNotPromiseAutomaticRetry`；F-AFTERCARE · `explains a submission under review and carries its task to support without a resubmit action`（U/M）。 | 已有自动化 |
| TC-062 / P1 | C；ready URL 与不可播放媒体各一组 | 从任务列表进入详情；等待元数据；让浏览器播放失败，再检查下载/保存操作。 | 只有交付 URL 才显示实际播放器；失败有说明但仍保留合法下载与保存；不把参考图模拟成视频。 | F-STUDIO · `uses only a delivered result for video preview and reports media errors`；F-JOURNEY · `refreshes delivery state and preserves the download action after a media error`（M，参考媒体）。 | 已有自动化 |
| TC-063 / P1 | C；待交付/访问权限撤销 | 打开未完成任务；随后让详情返回 403/404；观察轮询并显式重试。 | 未完成不签结果、不暴露旧 URL；失去访问权后停止无休止轮询；有明确重试入口。 | B-DELIVERY-U · `TestPendingTaskDoesNotSignOrExposeResult`；F-DELIVERY · `stops polling after a task becomes inaccessible and allows an explicit retry`（U/M）。 | 已有自动化 |
| TC-064 / P1 | C；已有成功任务，下载服务可用 | 从详情及结果入口点击“下载成片”；捕获 download；核对文件名、MIME、长度并打开播放。 | 发起附件下载而非仅打开预览；真正产生非空文件；浏览器视频宽高/时长有效或 ffprobe 能解码。 | F-AFTERCARE · `downloads an attachment instead of opening the preview from the ${entry}`（M，仅附件流程）；B-DELIVERY-H · `TestDeliveryURLHandlerIntegration`（D＋签名 fixture，不下载字节）。真实可播放步骤需 L/O/H，未由这些测试覆盖。 | 待执行 |
| TC-065 / P0 | C/C2/G；A 有任务与资产 | A 获取自己的下载；B 与匿名请求同 ID；观察是否先查归属再签名。 | A 获授权链接；B/匿名拒绝，不泄露 URL；资产下载始终经过归属检查。 | B-DELIVERY-H · `TestDeliveryURLHandlerIntegration`；B-DELIVERY-U · `TestAssetDTOKeepsDownloadBehindOwnershipCheck`（D/U）。 | 已有自动化 |
| TC-066 / P1 | C；持久记录有过期签名，签名器先失败再恢复 | 读取详情/资产列表；失败后重试；对比数据库与返回值。 | 生成新短期链接，不回传过期存储签名；签名失败明确 503/可重试且不泄露诊断；预览不被附件链接覆盖。 | B-DELIVERY-H · `TestDeliveryURLHandlerIntegration`；B-DELIVERY-U · `TestTaskDeliveryFailureNeverReturnsStoredSignature`、`TestTaskDeliveryReplacesExpiredURLWithoutChangingTask`；F-AFTERCARE · `retains an attachment preparation failure and lets the creator retry`（D/U/M）。 | 已有自动化 |
| TC-067 / P1 | C；待核对任务与真实工单详情 | 从任务点帮助；输入问题并让提交失败；重试后打开工单。 | 保留 task 上下文及用户文本；错误不清空；工单详情包含原问题；无未经确认的重新生成动作。 | F-AFTERCARE · `shows a real ticket detail and preserves task context and input after submission fails`（名字含 real，但该文件是 M）；B-CLIENT-U · `TestClientTicketDTOIncludesOriginalQuestionForDetails`（U）。真实工单写入另需 L 授权。 | 已有自动化 |
| TC-068 / P0 | C＋L0/O/P；三条任务预算已明确确认 | 按批准的三模式参数上传必要素材、估价、各提交一次；等待终态；下载解码、保存并对账。 | 当前版本各任务只有一次扣费；成功有真实视频、自有存储和持久保存；失败有正确退款或待核对，不能算成功交付；记录所有任务/订单证据。 | 无自动化可替代真实供应商验收。L-CREATE 仅估价；L-UPLOAD 仅图片上传。未确认预算时不执行、不得重试产生额外费用。 | 待执行 |

## 9. 作品、资产与回收站（TC-069—074）

目标：明确任务与保存作品的关系，刷新可定位，失败不丢输入。共享前置：C＋M0；真实写入只用 L0 明确独立目录。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-069 / P1 | C；成功任务，多个作品目录 | 保存时选定作品/目录；刷新任务详情；点“查看保存位置”。 | 返回的 savedDestination 指向真实目录；刷新后不丢；URL 可重建位置，而非依赖内存状态或 Toast。 | F-AFTERCARE · `opens the saved project folder after reloading task detail`；B-STATE · `TestClientTaskSavedDestinationSurvivesReloadIntegration`（M/D）。 | 已有自动化 |
| TC-070 / P0 | E＋D0；A/B、已交付与未交付任务 | 并发保存同任务；再尝试保存未交付/他人任务。 | 合法并发只产生一次资产；非法归属/未完成拒绝；原任务不被篡改。 | B-CATALOG · `TestSaveCompletedTaskToProjectIsIdempotentIntegration`（D，结果 URL 为 fixture，不证明真实文件存在）。 | 已有自动化 |
| TC-071 / P1 | C；创建作品/保存请求先失败 | 输入作品名后提交；失败不关闭；恢复后重试并进入新作品。 | 名称/选择保留，错误可见；成功以持久 URL 打开；只靠 Toast 不算完成。 | F-DELIVERY · `keeps a new project name after failure and opens the successful project by a durable URL`；F-JOURNEY · `continues a public tutorial through login, submission, result and a chosen project`（M）。 | 已有自动化 |
| TC-072 / P1 | C；隔离文件；删除先失败 | 打开删除确认并提交；返回错误；再次确认成功；观察目录刷新。 | 失败保留文件及确认框和原因；成功后才移除对应条目；不误删相邻文件。 | F-DELIVERY · `keeps the file and deletion confirmation after failure, then refreshes the directory on success`（M）；真实删除/恢复另验 TC-074。 | 已有自动化 |
| TC-073 / P1 | C；资产视频不可播放但来源任务存在 | 打开资产预览；触发媒体错误；下载；返回来源任务。 | 媒体错误不吞掉下载/任务入口；下载仍走受保护路径；来源关系清楚。 | F-DELIVERY · `preserves a protected download and source-task navigation when asset media cannot be played`（M）；B-DELIVERY-H · `TestDeliveryURLHandlerIntegration`（D，资产归属/链接子场景）。 | 已有自动化 |
| TC-074 / P0 | C/C2＋L0；专用作品与素材，无其他用户数据 | 搜索测试作品/文件；移入回收站；刷新；恢复到原目录；确认后永久删除本轮数据；B 尝试相同对象操作。 | 搜索/目录和回收站状态真实一致；恢复不丢归属/来源；永久删除仅目标范围；越权失败；删除结果及清理清单可复核。 | 当前用例库未找到完整搜索→回收→恢复→永久删除的真实 API 自动化；需补目录、权限与持久化断言，不拿 F-DELIVERY Mock 当证据。 | 待补自动化 |

## 10. 导演台与单镜头链路（TC-075—086）

目标：借鉴 Toonflow 的分镜组织方式，但继续使用 Sub2Video 的创作、任务和财务合同；导演准备、生成任务、候选成片和选择结果保持明确关系。共享前置：C/C2＋M0 或 D0；L-DIRECTOR 仅受控准备写入，不付费生成。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-075 / P1 | C；自己的作品，无分镜 | 从作品进入导演台；新增多个镜头并排序、编辑；保存后刷新。 | 镜头名称、顺序和参数持久化；revision 更新；空工作区有明确下一步。 | F-DIRECTOR · `enters from a project, saves ordered shots and restores them on refresh`（M）；B-DIRECTOR · `TestDirectorBoardPersistenceAndOwnershipIntegration`（D）。 | 已有自动化 |
| TC-076 / P0 | C；两页读取同 revision | 页面 A 保存；B 带旧 revision 保存；确认 B 草稿仍在，再明确放弃/读取最新后重做。 | B 收到 409，不覆盖 A；不自动丢 B 编辑；只有明确放弃后才替换本地草稿。 | F-DIRECTOR · `keeps unsaved edits on conflict and reloads only after an explicit discard decision`；L-DIRECTOR · `rejects a stale real board revision without losing edits or replacing the other page save`（M/L）；B-DIRECTOR 同步覆盖并发 PUT（D）。 | 已有自动化 |
| TC-077 / P0 | C；普通创作已有草稿 | 从镜头进入创作；先不确认应用，再明确应用；检查提交上下文。 | 原草稿未被擅自替换；确认后才带 project/shot/revision；打开或准备不自动扣费。 | F-DIRECTOR · `requires explicit application before replacing the normal draft, then submits with director context`（M）。 | 已有自动化 |
| TC-078 / P1 | C；余额为 0 或估价/交付存储不可用 | 编辑镜头描述、素材和参数；保存准备；刷新再打开。 | 准备可独立保存；保留其他镜头和候选；未取得可用估价仍不能生成；无新任务或扣费。 | F-DIRECTOR · `saves shot preparation without balance or an available estimate and preserves unrelated shots and candidates`（M）；L-DIRECTOR · `persists real shot material and parameters despite unavailable delivery storage, then restores them from the server`（L）。 | 已有自动化 |
| TC-079 / P1 | C；准备保存期间 revision 冲突 | 保存准备返回 409；保持编辑；主动读取最新并重新应用。 | 不因冲突丢准备或覆盖最新 board；需重新明确应用后才继续提交。 | F-DIRECTOR · `keeps preparation on a save conflict until the latest board is explicitly read and reapplied`、`requires another explicit application after a rejected board revision and retains shot edits`（M）。 | 已有自动化 |
| TC-080 / P1 | C；自己已有运行中、失败和已交付任务 | 打开任务选择器，关联已有任务；选择候选成片；刷新；重复关联同任务。 | 关联不会新建任务或扣费；仅已交付、有结果的候选可选中；选中结果持久；重复关联不重复追加或增加 revision。 | F-DIRECTOR · `links an existing task through the account task picker without creating a task`、`selects only a delivered candidate and preserves the choice after refresh`（M）；B-DIRECTOR-H · `TestDirectorBoardHandlerIntegration`（D）。 | 已有自动化 |
| TC-081 / P0 | C/C2/E；他人素材、任务、作品及内部字段 | 用 B 的对象 ID 更新 A board/关联候选；在 JSON 注入私有执行字段；读取非法 ID。 | 明确拒绝且不泄露他人信息；客户端不接收/输出供应商内部字段；合法描述含普通 token 文本不误报。 | B-DIRECTOR-H · `TestDirectorBoardHandlerIntegration`；B-DIRECTOR-U · `TestDirectorRequestRejectsInvalidOrPrivateFields`；B-DIRECTOR · `TestDirectorBoardPersistenceAndOwnershipIntegration`（D/U）。 | 已有自动化 |
| TC-082 / P0 | E＋D0/F0；相同 revision | 同 key 并发提交镜头；再用不同 key 同 revision 竞争；注入 board 写入失败。 | 同 key 一任务一扣款；不同 key 只一合法 revision 获胜；失败事务全回滚；绑定候选与订单保持一致。 | B-DIRECTOR · `TestDirectorTaskBindingAtomicAndIdempotentIntegration`（D）。 | 已有自动化 |
| TC-083 / P0 | C/E；请求已受理但响应丢失 | 保存冻结请求后移除镜头/关闭供给；刷新并用同请求 ID 重放；另测首次请求对应作品已删除。 | 已成功的重放返回原任务/订单；未成功且作品确定删除则清理待提交记录，不困在永远重试状态。 | F-DIRECTOR · `replays the frozen director request after a lost response even if the shot was removed`、`clears a director request that is definitively rejected because its project was removed`（M）；B-DIRECTOR · `TestDirectorTaskBindingAtomicAndIdempotentIntegration`（D）。 | 已有自动化 |
| TC-084 / P1 | C；镜头已有候选任务 | 删除镜头并保存；检查任务列表与账本；导出剩余镜头清单。 | 删除镜头不取消/删除生成任务、不改账；导出顺序和选择正确，不含上游/媒体签名 URL，导出不生成任务。 | F-DIRECTOR · `removes a shot without cancelling or deleting its tasks`、`exports a shot checklist without media URLs and does not trigger generation`（M）。 | 已有自动化 |
| TC-085 / P1 | C；首尾帧、多图镜头已保存真实上传引用 | 退出当前页重新打开 board；分别恢复每个模式到创作；读取受保护素材预览。 | 引用按槽位恢复，鉴权预览可读；非本地 blob；无丢图或自动重建新任务。 | F-DIRECTOR · `restores ${mode} upload references from the server board with authenticated previews`（M）；L-DIRECTOR 持久准备测试（L，仅其实际素材组合）。 | 已有自动化 |
| TC-086 / P0 | C/C2；A 有未保存导演编辑 | 在脏 board 状态退出；B 登录并进入自己的作品；再尝试浏览器返回。 | 退出可完成，不被未保存确认困住；A 内容不在 B 视图/缓存中泄露；不误保存 A 草稿到 B。 | F-DIRECTOR · `lets logout leave a dirty board without exposing the former account contents`（M）；服务端对象隔离由 TC-081补证。 | 已有自动化 |

## 11. 管理配置与审计（TC-087—094）

目标：操作方便但不损伤已有配置，关键操作有明确确认与审计。共享前置：A＋M0；真实配置 CRUD 仅隔离测试环境，不能修改 UAT 供给/价格来构造场景。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-087 / P1 | A；多承运商和多个模型；价格 API 故障 | 进入承运商列表、展开所属模型、切页、编辑；禁用/激活时先取消再确认；检查审计。 | 价格错误不使目录整体消失；子列表归属正确；编辑为弹窗；取消无写入，确认才修改；审计记录操作者和对象。 | F-CATALOG · `keeps the carrier catalog visible when pricing data is temporarily unavailable`（M，部分）；尚缺承运商/模型完整 CRUD＋确认＋审计的真实端到端矩阵。 | 待补自动化 |
| TC-088 / P1 | A；供应商与执行配置超过一页 | 依次打开供应商/API/Workflow 标签；按承运商和模型筛选；取后页供应商创建表单；查看 degraded 标记。 | 三标签可达；服务端分页拿到后页关系；筛选作用于正确配置；异常健康状态不显示可用。 | F-SUPPLIER · `keeps supplier, API, and Workflow surfaces available`；F-ADMIN · `new executions can select suppliers beyond the first page and degraded is truthful`、`supplier relationship pagination retrieves later executions from the server`（M）。 | 已有自动化 |
| TC-089 / P0 | A；多分辨率/音频成本不同 | 编辑执行名称等元数据并保存；再只修改一个成本维度；比较所有其他维度。 | 元数据更新不重写成本；单维成本保存只改该维度，不统一覆盖不同价格。 | F-ADMIN · `editing execution metadata preserves different cost dimensions`、`editing one cost saves only that resolution and audio dimension`（M）；真实 DB 更新另由供给事务测试补证，不能从请求体推导已存储。 | 已有自动化 |
| TC-090 / P1 | A；成本加载失败及部分保存失败 | 加载失败时尝试保存；重试加载；提交多个维度让其中一个失败；再次保存。 | 未加载原始成本前阻止覆盖；失败保留弹窗；重试只保存未成功维度并保留金额。 | F-ADMIN · `cost loading failure blocks save and retry restores original amounts`、`partial cost failure stays open and retries only unsaved dimensions`（M）。 | 已有自动化 |
| TC-091 / P0 | A；已有平台售价 | 编辑售价；尝试修改身份维度和空金额；保存合法金额后读取数据库。 | 编辑锁定原规则身份；空金额不能变为 0；售价仍为 mode/resolution/audio 维度，不绑模型或供应商。 | F-ADMIN · `retail edit locks identity and rejects an empty amount`（M）；B-STATE · `TestRetailPriceInsertAndUpdateRemainModeScopedIntegration`（D）。 | 已有自动化 |
| TC-092 / P1 | A；跨页任务；两次搜索响应逆序 | 搜索并翻页；快速输入新查询让旧结果晚到；注入加载/取消失败。 | 后端搜索覆盖全部匹配记录；新结果不被旧覆盖；错误不是空列表；已受理任务无普通取消，竞态拒绝留在确认框。 | F-ADMIN · `tasks paginate and search on the server; accepted tasks cannot be ordinarily cancelled`、`stale task search cannot replace newer results and load errors remain actionable`、`rejected cancellation keeps its confirmation open with the server reason`（M）；B-STATE · `TestAdminTaskSearchPaginatesAcrossAllMatchingRowsIntegration`（D）。 | 已有自动化 |
| TC-093 / P0 | A/E；SUBMIT_UNKNOWN；有可验证证据 | 分别提交已受理 ID 恢复轮询和明确未受理证据退款；先让核对失败，再重试；对已知 accepted ID 尝试“未受理退款”；检查审计及幂等。 | 必须显式确认和验证；失败保留证据；已接受 ID 固定且不能伪装未受理；恢复/退款一次、审计原子；不重新生成。 | F-RECON · `requires a verified upstream task and explicit confirmation before resuming`、`requires evidence of nonacceptance before refund and does not submit another task`、`keeps a captured provider task id fixed and offers no unaccepted refund`（M）；B-RECON · `TestProviderCrashRecoveryAndResumeIntegration`、`TestProviderReconciliationRefundAtomicAndIdempotentIntegration`、`TestProviderKnownAcceptedIDCannotBeRefundedAsUnacceptedIntegration`；B-RECON-H · `TestTaskReconciliationHandlerIntegration`（D＋假上游）。 | 已有自动化 |
| TC-094 / P0 | A/E；供应商/执行写入中途失败 | 在隔离配置创建/更新时注入失败；核对执行、绑定、成本及审计；再成功写入并由另一管理员读取。 | 失败不留下半套配置；成功审计记录准确、不含明文凭据；普通创作者不能读取配置或审计。 | B-SUPPLIER · `TestSupplierOfferingCreateRollbackIntegration`、`TestSupplierOfferingCreateUpdateAndRollbackIntegration`（D，覆盖供给原子性）；所有配置类型审计/授权完整矩阵未覆盖，需要补增量测试。 | 待补自动化 |

## 12. 布局、视觉与无障碍（TC-095—106）

目标：真实可操作而非仅截图存在。共享前置：G/C/A＋M0/V0；每个状态标记 Mock 或真实 API；量化标准直接使用 [UI 验收标准](./ui-layout-acceptance.md) 的 ID。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-095 / P1 | G/C/A；六宽度 | 按 L01/L02/L04/L08/L09 进入官网、创作、概览、资产、导演和承运商；测 frame；横滚局部表格。 | 页面不横向溢出；扣除侧栏后的内容轴线平衡；大屏利用率合理；最右操作列可达而不是七列挤竖排。 | F-LAYOUT · `public content shares balanced axes at ${width}px`、`workspace remains usable and balanced at ${width}px`；F-UI · `primary actions and admin table stay usable at ${width}px`；F-DIRECTOR · `renders the shot workspace across phone, tablet and desktop widths`（M）。未覆盖全部管理表格。 | 已有自动化 |
| TC-096 / P2 | C/A；可调实际内容容器 | 按 L03/L05/L06/L07 精确量测 gutter、header/sidebar；令实际创作容器处于断点两侧；检查输入/预览列。 | 符合设计合同尺寸及内容容器断点；不能按 viewport 猜列数，不能把表单强制等宽或以隐藏内容掩盖溢出。 | 六视口 F-LAYOUT 仅抽样；精确容器边界、所有 gutter/header/列宽未系统自动断言。 | 待补自动化 |
| TC-097 / P1 | C/A；六宽度、底部导航可见 | 按 L10/L11 自然滚动到提交、新建作品、管理新增与确认按钮；trial click 和多点 hit-test。 | 主要触摸操作尺寸合规且完整可达；不被底栏遮挡；禁止 force click 掩盖问题；不实际提交。 | F-UI · `primary actions and admin table stay usable at ${width}px`（M）；仅已列主要按钮，不代表全站按钮。 | 已有自动化 |
| TC-098 / P1 | C/A；320×640 和 1024×900 | 以 Enter 打开默认/宽弹窗；Tab/Shift+Tab 循环；按 Escape；检查焦点。 | 按 L12/K01/K02 弹窗在视口、内部可滚动；焦点被正确约束并归还触发器；不能只检查可见。 | F-UI · `${surface.name} dialog fits, traps focus and restores its trigger at ${width}px`（client/admin，M）；嵌套弹窗、长表单和软键盘另验。 | 已有自动化 |
| TC-099 / P1 | C/A；320/390px | 打开 drawer；聚焦链接后 Escape；重开并导航；检查遮罩、菜单及目的页操作。 | 按 K03 两种关闭方式都移除抽屉/遮罩；目标 URL/标题正确且可操作；不残留桌面侧栏占位。 | F-UI · `${surface.name} mobile navigation closes on Escape and navigation at ${width}px`（M）；完整读屏/焦点语义仍需人工。 | 已有自动化 |
| TC-100 / P2 | C/A；两个表面及 portal | 检查 main、弹窗背景、圆角、控件 recipe；审查本轮页面/feature 差异；若可达则在同 SPA 切换表面。 | 按 V01/V02/V03 继承正确；没有第二套颜色或任意视觉样式；portal 不继承错主题。 | F-UI 弹窗动态测试（M，独立页面加载）；`pnpm lint:check`＋人工 diff。当前 SPA 跨表面导航未覆盖，不伪造内部路由调用当证据。 | 已有自动化 |
| TC-101 / P2 | E；固定字体/fixture，预期变更清单 | 运行视觉回归；查看原图、新图和差异；额外审平板与宽屏；仅对已解释变化更新基线。 | 按 V04 能说明每个基线改动；不提升容差掩盖漂移；截图上的内容层次符合业务状态。 | F-AUTH-V · `${surface.name} uses the commercial film entrance`；F-JOURNEY-V · `${surface.name} retains the commercial journey baseline`；F-SURFACE-V · `${item.name} matches the surface baseline`（V）；通过仍需人工审图记录。 | 已有自动化 |
| TC-102 / P1 | C/A；隔离长文本 fixture | 按 A03 准备业务允许的长中文名、连续 ASCII 和长描述；显示于卡片、表格、错误、弹窗和导演镜头。 | 操作不被覆盖；全页不溢出；截断内容有可获得完整文本；不通过非法超限数据误报 UI。 | 当前只有常规 fixture，无系统长文本矩阵；需补各主要表面参数化用例。 | 待补自动化 |
| TC-103 / P1 | G/C/A＋真实桌面浏览器 | 用浏览器菜单设 200% 缩放；检查认证、创作、导演、钱包和管理弹窗；键盘完成主要操作后恢复缩放。 | 按 A01/A02 信息和操作保留，可自然滚动；CSS zoom/deviceScaleFactor/截图放大不能作为等价证据。 | 无真实浏览器缩放自动化；需 H，记录缩放及 innerWidth/截图。 | 待执行 |
| TC-104 / P1 | G/C/A；明暗两表面和媒体覆盖层 | 按 A05 量测正文、辅助文字、错误、按钮、focus、边界的最终渲染对比；检查 hover/focus 状态。 | 满足对应 WCAG 阈值或有明确适用例外；状态不只依赖颜色；不能只比较 token 原值。 | 尚无完整对比度审计；自动扫描可辅助，但透明度、图像背景与语义需要 H。 | 待执行 |
| TC-105 / P1 | G/C/A；键盘＋屏幕阅读器 | 按 K04/A07 从头遍历模式、素材、参数、估价、任务状态、弹窗、分页、FAQ；启用系统字体放大。 | 名称/角色/状态与错误可被理解；焦点可见，无无法退出陷阱；状态更新不丢；不为装饰图标添加噪声。 | F-PUB FAQ、F-STUDIO 图片移除、F-UI 弹窗有局部 M；完整读屏和字体放大未覆盖，需 H。 | 待执行 |
| TC-106 / P1 | C/A；真实 Android/iOS 设备 | 按 A06 打开登录密码、创作描述及管理弹窗；弹软键盘、旋转、滚至确认/提交；检查刘海/底部安全区。 | 字段、错误和操作始终可达；底栏/键盘不遮挡；触摸滚动和上传选择可用。 | 无真机证据；桌面 Chromium 390px 不算移动 Safari/Chrome 或软键盘验收。 | 待执行 |

## 13. 发布、兼容与恢复（TC-107—112）

目标：测试确实对应本次候选；迁移、备份和回滚不损伤用户数据。共享前置：E；所有实际发布操作需要 R 的独立授权，本用例库只定义方法和验收条件。

| ID / 优先级 | 角色与补充前置 | 可操作步骤 | 可观察预期 | 精确自动化映射、证据边界 | 准备状态 |
| --- | --- | --- | --- | --- | --- |
| TC-107 / P1 | E；当前源码与待验环境 | 记录分支/HEAD/脏树摘要、工具版本与产物 SHA；核对浏览器入口及加载资源；按约定执行静态、Mock、视觉和集成命令。 | 相同 commit 但不同工作区产物不能混同；每条结果绑定版本；skip 单列；旧 UAT 成功不能代替当前源码验收。 | `pnpm lint:check`、`pnpm typecheck`、`pnpm build`、`go test -count=1 ./...`、`pnpm test:visual`、`git diff --check`（各自作用域）；没有单个命令证明全链路。 | 待执行 |
| TC-108 / P0 | E＋D0；空 schema、Go 工具链 | 编译当前后端；以无效 Redis/listen 地址和禁止创建的上传目录执行 `--migrate-only` 两次；再传非法 CLI 参数。 | 仅迁移数据库；所有 SQL 的版本名与源码 SHA 一致、无多余记录；无用户/运行时 seed、HTTP/Redis/Worker/上传目录副作用；非法参数连接数据库前失败。 | B-MIGRATE · `TestMigrateOnlyCLIHasNoServiceSideEffectsIntegration`（D＋真实 CLI；本轮改为动态迁移集合，不再固定 21）。 | 已有自动化 |
| TC-109 / P0 | E＋D0；019 合成历史状态 | 建 019 schema 和已有财务/任务/配置；应用全部迁移两次；比较快照；篡改一个已应用 checksum 再试。 | 历史余额/订单/任务/管理员价格不变；020/021/022 结构到位；旧作品导演板为空且 revision 初始正确；checksum 不匹配必须拒绝，不回放历史清理 SQL。 | B-UPGRADE · `TestUpgradeFrom019PreservesExistingDataIntegration`；B-CATALOG · `TestDirectCatalogMigrationAndSeedIsolationIntegration`（D）。 | 已有自动化 |
| TC-110 / P1 | E；正式路由表与旧客户端请求 | 调用旧 quote；检查旧 workflow/supply/pricing 客户端入口；运行新 estimate/create 后比对 task_quotes。 | 旧 quote 明确 410；正式路由不暴露旧执行管理合同；新流程不读写 quote，不重新引入兼容锁价。 | B-CLIENT-U · `TestLegacyQuoteEndpointIsExplicitlyDisabled`；`backend/internal/server/router_contract_test.go` · `TestRouterDoesNotExposeLegacyWorkflowSupplyOrPricingEndpoints`（U）；B-BETA 估价再提交测试（D）。 | 已有自动化 |
| TC-111 / P0 | R/E；独立恢复演练环境和明确产物清单 | 验证备份 SHA 与 pg_restore --list；恢复到隔离库核对数据；演练旧前端入口/hashed assets保留；验证二进制与迁移兼容后演练回滚。 | 备份不仅文件存在而且可恢复；旧打开页面懒加载可用；新订单/账本不得被旧 dump 覆盖；二进制回滚不冒充数据库回滚；失败日志保留。 | 当前无覆盖全流程的自动化；需受控演练和清单。检查恢复不允许在 UAT 覆盖数据，也不下载含业务数据的 dump 入仓库。 | 待执行 |
| TC-112 / P0 | R/E；候选启动失败/进行中任务/未知提交的演练状态 | 发布前读任务状态；分别模拟 hash 错、备份失败、迁移异常、后端启动失败；验证停止条件和恢复步骤。 | 任一关键异常立即停止，不能猜测重试；未知提交不重新生成或直接退款；未确认兼容不自动起旧后端；服务启动后仍需业务验收。 | B-RECON · `TestProviderCrashRecoveryAndResumeIntegration`、B-WORKER · `TestWorkerSubmissionUncertaintyIntegration` 仅覆盖业务状态机；完整发布编排与真实进程边界尚无自动化，需隔离演练。线上发布必须另获授权。 | 待执行 |

## 执行顺序、环境开关与记录门槛

1. 先核对版本、规则及修改范围（TC-107），运行静态检查、单元测试与 Mock 用例。不得为了测试修改 UAT 真实供给、余额或资产。
2. 启动已确认隔离的 D0/W0，优先执行 P0 财务、幂等、归属、导演 revision 和 Worker 不确定边界；随后跑迁移。主运行器同时检查实际测试数、skip 和 schema/Redis 清理结果。
3. 执行 Mock UI 状态、六宽度几何和视觉回归；人工审图并记录未覆盖的管理页/长文本/真机等事项。
4. 执行 L0 只读检查；再执行明确获准的上传、导演准备保存与冲突。真实写入数据前缀与清理路径先登记。不要直接把 `live-*` 文件名视为安全只读，逐个看真实请求白名单。
5. 真实对象存储和供应商分别验收。付费生成与发布演练要满足各自前置授权；缺环境就记录未验证及具体缺失条件，不以 Mock 代替。

后端在 `backend/` 执行基础验证：

```text
go test -count=1 ./...
```

真实数据库集成需先配置 `SUB2VIDEO_RUN_INTEGRATION=1`、`SUB2VIDEO_TEST_DATABASE_URL`；Store 公共 helper 另需 `SUB2VIDEO_TEST_REDIS_URL`。以下命令只有在 D0 已核实后运行：

```text
go test -count=1 -v ./internal/store ./internal/handler ./internal/worker ./migrations
go test -count=1 -v ./cmd/server -run '^TestMigrateOnlyCLIHasNoServiceSideEffectsIntegration$'
```

CLI 测试内部会 `go build`，仅预编译测试二进制而无 Go 工具链的容器无法完成。公共 Store helper 的 Redis 目前只是随机 key 的 SET/GET/DEL 探针；不能据此宣布队列、验证码或所有缓存已集成验收。部分测试基础设施不可用会 Skip，退出码为 0 也必须检查具体结果。旧个别测试清理错误未显式上报，主运行器应独立核对遗留随机 schema，而不能执行广域清理。

Store fixture 默认预算保持 60 秒；共享迁移、18 个可用性 gate 及凭据 fallback 的 `TestDirectAvailabilityGatesStayConsistentIntegration` 单独使用 3 分钟预算，gate 还原使用独立 15 秒清理上下文。这是测试环境预算，不能改变生产执行 deadline、弱化业务断言或把一次超时重跑记成首次通过。

真实 MinIO 用例还需 `SUB2VIDEO_TEST_MINIO_ENDPOINT`、`SUB2VIDEO_TEST_MINIO_ACCESS_KEY`、`SUB2VIDEO_TEST_MINIO_SECRET_KEY`、`SUB2VIDEO_TEST_MINIO_BUCKET`（名含 test）、`SUB2VIDEO_TEST_MEDIA_FILE`；SSL 根据隔离环境配置。只删除本次随机对象。该测试验证字节一致和签名访问，不做视频解码：

```text
go test -count=1 -v ./internal/storage/minio -run '^TestRealObjectStorageMediaRoundTripIntegration$'
```

前端从 `frontend/` 执行既有命令，Playwright 配置、报告位置和服务器生命周期由协调者统一控制：

```text
pnpm lint:check
pnpm typecheck
pnpm build
pnpm exec playwright test tests/e2e/creation.contract.spec.ts tests/e2e/director.contract.spec.ts tests/e2e/create-availability.contract.spec.ts
pnpm exec playwright test tests/e2e/ui-standards.contract.spec.ts --project=desktop
pnpm test:visual
```

`live-*` 运行前阅读对应配置和环境变量的真实用法，配置专用账号；凭据只进入进程环境，不能写入本文、trace 或报告。上传/导演准备的真实写入与 `uat-readonly.spec.ts` 分开运行和汇报。

## 出口条件与结果表模板

封闭 Beta 判断需列出实际通过的用例 ID、已知问题、未验证风险及限制。所有 P0 必须有与风险对应的真实数据/权限证据；核心路径 P1 不能仍然阻断操作。Mock、视觉、D、L、O、P、H 分别报告，不因一个层级通过推导另一个层级通过。没有真实交付和真机证据时应明确说明可试用范围，不称已完成全部商业验收。

| 执行批次/版本 | 用例 ID | 环境与证据类型 | 实际结果（通过/失败/未验证） | 命令/断言/脱敏证据 | skip/问题及下一步 | 数据与清理 |
| --- | --- | --- | --- | --- | --- | --- |
| 由本轮验收记录填写 | TC-... | M/V/U/D/L/O/P/H | 不在本用例库预填 | 不含 Token、密码、连接串 | 明确最小缺失条件 | 精确对象/路径，保留失败证据 |

本文未新增生产代码、业务数据或测试运行产物；不包含本轮运行通过断言。后续增加功能或改变合同，应先更新相应用例步骤和证据映射，再更新独立执行记录。
