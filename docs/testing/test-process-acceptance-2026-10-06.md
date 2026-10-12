# 测试流程建设与执行记录（2026-10-06）

## 结果与范围

已建立 [测试入口](README.md)、[分层执行计划](test-plan-2026-10-06.md)、[112条业务用例](test-cases-2026-10-06.md) 和 [31项UI标准](ui-layout-acceptance.md)，并实际运行本地UI、视觉、Go和隔离真实数据库测试。没有把Mock、数据库和真实供应商混为一类证据。

本轮修复创作错误表达、草稿模型恢复、手机主要按钮尺寸及后端无供给错误映射。没有扩展支付、模型、供应商或影片工程；没有再次重置UAT数据库。本轮最终真实UAT结果仍为失败：107的旧后端将无可推荐模型映射为500，且供应商与对象存储配置未恢复，因此不能把当前107称为可生成的封闭Beta。

## 版本和环境

| 项目 | 本轮事实 |
| --- | --- |
| 分支/HEAD | `codex/commercial-studio-redesign` / `0cafa8d612f4ad8dcebcff58d7d25d674ebd5945` |
| 工作区 | 保留已有导演台及其测试、文档、迁移等未提交修改；本轮增量见下表，HEAD不是当前完整源码版本 |
| 本地工具 | Windows；Node22.16.0、pnpm10.32.0、Go1.25.1 windows/amd64；Vite6.4.3；Playwright1.62.1 |
| UI来源 | 当前Vue源码，Playwright本地Vite4185，确定性Mock；桌面Chromium1440×900、手机390×844，另有六档矩阵 |
| 真实DB | 临时PostgreSQL16-alpine、Redis7.4-alpine；专用库`s2v_test_20261006`、随机schema；独立内部网络，无主机发布端口，tmpfs数据 |
| 测试源码摘要 | 296个文件；`d6d01786988fe179c9ec5cc70519ceecdaf6dbd4a3f9b6c7314b0736a9b9f3fd`；不是可部署发布包的签名 |
| 前端本地build的index SHA-256 | `021594a7a606f580fae7790445b34f8e9179a114e928a8c6ee1772142d58e3f3`；只构建，没有上传 |
| 107入口 | `http://192.168.0.107:12000`；只读浏览器验收使用现有重置环境的测试管理员，本文不记录账号秘密 |
| 107进程 | `sub2video-backend` active，PID414725，启动于2026-10-05 21:27:11 CST；测试后PID未变 |
| 107后端SHA-256 | `b8ad0f489bc5b2007898ac6e479ea58cd4088430bb38d3a4e9e8f79b29a8f5d6` |
| 107前端index SHA-256 | `88cdf672df2f85dd5243f3093a471418bf4d7ea1bb6f88ef0de5060d9bd8d4fe` |

源码摘要算法：读取Git已跟踪和未忽略的未跟踪`backend/`、`frontend/`文件，排除`backend/data/`、`backend/server.linux`及`.tmp_`；只纳入go/mod/sum/sql/vue/ts/js/css/json/yaml/yml/html。按路径排序，对每个文件计算原始字节SHA256，将`<小写hash>  <路径>\n`连接后再SHA256。不含文档、截图基线、媒体和业务文件，所以不能当完整发布产物校验。上表数值在测试脚本复核定稿后计算。

## 可复现问题与本轮修改

