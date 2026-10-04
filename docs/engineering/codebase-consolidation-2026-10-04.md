# Sub2Video 代码收敛与导演台准备记录

日期：2026-10-04。分支：`codex/commercial-studio-redesign`。本轮开始时工作区干净，已有代码检查点 `08123b8` 和审查记录提交 `618a252`，因此没有创建空提交。问题基线见 [代码审查](./codebase-review-2026-10-04.md)，该历史记录不反向改写。

## 本轮结果

完成已定位的账号状态、余额、配置事务、旧报价边界和验证入口修复，保留 Vue 3 / Go、现有设计系统、数据库和业务合同。没有整站重写，没有新增导演台服务、模型、供应商、支付或迁移。

代码混乱的主要来源是多轮改造后新旧合同混用、状态所有权不清和验证入口不可信。本轮先处理这些可验证的问题；大页面拆分、旧 CSS 配方、历史不可达页面等维护工作仍未全部完成，不能把这次整理称为全面重构完成或商业上线验收。

## 对上一轮问题的处理

| 基线 | 本轮处理 | 证据和限制 |
| --- | --- | --- |
| R01 草稿跨账号 | 普通草稿按用户 ID 存储；无归属旧草稿不自动导入；原用户重新登录可恢复 | A→B→A、会话过期恢复的桌面/手机 Mock 测试；保留旧 localStorage，未删除用户草稿 |
| R02 私有缓存及迟到响应 | 会话变化统一清空 app 私有状态；HTTP 成功/错误响应和多段异步业务操作检查会话 | 延迟作品响应、跨账号加载失败测试；服务端归属规则未改 |
| R03 配置部分写入 | offering、binding、execution 创建及更新放入同一事务；检查写入数量，提交前读取完整结果 | Go 单测通过；第二/第三次 SQL 失败回滚用例已补，但真实数据库环境不可用，尚未执行 |
| R04 危险历史部署脚本 | `finish-107-backend.sh` 改成 exit 2 的拒绝入口，移除历史业务修改与登录输出 | 本地 Mock 确认拒绝且没有服务/网络动作；没有在 107 执行 |
| R05 类型门禁 | typecheck 显式检查 app/node 配置；build 使用同一命令；禁止配置 emit，移除生成的 `vite.config.js` | 有效类型检查和构建通过；确认 JS 配置未被重新生成 |
| R06 历史配置映射 | legacy SELECT/Scan 补齐五个字段，设置旧 offering ID；仅无记录时进入 fallback | 单测证明数据库故障不触发兼容 fallback；旧记录读取/编辑的 DB 用例尚未执行 |
| R07 空描述误报服务异常 | 稳定领域错误 `ErrPromptRequired` 映射 400 `PROMPT_REQUIRED`；前端明确拒绝时清除 pending | handler 测试覆盖包装错误；浏览器测试确认可修改后新提交，不误入待核对 |
| R08 个人资料假能力 | 展示真实只读资料，移除固定认证/等级、假保存和仅内存通知开关；重置密码进入找回流程 | Profile 桌面/手机视觉与交互检查；未扩展完整账户管理 |
| R09 多份余额与竞态 | auth 为唯一余额来源；读取版本与会话护栏；重叠交易不根据响应顺序猜余额；最后一项操作结束重新读取钱包 | 覆盖旧读取晚到、退款/兑换逆序、第三次失败操作打断对账；对账失败在钱包明确提示并可重试 |
| R10 刷新草稿竞态 | 提交前、pagehide 与组件退出同步持久化；草稿所有者固定到挂载时用户 | 恢复测试明确断言原提示词和模型；只更新预期值变化的两张快照，未提高容差 |

## 职责和合同收敛

- `legacy_quotes.go` 集中历史报价类型、TTL、创建/消费和专用查询。正式 `direct_routing.go` 不再引用 `TaskQuote` / `QuoteInput`；共享素材校验只接收模式和图片数量约束。
- 正式接口保持 estimate + direct submit，旧 quote 接口仍返回 410。没有删除历史报价表、重跑迁移或改变旧任务快照。
- 模式、推荐、成本和配置读取的本轮涉及路径保留数据库错误，不再一律当作候选不匹配。缺少成本规则仍正常排除候选。
- 产品、架构和路由文档统一为模型无关的售价、执行配置成本、先幂等后路由、事务扣费、Worker 快照和提交待核对。修正当前并不存在独立 mode/model 快照列的表述。
- 根 README 和 docs 索引区分当前合同、历史验收和原型。工程 UI 指引统一以当前设计系统为准，不同时要求新旧原型配方。
- 发布 Mock 测试改用每轮独立临时目录，验证路径不逃逸；`.gitattributes` 固定 Bash 脚本 LF。保留现有发布方式，本轮没有发布。