| 问题 | 复现和实际行为 | 本轮处理 | 验证 |
| --- | --- | --- | --- |
| 创作草稿丢失明确模型（P1） | 保存指定承运商/模型草稿后重新加载；目录尚未返回时watcher清空model；10条新测试初次全失败 | 目录就绪前不按空目录清空选择，保持显式模型；不弱化fixture | 五类场景×桌面/手机，10/10通过；完整回归通过 |
| 无供给被解释为网络故障（P1） | 选项接口UNAVAILABLE_COMBINATION/SUPPLY_UNAVAILABLE/500都曾提示检查网络，并可能重复显示“没有可用能力” | 按公开错误码和有无HTTP响应区分组合、供给、网络、服务错误；抑制重复误导提示；保留输入和重试 | 新可用性合同；旧journey用例改验服务异常而非旧文案 |
| 手机主要按钮太小（P1） | 320/390下管理新增/保存32px，客户端弹窗确认36px | shared `S2VButton`的primary sm/md手机最小44px，桌面仍保留32/36px；不在页面写视觉配方 | UI标准14/14；7张手机基线人工审查后更新；未提高截图容差 |
| 无推荐模型被映射为500（P1） | 真实107目录models=0；合法5秒无扣费估价返回500/INTERNAL | backend公开错误映射补全，返回503/SUPPLY_UNAVAILABLE与可理解文案；真实DB HTTP测试增加无供给断言和账务不变断言 | 本地单元与隔离真实PG通过；107仍旧版本，因此真实UAT保留FAIL |
| 迁移测试写死21个文件 | 当前代码已经有022，CLI测试仍假定21 | 动态读取完整SQL集合并逐项对比名称及SHA256，验证无多余记录；保留两次migrate-only及无seed副作用要求 | 真实PG＋实际CLI通过，当前22个迁移 |
| 能力矩阵共享60秒预算 | SSH逐SQL往返下，18个gate通过，最后fallback提交耗尽父级时间；另有整个store包10分钟总时限耗尽 | 仅大矩阵fixture用3分钟；还原用独立15秒；其他fixture/生产deadline/业务断言不变；最终在专用数据库容器内执行测试以移除隧道延迟 | store全套38个顶层测试通过，无skip |
| 显式UAT入口可能全跳过 | 旧opt-in integration缺变量可全skip退出0 | 新增`test:uat:readonly`严格校验目标、账号和profile；无配置直接失败；区分ready和unconfigured | 缺配置负向门禁如预期退出1；真实UAT拒绝把500视作通过 |
| 真实测试失败产物可能携带凭据 | `fill(password)`和`page.request`失败调用日志、默认error-context页面快照存在记录风险 | 新只读入口用原生表单事件、失败清空、页面内fetch及Token、同源只读守卫，关闭Service Worker和隐式快照；不扩大到旧live套件 | 两种合成故障如预期失败；扫描10份日志/JSON/error-context未检出合成邮箱、密码或Token；带Bearer请求确已由合成服务接收 |

测试自身两处误报也已修正：搜索框文字距外框37px来自边框、padding、搜索图标和gap，轴线应测完整控件外框；滚动后的0.14/0.34px使用规范已有的≤1px子像素舍入。44px主操作、底栏避让和5点命中阈值未放宽。弹窗焦点循环和Escape回焦实际通过，因此没有修改Dialog。

## 测试命令与结果

| 层/命令 | 结果 | 证据边界 |
| --- | --- | --- |
| `pnpm.cmd lint:check` | PASS | ESLint与现有样式规则；不等于全站样式所有权/无障碍审计 |
| `pnpm.cmd typecheck` | PASS | Vue/TS编译检查 |
| `pnpm.cmd build` | PASS；2472 modules，Vite构建23.21s | 官网/客户端/管理端完整前端构建；没有部署 |
| `pnpm.cmd test:visual create-availability.contract.spec.ts` | 修复后10 PASS，9.2s | Mock交互，包含草稿与模型保留、错误恢复、不提交 |
| `pnpm.cmd test:visual ui-standards.contract.spec.ts --project=desktop` | 14 PASS，12.6s | 六视口、主要操作、管理表格、弹窗和移动导航；只读Mock |
| 七个受影响手机surface的定向`--update-snapshots` | 7 PASS，8.5s | 逐张审图后更新，属于基线维护；最终回归另行执行 |
| `pnpm.cmd test:visual`最终完整回归 | **341 PASS / 29 SKIP / 0 FAIL**，2.4m | 370个发现用例，Mock交互＋视觉混合；不是370条独立业务场景 |
| 当前桌面创作/承运商页面额外截图 | 2 PASS，6.5s，并人工查看实际图 | 临时配置`screenshot:on`，只用于审图，未改桌面基线 |
| `go test -count=1 ./...`最终本地回归 | PASS | 该次未启集成；真实数据库必须看下列执行，不能混淆 |
| 首次启用集成的`go test -count=1 -json ./...` | FAIL | store矩阵60s及包10m预算问题；其余包已执行通过；MinIO1项SKIP |
| 当前源码Linux `store.test -test.v -test.count=1 -test.timeout=6m` | **38个顶层PASS，0 FAIL，0 SKIP** | 专用PG/Redis，同容器网络；涵盖财务、路由、归属、导演、恢复等 |
| 当前源码Linux `handler.test -test.v -test.count=1 -test.timeout=6m` | **26个顶层PASS，0 FAIL，0 SKIP** | 含4个真实DB HTTP集成；其余单元，不把26个全称为集成测试 |
| `TestUpgradeFrom019PreservesExistingDataIntegration` | PASS，17.10s | 本轮真实PG，保留历史数据与checksum拒绝 |
| `TestMigrateOnlyCLIHasNoServiceSideEffectsIntegration` | PASS，38.72s | 本轮真实PG＋当前编译的CLI，两次迁移、未启动HTTP/Redis/Worker/seed |
| `TestWorkerSubmissionUncertaintyIntegration` | PASS，56.58s | 本轮真实PG＋Runner＋本地故障上游，**不是真实供应商** |
| `TestRealObjectStorageMediaRoundTripIntegration` | SKIP | 未配置独立MinIO endpoint/access/secret/bucket及媒体fixture；没有借用UAT配置 |
| `pnpm.cmd test:uat:readonly` | FAIL，桌面/手机2条；加固后再次执行结果一致 | 107未部署本轮后端修复；仅500/INTERNAL公开不可用合同断言失败，其他只读检查继续完成 |
| 只读入口故障产物验证 | PASS，2种故障的测试本身预期FAIL | 合成登录点击超时、认证请求主动断链；共10份产物无合成凭据；仅证明本轮新入口和Playwright1.62.1，非全项目凭据审计 |
| `git diff --check` | PASS | 只报既有LF/CRLF转换提醒，无空白错误 |

两个Linux测试二进制均在上传后校验SHA256，再只复制到本轮临时PG容器执行：

- store：`8c4c3b16a8df06a77e92a141eba2b0f13970e4b62ed1d72fed69af2d868d9b7c`
- handler：`f04433434f42638911f6cbb5d3e2d2a9e2b04666e31453215dd39d76118511ab`

这是测试二进制，不是替换UAT服务的后端。完整本地Go单元回归＋最终store/handler真实DB执行，覆盖最终修改；worker、CLI和迁移包未在首次成功后重复跑。没有将首次超时隐藏为一次性全绿。

29个Playwright SKIP全部为显式避免重复矩阵：手机项目跳过12个responsive布局矩阵和14个新UI矩阵；桌面项目跳过auth-forms、journey、studio中3个手机专属最小宽度用例。对应场景均在另一个项目实际执行。没有因供应商或数据库未配置而跳过Mock核心流程。

## 真实数据库与财务证据

| 已覆盖范围 | 当前运行的关键证明 |
| --- | --- |
| 估价 | 正向5秒768P估价0.15，余额100保持；任务、订单、quote和账本均0；无供给返回503也无财务副作用；存储/上游为故障fixture |
| 幂等和原子性 | 顺序/并发请求只创建一次；首次成功后供给下线仍重放原结果；余额不足和触发器写入失败整体回滚 |
| 退款 | Worker真实DB故障测试记录：合成余额100→99.85→100，原扣款0.15，退款1次，订单refunded；这是隔离合成钱包，没有动真实余额 |
| 快照/执行 | 提交受理后本地存ID失败进入待核对，不重提/退款；Worker使用旧快照；提交/轮询租约fencing、执行和存储等待超时通过 |
| 归属与作品 | 下载鉴权、签名接口、并发保存、savedDestination刷新定位；视频URL为合成/签名fixture，不表示真实文件已解码 |
| 导演台 | PG持久化、跨用户拒绝、revision冲突、候选校验、提交绑定与扣款整体原子性通过 |

Worker合成证据示例（稳定脱敏，不是UAT订单）：`tsk_…2dfe` / `ord_…b935`，实扣0.15、退款0.15、退款记录1条、状态refunded。该证据证明错误处理和账务，不代表真实供应商生成。

## 视觉审图记录

已查看当前桌面1440px创作和承运商实际截图，输入/预览分列、标题/搜索/记录轴线正常。手机390px逐张查看概览、任务、工单、承运商、供应商、价格及管理弹窗；44px主要按钮可见，表格维持容器内横滚，弹窗留在视口。更新7张mobile-win32基线，比较了旧/新管理弹窗，增加的高度来自按钮而非主题漂移。

320/390/768/1024/1440/1920均有几何与命中自动化；不能因此宣称每个页面各状态均人工审过。200%浏览器缩放、真实软键盘、iOS/Android、屏幕阅读器、全站对比度、长内容和精确840px容器边界仍未验证。

## 107真实API结果

真实登录、创作目录、钱包、任务和流水读取可执行；目录保留三种模式，但可用模型列表为0。无扣费文生估价参数为5秒、768P、16:9、无自动音频、无指定模型（空目录下请求推荐）。实际返回500/INTERNAL，测试明确失败；没有用虚构模型ID制造可用状态。

只读套件继续收集六档宽度、按钮禁用、业务响应内部字段检查和余额/任务/流水前后比对；这些断言通过，任务/账务未变化。它没有读取数据库订单/quote数量，完整无副作用证据来自独立真实数据库HTTP测试，不能在此偷换。

最终使用加固后的只读脚本再次运行桌面/手机，两个用例仍仅因估价500/INTERNAL失败；lint与typecheck也在脚本定稿后重跑通过。扫描该轮4份产物（运行日志、两个error-context及结果元数据），未发现测试账号、密码或JWT形态令牌，也没有页面快照；该扫描不等于全项目安全审计。当前未执行ready正向真实验收，不能因脚本已实现参数选择/估价/按钮断言就称该路径通过。旧live-*测试尚未完成相同失败日志加固；本轮真实环境推荐入口仅为`test:uat:readonly`。隐式DOM快照禁用依赖当前Playwright1.62行为，升级后需要重新做合成秘密故障验证。