这不是删除兼容数据的授权。后续处理 supplier_routes、offering/执行标识桥接及历史 Worker 分支前，仍要检查实际调用、历史任务与迁移兼容性。

## 修改文件范围

| 范围 | 文件 |
| --- | --- |
| 后端 | `internal/handler/client.go`、`client_failure_contract_test.go`；`internal/store/creation_catalog.go`、`direct_routing.go`、`source_shape_test.go`；新增 `legacy_quotes.go`、`supplier_offering_test.go`、`supplier_offering_integration_test.go` |
| 前端业务 | `stores/auth.ts`、`stores/app.ts`、`api/http.ts`、`shared/lib/create-draft.ts`；Create、Wallet、Works、Profile；Login/Register、TaskDelivery、useStudioOverview、ClientLayout、AppHeader |
| 前端测试 | auth-session、creation、journey、public-home 四份契约测试；Profile 和 pending-creation 各桌面/手机四张快照 |
| 工程入口 | `frontend/package.json`、`tsconfig.node.json`，删除重复 `vite.config.js`；`.gitattributes`；两份工程指引；根 README、docs README；历史部署拒绝入口与部署 Mock 测试 |
| 业务/研究文档 | 产品总览、后台架构、创作路由设计、client-create、client-direct-routing；本文与开源选型记录 |

前后端测试与 UI 修改沿用现有 Vue/Go 和测试技能。UI/UX 指导仅用于真实状态、错误可见性和重试操作，没有建立第二套设计系统。

## 验证记录

本机 Node `v22.16.0`、pnpm `10.32.0`、Go `1.25.1 windows/amd64`、Vite `6.4.3`。Mock 浏览器使用 `VITE_API_PROXY=http://127.0.0.1:1`、`PLAYWRIGHT_REUSE_EXISTING_SERVER=0`，本地 Vite 4185，未将测试请求发往 UAT。

| 层级 | 命令/检查 | 本轮结果 |
| --- | --- | --- |
| 静态 | `pnpm lint:check` | 通过 |
| 类型 | `pnpm typecheck`（显式 app/node） | 通过；源码与 Vite 配置均被检查 |
| 构建 | `pnpm build --outDir ../.tmp_consolidate_20261004_01/frontend-dist-final` | 通过，2,449 modules，15.40 秒；没有覆盖既有 dist 或受保护二进制 |
| Go 单元/合同 | `SUB2VIDEO_RUN_INTEGRATION=0 go test -count=1 ./...` | 通过；含 handler、store、Worker、Adapter 本地故障服务器与迁移单元检查；不代表真实 DB/供应商 |
| 部署脚本 Mock | `bash deploy/tests/update-107.test.sh /e/apps/github.com/sbu2video/.tmp_consolidate_20261004_01/deploy` | 12 项通过；service/network 命令均为 Mock |
| 浏览器定向 Mock | auth-session、creation、experience-delivery 两尺寸 | 最后一轮 100 passed，0 failed，55.6 秒 |
| 浏览器完整 Mock / 视觉 | `pnpm test:visual --reporter=dot --output=../.tmp_consolidate_20261004_01/final-playwright` | 最终 283 passed / 15 skipped / 0 failed，共 298 项，2.4 分钟 |
| 数据库集成入口 | `SUB2VIDEO_RUN_INTEGRATION=1 go test -count=1 -v ./internal/store -run 'TestSupplierOffering.*Integration\|TestDirectRouteDatabaseFailuresAreNotMissingSupplyIntegration'` | 5 个实际场景均 skip：缺少 `SUB2VIDEO_TEST_DATABASE_URL`；父测试显示 PASS 不计为集成通过 |
| 差异 | `git diff --check`，每次提交前 `git diff --cached --check` | 通过 |

测试过程保留以下事实：