尚未完成：三模式成功估价、真实图片上传、付费生成、真实视频下载/解码与保存、真实失败退款、当前后端修复在107的复验。107没有新后端修复是已知版本差异，不得把本地PASS写成线上PASS。

## 本轮增量文件

既有导演台源码、022迁移和之前文档继续保留，不全部归到本轮修改。

| 文件 | 本轮增量 |
| --- | --- |
| `frontend/src/pages/client/CreatePage.vue` | 可用性错误分类、抑制重复错误、草稿恢复时机；保留原导演台修改 |
| `frontend/src/shared/ui/S2VButton.vue` | primary手机触摸尺寸，复用现有variant |
| `frontend/tests/e2e/create-availability.contract.spec.ts` | 新增5场景×2项目 |
| `frontend/tests/e2e/ui-standards.contract.spec.ts` | 新增14条实际布局/键盘矩阵 |
| `frontend/tests/e2e/journey.contract.spec.ts` | 更新明确服务失败的预期提示 |
| `frontend/tests/e2e/surfaces.visual.spec.ts-snapshots/` | 7张手机基线；既有journey基线不由本轮重写 |
| `frontend/package.json` | 分离交互、布局、截图和严格只读入口 |
| `frontend/playwright.config.ts` | Mock代理禁止误连开发者后端 |
| `frontend/playwright.integration.config.ts` | 真实环境禁用自动retry及认证trace/录像/失败截图 |
| `frontend/playwright.readonly.config.ts`、`frontend/tests/integration/uat-readonly.spec.ts` | 严格目标/profile检查、只读守卫和结果分层 |
| `backend/internal/handler/client.go` | 新增无推荐模型公开错误映射；保留原导演台合同 |
| `backend/internal/handler/client_failure_contract_test.go`、`estimate_integration_test.go` | 公开错误单元/真实PG回归与无副作用断言 |
| `backend/cmd/server/main_integration_test.go` | 动态迁移集合和checksum |
| `backend/internal/store/integration_test_helpers_test.go`、`direct_beta_integration_test.go` | 可显式设置长矩阵预算及独立还原context；保留原隔离增强 |
| `docs/testing/README.md`、`test-plan-2026-10-06.md`、`test-cases-2026-10-06.md`、`ui-layout-acceptance.md`、本文 | 测试入口、流程、用例、规范和实际结果 |

## 数据影响与清理

没有UAT付费任务、供给或价格变更、余额调整、用户资产写入；只登录、读取与estimate。没有修改`backend/data/`、`backend/server.linux`，没有Git提交/推送或部署/reload/restart。

远端本轮资源为`/tmp/s2v-qa-20261006-01/`、`s2v-qa-20261006-01-pg`、`s2v-qa-20261006-01-redis`和同名内部网络。清理前校验精确名称、专属label、无业务端口/挂载。首次被总时限终止的测试留下1个随机schema；最终两套测试前后均为1个，public表0、Redis键0，说明最终测试未新增遗留。随后删除本轮临时容器/tmpfs及暂存二进制，遗留合成schema随临时库清除；独立检查这些资源已不存在，UAT服务仍原PID active。

本地专属SSH端口25439/26389的隧道已按完整命令精确匹配关闭。本轮临时目录`E:\apps\github.com\sbu2video\.tmp_testplan_20261006_01`已删除并通过`Test-Path`确认不存在；删除前核对绝对路径和0个reparse point。移除内容为133个本轮日志、故障trace、临时配置及测试二进制，未使用回收站，可按本文流程重新生成。历史`.tmp_*`目录未清理。收尾端口检查未见4185、4197、25439、26389监听；进程列表补充查询因本地权限拒绝，未将其记为通过。`frontend/dist`是被忽略的构建产物，不提交。长期保留的是本报告文字证据和经审阅的7张源代码视觉基线。

## 剩余风险、是否达到Beta与下一步

1. **107生成路径仍有P1**：无可用供给/存储配置，且旧部署估价错误仍为500。本地修复通过不代表线上已修好。
2. 没有本轮真实供应商交付、真实对象存储往返、可播放下载和完整受控写入证据。
3. Worker进程实际kill/restart、认证/验证码完整生命周期、真机及完整无障碍尚未覆盖，见用例TC-060、TC-064/068、TC-102—106。
4. 当前测试命令和计划可重复执行，但CI流水线尚未接入，112条用例不是112条已全自动/全通过。

未发现本轮新增的已证实P0，不等于所有P0场景已完成真机/真实供应商验证。**现在适合审阅UI和导演准备流程，不适合把107开放成可生成的5～10人Beta。**

一个下一步：将本轮最小修复纳入经审阅的107发布包并另行授权部署，然后先执行严格只读回归；生成试用还须管理员恢复真实供给和存储配置，再按明确预算补实际交付验收。