1. 初轮定向 124 项中有 5 项失败：3 项测试点击了手机不可见的侧栏链接；2 项 pending 快照依赖防抖草稿时序。分别改用可见入口、同步保存并断言恢复值；没有增大像素容差。
2. 一次完整运行 279 passed / 15 skipped。其后独立审查发现新的余额对账缺口，补修并增加回归，因此又运行最终全套；前一次结果不冒充最终代码验收。
3. 15 个跳过来自尺寸矩阵去重：12 项自定义尺寸只在 desktop 运行，3 项最小宽度只在 mobile 运行。mobile 为 390px、touch 开启的桌面 Chromium，不能当作真实手机系统验收。
4. 主线程审图检查了四张预期更新：Profile 去除假能力；pending 原提示词和模型被正确恢复。正常余额页面没有新视觉配方。
5. 初次 Vite/Go/Git Bash 的进程或缓存被本机沙箱限制；授权本地执行后完成。没有将 EPERM 解释为代码测试通过。

真实环境边界：本机 PATH 未发现 Docker/PostgreSQL/Redis 工具；一次 SSH 只读探测 `192.168.0.107:22` 超时，没有远端命令执行。未获得可用的专用测试数据库和 Redis，因此没有运行真实 SQL 回滚、真实用户/钱包、对象存储、真实下载播放或供应商生成。最小缺失条件是一个确认隔离的 PostgreSQL 测试库和 Redis 测试配置，不能把 UAT 业务库直接交给自动清理逻辑。

## 开源导演台结论

详见 [8 个项目族的固定版本与许可证比较](./open-source-director-options-2026-10-04.md)。阅读了 README/LICENSE/依赖与关键源码，未安装或运行任何第三方工程。

建议采用 Toonflow 的分镜、素材引用、候选结果交互思路，按需复用经许可核对的小模块。它的当前固定版本是 MIT、Vue 3，但没有独立登录鉴权，并使用服务器目录项目和供应商直连入口；整套嵌入会产生第二套账户/资产/执行边界。

NarratoAI、MoneyPrinterTurbo 是宣传/解说与后期候选，不代替镜头管理。OpenCut 当前重写、classic 已归档，暂作观察。Huobao 当前许可证含非商业限制；OpenVideo 有自定义条款；Storyboarder 的商业复用授权没有澄清，本轮不引入。

第一期只考虑三镜头工作区：镜头列表 → 参考素材/参数 → 现有估价与幂等提交 → 候选成片 → 选片与顺序。镜头编辑、生成任务、消费订单、退款和合成状态分开，继续复用当前用户、钱包、Worker、任务快照和资产归属，不公开供应商配置。

## 未完成与下一步

| 未完成项 | 是否影响当前判断 |
| --- | --- |
| 新配置事务、旧配置映射的真实数据库回归 | 阻止把这次修改直接宣称为已验收的发布版；上线前必须补齐 |
| 目录/推荐/选项/路由筛选 SQL 全面合并、退款 helper 统一 | 后续按实际问题分包处理；本轮只修涉及路径，未引入通用引擎 |
| CreatePage/供应商页面进一步拆 composable、旧占位方法和不可达页面清理 | 维护债仍在；需逐个证明调用边界再删除，不能一口气删目录 |
| 历史 CSS 配方归位、教程图片授权清单、CI | 尚未完成，不因本地全绿推断已有自动门禁或素材商用审计 |
| 导演台原型、镜头持久化、渲染导出 | 本轮只有研究，未安装集成，也未创建新表或导航 |
| 当前修改的 107/UAT 实机体验 | 未部署、未验收；历史交付记录不替代本轮版本 |

一个下一步：先完成隔离数据库回归，作为本轮修复的验收门槛；导演台可在此期间审阅三镜头可点击原型，确认信息关系后再决定具体复用模块。

## Git、数据与清理

- 后端修复提交：`414c6ac`；前端状态与验证入口提交：`4c3dc24`。工程/文档与选型由主线程单独提交，其哈希以 Git 历史及最终交付为准。
- 未推送、未部署、未重启 107，未修改供给、售价、余额或用户资产。没有数据库迁移。
- `backend/data/`、`backend/server.linux`、旧临时目录保持不变；不执行 reset/checkout/clean。
- 本轮唯一临时根：`E:\apps\github.com\sbu2video\.tmp_consolidate_20261004_01`，仅本地构建、Mock fixture、截图差异与日志，不纳入 Git。测试进程结束后核对准确路径与无重解析点，已清理 378 个文件（约 17.5 MB），`Test-Path` 确认目录不存在。诊断结论保留在本文；这些产物可重新运行测试/构建生成，旧临时目录未清理。
